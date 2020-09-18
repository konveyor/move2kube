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

package collector

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	sourcetypes "github.com/konveyor/move2kube/internal/collector/sourcetypes"
	common "github.com/konveyor/move2kube/internal/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
)

//ImagesCollector collects the docker images
type ImagesCollector struct {
}

// GetAnnotations returns annotations on which this collector should be invoked
func (c ImagesCollector) GetAnnotations() []string {
	annotations := []string{"k8s", "dockerswarm", "dockercompose"}
	return annotations
}

//Collect gets the image metadata using docker inspect
func (c *ImagesCollector) Collect(inputDirectory string, outputPath string) error {
	//Creating the output sub-directory if it does not exist
	outputPath = filepath.Join(outputPath, "images")
	err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create output directory %s : %s", outputPath, err)
		return err
	}
	imageNames, err := getImageNames(inputDirectory)
	if err != nil {
		return err
	}
	log.Debugf("Images : %s", imageNames)
	for _, imageName := range imageNames {
		imagedata, err := getDockerInspectResult(imageName)
		if err != nil {
			continue
		}
		if imagedata != nil {
			imageInfo := getImageInfo(imagedata)
			shortesttag := ""
			for _, tag := range imageInfo.Spec.Tags {
				if shortesttag == "" {
					shortesttag = tag
				} else {
					if len(shortesttag) > len(tag) {
						shortesttag = tag
					}
				}
			}
			imagefile := filepath.Join(outputPath, common.NormalizeForFilename(shortesttag)+".yaml")
			err := common.WriteYaml(imagefile, imageInfo)
			log.Errorf("Unable to write file %s : %s", imagefile, err)
		}
	}

	return nil
}

func getDockerInspectResult(imageName string) ([]byte, error) {
	cmd := exec.Command("docker", "inspect", imageName)
	jsonOutput, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(jsonOutput), "permission denied") {
			log.Warnf("Error while running docker-inspect due to lack of permissions")
			log.Warnf("Please refer to [https://docs.docker.com/engine/install/linux-postinstall/] to fix this issue")
		} else if strings.Contains(string(jsonOutput), "No such object") {
			log.Warnf("Image [%s] not available in local image repo. Run \"docker pull %s\"", imageName, imageName)
			return nil, nil
		} else {
			log.Warnf("Error while running docker-inspect: %s", err)
		}
		return nil, err
	}
	return jsonOutput, nil
}

func getImageInfo(data []byte) collecttypes.ImageInfo {
	imageInfo := collecttypes.NewImageInfo()
	imgLayerInfo := make([]sourcetypes.DockerImage, 0)
	err := json.Unmarshal(data, &imgLayerInfo)
	if err != nil {
		log.Errorf("Unable to unmarshal image info : %s", err)
	}
	for _, image := range imgLayerInfo {
		imageInfo.Spec.Tags = image.RepoTags
		imageInfo.Spec.UserID, err = strconv.Atoi(image.CConfig.User)
		if err != nil {
			log.Debugf("UserID not available in image metadata for [%s]", image.RepoTags[0])
			imageInfo.Spec.UserID = -1
		}
		imageInfo.Spec.AccessedDirs = append(imageInfo.Spec.AccessedDirs, image.CConfig.WorkingDir)
		for key := range image.CConfig.EPorts {
			regex := regexp.MustCompile("[0-9]+")
			portNumber, err := strconv.Atoi(string(regex.FindAll([]byte(key), -1)[0]))
			if err != nil {
				log.Debugf("PortNumber not available in image metadata for [%s]", image.RepoTags[0])
			} else {
				imageInfo.Spec.PortsToExpose = append(imageInfo.Spec.PortsToExpose, portNumber)
			}
		}
	}
	return imageInfo
}

func getImageNames(inputPath string) ([]string, error) {
	if inputPath == "" {
		return getAllImageNames()
	}
	return getDCImageNames(inputPath)
}

func getAllImageNames() ([]string, error) {
	cmd := exec.Command("bash", "-c", "docker image list --format '{{.Repository}}:{{.Tag}}'")
	outputStr, err := cmd.Output()
	if err != nil {
		log.Warnf("Error while running docker image list : %s", err)
		return nil, err
	}
	images := strings.Split(string(outputStr), "\n")
	cleanimages := []string{}
	for _, image := range images {
		if strings.HasPrefix(image, "<none>") || strings.HasSuffix(image, "<none>") {
			log.Debugf("Ignore image with <none> : %s", image)
			continue
		}
	}
	return cleanimages, err
}

func getDCImageNames(directorypath string) ([]string, error) {
	var imageNames []string
	files, err := common.GetFilesByExt(directorypath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize Docker image yamls : %s", err)
	}
	for _, path := range files {
		dc := new(sourcetypes.DockerCompose)
		if common.ReadYaml(path, &dc) == nil {
			for _, dcservice := range dc.DCServices {
				imageNames = append(imageNames, dcservice.Image)
			}
		}
	}
	return imageNames, nil
}
