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
	sourceCodeDir := service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0] // TODO: what about the other source artifacts?

	relOutputPath, err := filepath.Rel(plan.Spec.Inputs.RootDir, sourceCodeDir)
	if err != nil {
		log.Errorf("Failed to make the source code directory %q relative to the root directory %q Error: %q", sourceCodeDir, plan.Spec.Inputs.RootDir, err)
		return container, err
	}

	//　1. Execute detect to obtain the json response from the .sh file
	output, err := d.detect(containerizerDir, sourceCodeDir)
	if err != nil {
		log.Errorf("Detect using Dockerfile containerizer at path %q on the source code at path %q failed. Error: %q", containerizerDir, sourceCodeDir, err)
		return container, err
	}
	log.Debugf("The Dockerfile containerizer at path %q produced the following output: %q", containerizerDir, output)

	// 2. Parse json output
	if output == "" {
		log.Warnf("Output variable is empty")
		return container, nil
	}

	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		log.Errorf("Unable to unmarshal the output of the detect script at path %q Output: %q Error: %q", containerizerDir, output, err)
		return container, err
	}

	// Final multiline string containing the generated Dockerfile will be stored here
	dockerfileContents := ""
	// Filled segments will be stored here
	segmentSlice := []string{}

	// 2.1 Obtain segmentRecords slice for both cases (segment-based, df)
	segmentRecords := []map[string]interface{}{}

	if _, ok := m["type"]; ok {
		// segment-based
		if _, segmentsFound := m["segments"]; segmentsFound {
			if segments, ok := m["segments"].([]interface{}); ok {
				for _, segment := range segments {
					if rec, ok := segment.(map[string]interface{}); ok {
						segmentRecords = append(segmentRecords, rec)
					}
				}
			}
		}

	} else {
		// Add the fixed segment id associated to the full Dockerfile and append to segmentRecords
		m["segment_id"] = "Dockerfile"
		segmentRecords = append(segmentRecords, m)
	}

	// 2.2 Iterate over segmentRecords slice
	for _, segmentRecord := range segmentRecords {

		// is "segment_id" present ?
		if val, ok := segmentRecord["segment_id"]; ok {

			strPath, ok := val.(string)

			if !ok {
				log.Warnf("Segment id is not a valid string: %v", val)
				continue
			}

			// Get the path to the segment and read it as a template
			dockerfileSegmentTemplatePath := filepath.Join(containerizerDir, strPath)
			dockerfileSegmentTemplateBytes, err := ioutil.ReadFile(dockerfileSegmentTemplatePath)
			if err != nil {
				log.Errorf("Unable to read the Dockerfile segment template at path %q Error: %q", dockerfileSegmentTemplatePath, err)
				//return container, err
			}
			dockerfileSegmentTemplate := string(dockerfileSegmentTemplateBytes)
			dockerfileSegmentContents := dockerfileSegmentTemplate

			// Fill the segment template with the corresponding data
			dockerfileSegmentContents, err = common.GetStringFromTemplate(dockerfileSegmentTemplate, segmentRecord)
			if err != nil {
				log.Warnf("Template conversion failed : %s", err)
			}

			// Append filled segment template to segmentSlice
			segmentSlice = append(segmentSlice, dockerfileSegmentContents)
		}

		// is "port" present ?
		if val, ok := segmentRecord["port"]; ok {
			portToExpose := int(val.(float64)) // Type assert to float64 because json numbers are floats.
			container.AddExposedPort(portToExpose)
		}

		// is "files_to_copy" present ?
		if pathsToCopySlice, ok := segmentRecord["files_to_copy"]; ok {

			log.Debugf("listing files to copy:")

			// Iterate over the paths
			for _, pathToCopy := range pathsToCopySlice.([]interface{}) {

				// Check if path is a valid string
				pathToCopyStr, ok := pathToCopy.(string)
				if !ok {
					log.Warnf("pathToCopy is not a valid string: %v", pathToCopy)
					continue
				}
				// Generate the absolute path
				pathToCopyStr = filepath.Join(containerizerDir, pathToCopyStr)

				log.Debugf(pathToCopyStr)

				// Get path info to determine if it is file/dir
				fileInfo, err := os.Stat(pathToCopyStr)
				if err != nil {
					log.Warnf("Cannot determine if the path is a file or folder: %v", pathToCopy)
					continue
				}

				filename := filepath.Base(pathToCopyStr)

				if fileInfo.IsDir() {
					log.Debugf("it is a directory")

					err = filepath.Walk(pathToCopyStr, func(path string, info os.FileInfo, err error) error {
						if err != nil {
							log.Warnf("Skipping path %s due to error. Error: %q", path, err)
							return nil
						}

						if info.IsDir() {
							return nil
						}
						// At this point it means we have a file
						log.Debugf(path)

						filename := filepath.Base(path)

						if filename == "Dockerfile" || filename == dockerfileDetectScript {
							return nil
						}

						// Obtain the relative path of the file wrt the containerizerDir
						relFilePath, err := filepath.Rel(containerizerDir, path)
						if err != nil {
							log.Fatalf("Failed to obtain relative path to file")
						}
						log.Debugf(relFilePath)

						// Get file contents
						contentBytes, err := ioutil.ReadFile(path)
						if err != nil {
							log.Fatalf("Failed to read the file at path %q Error: %q", path, err)
						}

						// Add file contents and relative path to the container object
						container.AddFile(filepath.Join(relOutputPath, relFilePath), string(contentBytes))
						return nil
					})
					if err != nil {
						log.Warnf("Error in walking through files at path %q Error: %q", containerizerDir, err)
					}

				} else {
					log.Debugf("it is a file")
					// Get content and add it to the container object
					contentBytes, err := ioutil.ReadFile(pathToCopyStr)
					if err != nil {
						log.Fatalf("Failed to read the file at path %q Error: %q", pathToCopy, err)
					}
					container.AddFile(filepath.Join(relOutputPath, filename), string(contentBytes))
				}
			}
		}
	}

	// 3. Merge the filled segments into dockerfileContents
	dockerfileContents = strings.Join(segmentSlice, "\n")

	// 4. Add result to the container object
	dockerfileName := "Dockerfile." + service.ServiceName
	dockerfilePath := filepath.Join(relOutputPath, dockerfileName)
	container.AddFile(dockerfilePath, dockerfileContents)

	// 5. Create the docker build script.
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

	return container, nil
}
