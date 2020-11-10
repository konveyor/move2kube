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

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

// Any2KubeTranslator implements Translator interface for loading any source folder that can be containerized
type Any2KubeTranslator struct {
}

// GetTranslatorType returns translator type
func (*Any2KubeTranslator) GetTranslatorType() plantypes.TranslationTypeValue {
	return plantypes.Any2KubeTranslation
}

// GetServiceOptions - output a plan based on the input directory contents
func (any2KubeTranslator *Any2KubeTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	services := []plantypes.Service{}
	containerizers := new(containerizer.Containerizers)
	containerizers.InitContainerizers(inputPath)
	preContainerizedSourcePaths := []string{}
	for _, existingServices := range plan.Spec.Inputs.Services {
		for _, existingService := range existingServices {
			if len(existingService.SourceArtifacts[plantypes.SourceDirectoryArtifactType]) > 0 {
				preContainerizedSourcePaths = append(preContainerizedSourcePaths, existingService.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0])
			}
		}
	}

	ignoreDirectories, ignoreContents := any2KubeTranslator.getIgnorePaths(inputPath)

	err := filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Warnf("Skipping path %q due to error. Error: %q", path, err)
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		if common.IsStringPresent(preContainerizedSourcePaths, path) {
			return filepath.SkipDir //TODO: Should we go inside the directory in this case?
		}
		if common.IsStringPresent(ignoreDirectories, path) {
			if common.IsStringPresent(ignoreContents, path) {
				return filepath.SkipDir
			}
			return nil
		}
		containerizationOptions := containerizers.GetContainerizationOptions(plan, path)
		if len(containerizationOptions) == 0 {
			log.Debugf("No known containerization approach is supported for directory %q", path)
			if common.IsStringPresent(ignoreContents, path) {
				return filepath.SkipDir
			}
			return nil
		}
		for _, containerizationOption := range containerizationOptions {
			serviceName := filepath.Base(path)
			service := any2KubeTranslator.newService(serviceName)
			service.ContainerBuildType = containerizationOption.ContainerizationType
			service.ContainerizationTargetOptions = containerizationOption.TargetOptions
			if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], path) {
				service.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = append(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType], path)
				service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType] = append(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], path)
			}
			if foundRepo, err := service.GatherGitInfo(path, plan); foundRepo && err != nil {
				log.Warnf("Error while parsing the git repo at path %q Error: %q", path, err)
			}
			services = append(services, service)
		}
		return filepath.SkipDir // Skip all subdirectories when base directory is a valid package
	})

	if err != nil {
		log.Errorf("Error occurred while walking through the directory at path %q Error: %q", inputPath, err)
	}

	return services, err
}

// Translate translates artifacts to IR
func (any2KubeTranslator *Any2KubeTranslator) Translate(services []plantypes.Service, plan plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(plan)
	containerizers := new(containerizer.Containerizers)
	containerizers.InitContainerizers(plan.Spec.Inputs.RootDir)
	for _, service := range services {
		if service.TranslationType != any2KubeTranslator.GetTranslatorType() {
			continue
		}
		log.Debugf("Translating %s", service.ServiceName)
		container, err := containerizers.GetContainer(plan, service)
		if err != nil {
			log.Errorf("Unable to translate service %s Error: %q", service.ServiceName, err)
			continue
		}
		ir.AddContainer(container)
		serviceContainer := corev1.Container{Name: service.ServiceName}
		serviceContainer.Image = service.Image
		serviceContainerPorts := []corev1.ContainerPort{}
		for _, port := range container.ExposedPorts {
			serviceContainerPort := corev1.ContainerPort{ContainerPort: int32(port)}
			serviceContainerPorts = append(serviceContainerPorts, serviceContainerPort)
		}
		serviceContainer.Ports = serviceContainerPorts
		irService := irtypes.NewServiceFromPlanService(service)
		irService.Containers = []corev1.Container{serviceContainer}
		ir.Services[service.ServiceName] = irService
	}
	return ir, nil
}

func (any2KubeTranslator *Any2KubeTranslator) newService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, any2KubeTranslator.GetTranslatorType())
	service.AddSourceType(plantypes.DirectorySourceTypeValue)
	service.UpdateContainerBuildPipeline = true
	service.UpdateDeployPipeline = true
	return service
}

func (*Any2KubeTranslator) getIgnorePaths(inputPath string) (ignoreDirectories []string, ignoreContents []string) {
	filePaths, err := common.GetFilesByName(inputPath, []string{common.IgnoreFilename})
	if err != nil {
		log.Warnf("Unable to fetch .m2kignore files at path %q Error: %q", inputPath, err)
		return ignoreDirectories, ignoreContents
	}
	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			log.Warnf("Failed to open the .m2kignore file at path %q Error: %q", filePath, err)
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasSuffix(line, "*") {
				line = strings.TrimSuffix(line, "*")
				path := filepath.Join(filepath.Dir(filePath), line)
				ignoreContents = append(ignoreContents, path)
			} else {
				path := filepath.Join(filepath.Dir(filePath), line)
				ignoreDirectories = append(ignoreDirectories, path)
			}
		}
	}
	return ignoreDirectories, ignoreContents
}
