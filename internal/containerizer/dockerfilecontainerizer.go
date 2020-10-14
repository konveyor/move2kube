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

	log "github.com/sirupsen/logrus"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

const (
	dockerfileDetectscript string = types.AppNameShort + "dfdetect.sh"
)

// DockerfileContainerizer implements Containerizer interface
type DockerfileContainerizer struct {
	dfcontainerizers []string //Paths to directories containing containerizers
}

// Init initializes docker file containerizer
func (d *DockerfileContainerizer) Init(path string) {
	var files, err = common.GetFilesByName(path, []string{dockerfileDetectscript})
	if err != nil {
		log.Warnf("Unable to fetch files to recognize docker detect scripts : %s", err)
	}
	for _, file := range files {
		fpath := filepath.Dir(file)
		d.dfcontainerizers = append(d.dfcontainerizers, fpath)
	}
	log.Debugf("Detected dockerfile containerization options : %s ", d.dfcontainerizers)
}

// GetTargetOptions returns the target options for a path
func (d *DockerfileContainerizer) GetTargetOptions(plan plantypes.Plan, path string) []string {
	targetOptions := []string{}
	for _, dfcontainerizer := range d.dfcontainerizers {
		abspath, err := filepath.Abs(path)
		if err != nil {
			log.Errorf("Unable to resolve full path of directory %s : %s", abspath, err)
		}
		outputStr, err := d.detect(dfcontainerizer, abspath)
		log.Debugf("Detect output of %s : %s", dfcontainerizer, outputStr)
		if err != nil {
			log.Debugf("%s detector cannot containerize %s : %s", dfcontainerizer, path, err)
			continue
		} else {
			if path != common.AssetsPath {
				dfcontainerizer, _ = plan.GetRelativePath(dfcontainerizer)
			}
			targetOptions = append(targetOptions, dfcontainerizer)
		}
	}
	return targetOptions
}

func (d *DockerfileContainerizer) detect(scriptpath string, directory string) (string, error) {
	cmd := exec.Command("/bin/sh", dockerfileDetectscript, directory)
	cmd.Dir = scriptpath
	log.Debugf("Executing detect script %s on %s : %s", scriptpath, directory, cmd)
	outputbytes, err := cmd.Output()
	return string(outputbytes), err
}

// GetContainerBuildStrategy returns the ContaierBuildStrategy
func (d *DockerfileContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.DockerFileContainerBuildTypeValue
}

// GetContainer returns the container for a service
func (d *DockerfileContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	// TODO: Fix exposed ports too
	if service.ContainerBuildType != d.GetContainerBuildStrategy() || len(service.ContainerizationTargetOptions) == 0 {
		return irtypes.Container{}, fmt.Errorf("Unsupported service type for Containerization or insufficient information in service")
	}
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)
	container.RepoInfo = service.RepoInfo
	dfdirectory := plan.GetFullPath(service.ContainerizationTargetOptions[0])
	content, err := ioutil.ReadFile(filepath.Join(dfdirectory, "Dockerfile"))
	dockerfilename := "Dockerfile." + service.ServiceName
	if err != nil {
		log.Errorf("Unable to read docker file at %s : %s", dfdirectory, err)
		return container, err
	}
	dockerfilestring := string(content)
	abspath, err := filepath.Abs(plan.GetFullPath(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0]))
	if err != nil {
		log.Errorf("Unable to resolve full path of directory %q Error: %q", abspath, err)
		return container, err
	}

	outputStr, err := d.detect(dfdirectory, abspath)
	if err != nil {
		log.Errorf("Detect failed : %s", err)
		return container, err
	}
	if outputStr != "" {
		m := map[string]interface{}{}
		if err := json.Unmarshal([]byte(outputStr), &m); err != nil {
			log.Errorf("Unable to unmarshal the output of the detect script at path %q. Output: %q Error: %q", dfdirectory, outputStr, err)
			return container, err
		}

		if value, present := m["Port"]; present {
			portToExpose := int(value.(float64))
			container.ExposedPorts = append(container.ExposedPorts, portToExpose)
		}

		dockerfilestring, err = common.GetStringFromTemplate(string(content), m)
		if err != nil {
			log.Warnf("Template conversion failed : %s", err)
		}
	}
	//log.Debugf("Creating Dockerfile at %s with %s", filepath.Join(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0], service.ServiceName+"Dockerfile"), dockerfilestring)
	dockerfilePath := filepath.Join(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0], dockerfilename)
	container.AddFile(dockerfilePath, dockerfilestring)
	dockerbuildscript, err := common.GetStringFromTemplate(scripts.Dockerbuild_sh, struct {
		Dockerfilename string
		ImageName      string
		Context        string
	}{
		Dockerfilename: dockerfilename,
		ImageName:      service.Image,
		Context:        ".",
	})
	if err != nil {
		log.Warnf("Unable to translate template to string : %s", scripts.Dockerbuild_sh)
	} else {
		container.AddFile(filepath.Join(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0], service.ServiceName+"dockerbuild.sh"), dockerbuildscript)
		container.RepoInfo.TargetPath = dockerfilePath
	}
	err = filepath.Walk(dfdirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %s due to error: %s", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		filename := filepath.Base(path)
		if filename == "Dockerfile" || filename == dockerfileDetectscript {
			return nil
		}
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal(err)
		}
		//TODO: Should we allow subdirectories?
		container.AddFile(filepath.Join(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0], filename), string(content))
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files due to : %s", err)
	}

	return container, nil
}
