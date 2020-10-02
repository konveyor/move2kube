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

	log "github.com/sirupsen/logrus"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

const (
	s2iDetectscript string = types.AppNameShort + "s2idetect.sh"
)

// S2IContainerizer implements Containerizer interface
type S2IContainerizer struct {
	s2icontainerizers []string //Paths to directories containing containerizers
}

// Init initializes the containerizer
func (d *S2IContainerizer) Init(path string) {
	var files, err = common.GetFilesByName(path, []string{s2iDetectscript})
	if err != nil {
		log.Warnf("Unable to fetch files to recognize s2i detect files : %s", err)
	}
	for _, file := range files {
		fpath := filepath.Dir(file)
		d.s2icontainerizers = append(d.s2icontainerizers, fpath)
	}
	log.Debugf("Detected S2I containerization options : %s ", d.s2icontainerizers)
}

// GetTargetOptions returns the target options for a path
func (d *S2IContainerizer) GetTargetOptions(plan plantypes.Plan, path string) []string {
	abspath, err := filepath.Abs(path)
	if err != nil {
		log.Errorf("Unable to resolve path %q Error: %q", abspath, err)
		return nil
	}
	targetOptions := []string{}
	for _, s2icontainerizer := range d.s2icontainerizers {
		outputStr, err := d.detect(s2icontainerizer, abspath)
		log.Debugf("Detect output of %q : %q", s2icontainerizer, outputStr)
		if err != nil {
			log.Debugf("%q detector cannot containerize %q Error: %q", s2icontainerizer, path, err)
			continue
		}
		if path != common.AssetsPath {
			s2icontainerizer, _ = plan.GetRelativePath(s2icontainerizer)
		}
		targetOptions = append(targetOptions, s2icontainerizer)
	}
	return targetOptions
}

func (d *S2IContainerizer) detect(scriptpath string, directory string) (string, error) {
	cmd := exec.Command("/bin/sh", s2iDetectscript, directory)
	cmd.Dir = scriptpath
	log.Debugf("Executing detect script %q on %q : %q", scriptpath, directory, cmd)
	outputbytes, err := cmd.Output()
	return string(outputbytes), err
}

// GetContainerBuildStrategy returns the containerization build strategy
func (d *S2IContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.S2IContainerBuildTypeValue
}

// GetContainer returns the container for a service
func (d *S2IContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	if service.ContainerBuildType != d.GetContainerBuildStrategy() || len(service.ContainerizationTargetOptions) == 0 {
		return irtypes.Container{}, fmt.Errorf("Unsupported service type for Containerization or insufficient information in service")
	}

	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)
	dfdirectory := plan.GetFullPath(service.ContainerizationTargetOptions[0])
	abspath, err := filepath.Abs(plan.GetFullPath(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0]))
	if err != nil {
		log.Errorf("Unable to resolve full path of directory %s : %s", abspath, err)
		return container, err
	}
	outputStr, err := d.detect(dfdirectory, abspath)
	if err != nil {
		log.Errorf("Detect failed for S2I (%s) : %s (%s)", dfdirectory, err, outputStr)
		return container, err
	}
	outputStr = strings.TrimSpace(outputStr)

	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(outputStr), &m); err != nil {
		log.Errorf("Unable to unmarshal the output of the detect script at path %q. Output: %q Error: %q", dfdirectory, outputStr, err)
		return container, err
	}

	if value, ok := m["Port"]; ok {
		portToExpose := int(value.(float64)) // json numbers are float64
		container.ExposedPorts = append(container.ExposedPorts, portToExpose)
	}

	m["ImageName"] = service.Image
	s2ibuildscript, err := common.GetStringFromTemplate(scripts.S2IBuilder_sh, m)
	if err != nil {
		log.Errorf("Unable to translate the template %q to string. Error: %q", scripts.S2IBuilder_sh, err)
		return container, err
	}

	container.AddFile(filepath.Join(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0], service.ServiceName+"s2ibuild.sh"), s2ibuildscript)

	var relFilePath string
	err = filepath.Walk(dfdirectory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %s due to error: %s", path, err)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		filename := filepath.Base(path)
		if filename == s2iDetectscript {
			return nil
		}

		relFilePath, err = filepath.Rel(dfdirectory, path)
		if err != nil {
			log.Errorf("Skipping path %q . Failed to get the relative path. Error: %q", path, err)
			return nil
		}

		tmpl, err := ioutil.ReadFile(path)
		if err != nil {
			log.Errorf("Skipping path %q . Failed to read the template. Error: %q", path, err)
			return nil
		}

		contentStr, err := common.GetStringFromTemplate(string(tmpl), m)
		if err != nil {
			log.Errorf("Skipping path %q . Unable to translate the template to string. Error %q", path, err)
			return nil
		}

		//Allowing sub-directories
		container.AddFile(filepath.Join(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0], relFilePath), contentStr)
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files due to : %s", err)
	}

	return container, nil
}
