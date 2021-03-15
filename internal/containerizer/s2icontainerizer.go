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
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

const (
	s2iDetectScript string = types.AppNameShort + "s2idetect.sh"
)

// S2IContainerizer implements Containerizer interface
type S2IContainerizer struct {
	s2icontainerizers []string //Paths to directories containing containerizers
}

// GetContainerBuildStrategy returns the containerization build strategy
func (d *S2IContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.S2IContainerBuildTypeValue
}

// Init initializes the containerizer
func (d *S2IContainerizer) Init(path string) {
	files, err := common.GetFilesByName(path, []string{s2iDetectScript})
	if err != nil {
		log.Warnf("Unable to fetch files to recognize s2i detect files : %s", err)
	}
	for _, file := range files {
		d.s2icontainerizers = append(d.s2icontainerizers, filepath.Dir(file))
	}
	log.Debugf("Detected S2I containerization options : %s ", d.s2icontainerizers)
}

// GetTargetOptions returns the target options for a path
func (d *S2IContainerizer) GetTargetOptions(_ plantypes.Plan, path string) []string {
	targetOptions := []string{}
	for _, s2icontainerizer := range d.s2icontainerizers {
		output, err := d.detect(s2icontainerizer, path)
		if err != nil {
			log.Debugf("%s detector cannot containerize %s Error: %q", s2icontainerizer, path, err)
			continue
		}
		log.Debugf("Output of S2I containerizer detect script %s : %s", s2icontainerizer, output)
		targetOptions = append(targetOptions, s2icontainerizer)
	}
	return targetOptions
}

func (*S2IContainerizer) detect(scriptDir string, directory string) (string, error) {
	scriptPath := filepath.Join(scriptDir, s2iDetectScript)
	cmd := exec.Command(scriptPath, directory)
	cmd.Dir = scriptDir
	cmd.Stderr = os.Stderr
	log.Debugf("Executing detect script %s on %s : %s", scriptDir, directory, cmd)
	outputBytes, err := cmd.Output()
	return string(outputBytes), err
}

// GetContainer returns the container for a service
func (d *S2IContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	if service.ContainerBuildType != d.GetContainerBuildStrategy() || len(service.ContainerizationTargetOptions) == 0 {
		return irtypes.Container{}, fmt.Errorf("Unsupported service type for Containerization or insufficient information in service")
	}
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)
	container.RepoInfo = service.RepoInfo // TODO: instead of passing this in from plan phase, we should gather git info here itself.
	containerizerDir := service.ContainerizationTargetOptions[0]
	sourceCodeDir := service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0] // TODO: what about the other source artifacts?

	// Create the s2i build script.
	output, err := d.detect(containerizerDir, sourceCodeDir)
	if err != nil {
		log.Errorf("Detect using S2I containerizer at path %q on the source code at path %q failed. Error: %q", containerizerDir, sourceCodeDir, output)
		return container, err
	}
	log.Debugf("The S2I containerizer at path %q produced the following output: %q", containerizerDir, output)

	output = strings.TrimSpace(output)

	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		log.Errorf("Unable to unmarshal the output of the detect script at path %q Output: %q Error: %q", containerizerDir, output, err)
		return container, err
	}

	if value, ok := m[containerizerJSONPort]; ok {
		portToExpose := int(value.(float64)) // json numbers are float64
		container.AddExposedPort(portToExpose)
	}

	m[containerizerJSONImageName] = service.Image
	s2iBuildScript, err := common.GetStringFromTemplate(scripts.S2IBuilder_sh, struct {
		Builder   string
		ImageName string
	}{
		Builder:   m[containerizerJSONBuilder].(string),
		ImageName: m[containerizerJSONImageName].(string),
	})
	if err != nil {
		log.Errorf("Unable to translate the template %q to string. Error: %q", scripts.S2IBuilder_sh, err)
		return container, err
	}

	relOutputPath, err := filepath.Rel(plan.Spec.Inputs.RootDir, sourceCodeDir)
	if err != nil {
		log.Errorf("Failed to make the source code directory %q relative to the root directory %q Error: %q", sourceCodeDir, plan.Spec.Inputs.RootDir, err)
		return container, err
	}
	container.AddFile(filepath.Join(relOutputPath, service.ServiceName+"-s2i-build.sh"), s2iBuildScript)

	// Add any other files that are in the containerizer directory.
	err = filepath.Walk(containerizerDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %s due to error. Error: %q", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		filename := filepath.Base(path)
		if filename == s2iDetectScript {
			return nil
		}

		relFilePath, err := filepath.Rel(containerizerDir, path)
		if err != nil {
			log.Errorf("Skipping path %q . Failed to get the relative path. Error: %q", path, err)
			return nil
		}

		templateBytes, err := ioutil.ReadFile(path)
		if err != nil {
			log.Errorf("Skipping path %q . Failed to read the template. Error: %q", path, err)
			return nil
		}

		contents, err := common.GetStringFromTemplate(string(templateBytes), m)
		if err != nil {
			log.Errorf("Skipping path %q . Unable to translate the template to string. Error %q", path, err)
			return nil
		}

		//Allowing sub-directories
		container.AddFile(filepath.Join(relOutputPath, relFilePath), contents)
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files at path %q Error: %q", containerizerDir, err)
	}

	return container, nil
}
