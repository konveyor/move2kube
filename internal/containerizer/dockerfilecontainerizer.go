/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package containerizer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

const (
	dockerfileDetectScript string = types.AppNameShort + "dfdetect.sh"
)

// DockerfileContainerizer implements Containerizer interface
type DockerfileContainerizer struct {
	dfcontainerizers []string //Paths to directories containing containerizers
}

// GetContainerBuildStrategy returns the ContaierBuildStrategy
func (*DockerfileContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.DockerFileContainerBuildTypeValue
}

// Init initializes docker file containerizer
func (d *DockerfileContainerizer) Init(path string) {
	files, err := common.GetFilesByName(path, []string{dockerfileDetectScript})
	if err != nil {
		log.Warnf("Unable to fetch files to recognize docker detect scripts : %s", err)
	}
	for _, file := range files {
		d.dfcontainerizers = append(d.dfcontainerizers, filepath.Dir(file))
	}
	log.Debugf("Detected Dockerfile containerization options : %v", d.dfcontainerizers)
}

// GetTargetOptions returns the target options for a path
func (d *DockerfileContainerizer) GetTargetOptions(_ plantypes.Plan, path string) []string {
	targetOptions := []string{}
	for _, dfcontainerizer := range d.dfcontainerizers {
		output, err := d.detect(dfcontainerizer, path)
		if err != nil {
			log.Debugf("%s detector cannot containerize %s Error: %q", dfcontainerizer, path, err)
			continue
		}
		log.Debugf("Output of Dockerfile containerizer detect script %s : %s", dfcontainerizer, output)
		targetOptions = append(targetOptions, dfcontainerizer)
	}
	return targetOptions
}

func (*DockerfileContainerizer) detect(scriptDir string, directory string) (string, error) {
	cmd := exec.Command("/bin/sh", dockerfileDetectScript, directory)
	cmd.Dir = scriptDir
	cmd.Stderr = os.Stderr
	log.Debugf("Executing detect script %s on %s : %s", scriptDir, directory, cmd)
	outputBytes, err := cmd.Output()
	return string(outputBytes), err
}

// GetContainer returns the container for a service
func (d *DockerfileContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	// TODO: Fix exposed ports too
	if service.ContainerBuildType != d.GetContainerBuildStrategy() || len(service.ContainerizationTargetOptions) == 0 {
		return irtypes.Container{}, fmt.Errorf("Unsupported service type for containerization or insufficient information in service")
	}
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)
	container.RepoInfo = service.RepoInfo // TODO: instead of passing this in from plan phase, we should gather git info here itself.
	containerizerDir := service.ContainerizationTargetOptions[0]

	// Create the Dockerfile.
	dockerfileTemplatePath := filepath.Join(containerizerDir, "Dockerfile")
	dockerfileTemplateBytes, err := ioutil.ReadFile(dockerfileTemplatePath)
	if err != nil {
		log.Errorf("Unable to read the Dockerfile template at path %q Error: %q", dockerfileTemplatePath, err)
		return container, err
	}
	dockerfileTemplate := string(dockerfileTemplateBytes)
	sourceCodeDir := service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0] // TODO: what about the other source artifacts?

	output, err := d.detect(containerizerDir, sourceCodeDir)
	if err != nil {
		log.Errorf("Detect using Dockerfile containerizer at path %q on the source code at path %q failed. Error: %q", containerizerDir, sourceCodeDir, err)
		return container, err
	}
	log.Debugf("The Dockerfile containerizer at path %q produced the following output: %q", containerizerDir, output)

	dockerfileContents := dockerfileTemplate
	if output != "" {
		m := map[string]interface{}{}
		if err := json.Unmarshal([]byte(output), &m); err != nil {
			log.Errorf("Unable to unmarshal the output of the detect script at path %q Output: %q Error: %q", containerizerDir, output, err)
			return container, err
		}

		if value, ok := m[containerizerJSONPort]; ok {
			portToExpose := int(value.(float64)) // Type assert to float64 because json numbers are floats.
			container.ExposedPorts = append(container.ExposedPorts, portToExpose)
		}

		dockerfileContents, err = common.GetStringFromTemplate(dockerfileTemplate, m)
		if err != nil {
			log.Warnf("Template conversion failed : %s", err)
		}
	}

	relOutputPath, err := filepath.Rel(plan.Spec.Inputs.RootDir, sourceCodeDir)
	if err != nil {
		log.Errorf("Failed to make the source code directory %q relative to the root directory %q Error: %q", sourceCodeDir, plan.Spec.Inputs.RootDir, err)
		return container, err
	}
	dockerfileName := "Dockerfile." + service.ServiceName
	dockerfilePath := filepath.Join(relOutputPath, dockerfileName)
	container.AddFile(dockerfilePath, dockerfileContents)

	// Create the docker build script.
	dockerBuildScriptContents, err := common.GetStringFromTemplate(scripts.Dockerbuild_sh, struct {
		Dockerfilename string
		ImageName      string
		Context        string
	}{
		Dockerfilename: dockerfileName,
		ImageName:      service.Image,
		Context:        ".",
	})
	if err != nil {
		log.Errorf("Unable to translate Dockerfile template %s to string Error: %q", scripts.Dockerbuild_sh, err)
	} else {
		dockerBuildScriptPath := filepath.Join(relOutputPath, service.ServiceName+"-docker-build.sh")
		container.AddFile(dockerBuildScriptPath, dockerBuildScriptContents)
		container.RepoInfo.TargetPath = dockerfilePath
	}

	// Add any other files that are in the containerizer directory.
	err = filepath.Walk(containerizerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %s due to error. Error: %q", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		filename := filepath.Base(path)
		if filename == "Dockerfile" || filename == dockerfileDetectScript {
			return nil
		}
		contentBytes, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Failed to read the file at path %q Error: %q", path, err)
		}
		//TODO: Should we allow subdirectories?
		container.AddFile(filepath.Join(relOutputPath, filename), string(contentBytes))
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files at path %q Error: %q", containerizerDir, err)
	}

	return container, nil
}
