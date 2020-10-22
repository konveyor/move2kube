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

package source

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// Any2KubeTranslator implements Translator interface for loading any source folder that can be containerized
type Any2KubeTranslator struct {
}

// GetTranslatorType returns translator type
func (c Any2KubeTranslator) GetTranslatorType() plantypes.TranslationTypeValue {
	return plantypes.Any2KubeTranslation
}

// GetServiceOptions - output a plan based on the input directory contents
func (c Any2KubeTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	services := []plantypes.Service{}
	containerizers := new(containerizer.Containerizers)
	containerizers.InitContainerizers(inputPath)
	preContainerizedSourcePaths := []string{}
	for _, existingservices := range plan.Spec.Inputs.Services {
		for _, existingservice := range existingservices {
			if len(existingservice.SourceArtifacts[plantypes.SourceDirectoryArtifactType]) > 0 {
				preContainerizedSourcePaths = append(preContainerizedSourcePaths, existingservice.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0])
			}
		}
	}
	ignoreDirectories, ignoreContents := c.getIgnorePaths(inputPath)
	err := filepath.Walk(inputPath, func(fullpath string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %s due to error: %s", fullpath, err)
			return nil
		}
		if info.IsDir() {
			path, _ := plan.GetRelativePath(fullpath)
			if common.IsStringPresent(preContainerizedSourcePaths, path) {
				return filepath.SkipDir //TODO: Should we go inside the directory in this case?
			}
			fullcleanpath, err := filepath.Abs(fullpath)
			if err != nil {
				log.Errorf("Unable to resolve full path of directory %s : %s", fullcleanpath, err)
			}
			if common.IsStringPresent(ignoreDirectories, fullcleanpath) {
				if common.IsStringPresent(ignoreContents, fullcleanpath) {
					return filepath.SkipDir
				}
				return nil
			}
			containerizationoptions := containerizers.GetContainerizationOptions(plan, fullpath)
			if len(containerizationoptions) == 0 {
				log.Debugf("No known containerization approach is supported for %s", fullpath)
				if common.IsStringPresent(ignoreContents, fullcleanpath) {
					return filepath.SkipDir
				}
				return nil
			}
			for _, co := range containerizationoptions {
				expandedPath, err := filepath.Abs(fullpath) // If fullpath is "." it will expand to the absolute path.
				if err != nil {
					log.Warnf("Failed to get the absolute path for %s", fullpath)
					continue
				}
				service := c.newService(filepath.Base(expandedPath))
				service.ContainerBuildType = co.ContainerizationType
				service.ContainerizationTargetOptions = co.TargetOptions
				if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], path) {
					service.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = append(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType], path)
					service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType] = append(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], path)
				}
				if foundRepo, err := service.GatherGitInfo(fullpath, plan); foundRepo && err != nil {
					log.Warnf("Error while parsing the git repo at path %q Error: %q", fullpath, err)
				}
				services = append(services, service)
			}
			//return nil
			return filepath.SkipDir // Skipping all subdirectories when base directory is a valid package
		}
		return nil
	})
	return services, err
}

// Translate translates artifacts to IR
func (c Any2KubeTranslator) Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(p)
	containerizers := new(containerizer.Containerizers)
	containerizers.InitContainerizers(p.Spec.Inputs.RootDir)
	for _, service := range services {
		if service.TranslationType != c.GetTranslatorType() {
			continue
		}
		log.Debugf("Translating %s", service.ServiceName)
		serviceConfig := irtypes.NewServiceFromPlanService(service)
		container, err := containerizers.GetContainer(p, service)
		if err != nil {
			log.Errorf("Unable to translate service %s : %s", service.ServiceName, err)
			continue
		}
		ir.AddContainer(container)
		serviceContainer := corev1.Container{Name: service.ServiceName}
		serviceContainer.Image = service.Image
		serviceConfig.Containers = []corev1.Container{serviceContainer}
		ir.Services[service.ServiceName] = serviceConfig
	}
	return ir, nil
}

func (c Any2KubeTranslator) newService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, c.GetTranslatorType())
	service.AddSourceType(plantypes.DirectorySourceTypeValue)
	service.UpdateContainerBuildPipeline = true
	service.UpdateDeployPipeline = true
	return service
}

func (c Any2KubeTranslator) getIgnorePaths(inputPath string) (ignoreDirectories []string, ignoreContents []string) {
	ignorefiles, err := common.GetFilesByName(inputPath, []string{"." + types.AppNameShort + "ignore"})
	if err != nil {
		log.Warnf("Unable to fetch files to recognize ignore files : %s", err)
	}
	for _, ignorefile := range ignorefiles {
		file, err := os.Open(ignorefile)
		if err != nil {
			log.Warnf("Failed opening ignore file: %s", err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasSuffix(line, "*") {
				line = strings.TrimSuffix(line, "*")
				path, err := filepath.Abs(filepath.Join(filepath.Dir(ignorefile), line))
				if err != nil {
					log.Errorf("Unable to resolve full path of directory %s : %s", path, err)
				}
				ignoreContents = append(ignoreContents, path)
			} else {
				path, err := filepath.Abs(filepath.Join(filepath.Dir(ignorefile), line))
				if err != nil {
					log.Errorf("Unable to resolve full path of directory %s : %s", path, err)
				}
				ignoreDirectories = append(ignoreDirectories, path)
			}
		}
	}
	return ignoreDirectories, ignoreContents
}
