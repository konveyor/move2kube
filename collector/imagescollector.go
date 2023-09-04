/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package collector

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	sourcetypes "github.com/konveyor/move2kube/collector/sourcetypes"
	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
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
		logrus.Errorf("Unable to create output directory %s : %s", outputPath, err)
		return err
	}
	imageNames, err := getImageNames(inputDirectory)
	if err != nil {
		return err
	}
	logrus.Debugf("Images : %s", imageNames)
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
			if err != nil {
				logrus.Errorf("Unable to write file %s : %s", imagefile, err)
			}
		}
	}

	return nil
}

func getDockerInspectResult(imageName string) ([]byte, error) {
	cmd := exec.Command("docker", "inspect", imageName)
	jsonOutput, err := cmd.CombinedOutput()
	if err != nil {
		if strings.Contains(string(jsonOutput), "permission denied") {
			logrus.Warnf("Error while running docker-inspect due to lack of permissions")
			logrus.Warnf("Please refer to [https://docs.docker.com/engine/install/linux-postinstall/] to fix this issue")
		} else if strings.Contains(string(jsonOutput), "No such object") {
			logrus.Warnf("Image [%s] not available in local image repo. Run \"docker pull %s\"", imageName, imageName)
			return nil, nil
		} else {
			logrus.Warnf("Error while running docker-inspect: %s", err)
		}
		return nil, err
	}
	return jsonOutput, nil
}

func getImageInfo(data []byte) collecttypes.ImageInfo {
	imageInfo := collecttypes.NewImageInfo()
	imgLayerInfo := []sourcetypes.DockerImage{}
	err := json.Unmarshal(data, &imgLayerInfo)
	if err != nil {
		logrus.Errorf("Unable to unmarshal image info : %s", err)
	}
	for _, image := range imgLayerInfo {
		imageInfo.Spec.Tags = image.RepoTags
		imageInfo.Spec.UserID, err = cast.ToIntE(image.CConfig.User)
		if err != nil {
			logrus.Debugf("UserID not available in image metadata for [%s]", image.RepoTags[0])
			imageInfo.Spec.UserID = -1
		}
		imageInfo.Spec.AccessedDirs = append(imageInfo.Spec.AccessedDirs, image.CConfig.WorkingDir)
		for key := range image.CConfig.EPorts {
			regex := regexp.MustCompile("[0-9]+")
			portNumber, err := cast.ToInt32E(string(regex.FindAll([]byte(key), -1)[0]))
			if err != nil {
				logrus.Debugf("PortNumber not available in image metadata for [%s]", image.RepoTags[0])
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
		logrus.Warnf("Error while running docker image list : %s", err)
		return nil, err
	}
	if len(outputStr) == 0 {
		return []string{}, err
	}
	cleanimages := []string{}
	images := strings.Split(strings.TrimSpace(string(outputStr)), "\n")
	for _, image := range images {
		if strings.HasPrefix(image, "<none>") || strings.HasSuffix(image, "<none>") {
			logrus.Debugf("Ignore image with <none> : %s", image)
			continue
		} else {
			cleanimages = append(cleanimages, image)
		}
	}
	logrus.Debugf("clean images : %s", cleanimages)
	return cleanimages, err
}

func getDCImageNames(directorypath string) ([]string, error) {
	var imageNames []string
	files, err := common.GetFilesByExt(directorypath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to fetch yaml files and recognize Docker image yamls : %s", err)
	}
	for _, path := range files {
		dc := sourcetypes.DockerCompose{}
		if common.ReadYaml(path, &dc) == nil {
			for _, dcservice := range dc.DCServices {
				imageNames = append(imageNames, dcservice.Image)
			}
		}
	}
	return imageNames, nil
}
