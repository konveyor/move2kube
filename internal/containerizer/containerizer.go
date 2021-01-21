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
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

//go:generate go run github.com/konveyor/move2kube/internal/common/generator scripts

// Containerizer can be used to containerize applications given path to the source code
type Containerizer interface {
	Init(path string)
	GetTargetOptions(plan plantypes.Plan, path string) []string
	GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue
	GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error)
}

// ContainerizationOption defines the containerization option for a path
type ContainerizationOption struct {
	ContainerizationType plantypes.ContainerBuildTypeValue
	TargetOptions        []string
}

const (
	containerizerJSONPort      = "port"
	containerizerJSONBuilder   = "builder"
	containerizerJSONImageName = "image_name"
)

var containerizers []Containerizer

// InitContainerizers initializes the containerizers
func InitContainerizers(path string, containerizerTypes []string) {
	containerizers = []Containerizer{}
	for _, containerizer := range getAllContainerizers() {
		cbs := string(containerizer.GetContainerBuildStrategy())
		if containerizerTypes == nil || common.IsStringPresent(containerizerTypes, cbs) {
			containerizer.Init(path)
			containerizer.Init(common.AssetsPath)
			containerizers = append(containerizers, containerizer)
		}
	}
}

// getAllContainerizers gets the all containerizers uninitialized
func getAllContainerizers() []Containerizer {
	return []Containerizer{new(DockerfileContainerizer), new(S2IContainerizer), new(CNBContainerizer), new(ReuseContainerizer)}
}

// GetAllContainerBuildStrategies returns all translator types
func GetAllContainerBuildStrategies() []string {
	cbs := []string{}
	for _, c := range getAllContainerizers() {
		cbs = append(cbs, string(c.GetContainerBuildStrategy()))
	}
	return cbs
}

// GetContainerizationOptions returns ContainerizerOptions for given sourcepath
func GetContainerizationOptions(plan plantypes.Plan, sourcepath string) []ContainerizationOption {
	cops := []ContainerizationOption{}
	for _, containerizer := range containerizers {
		if targetOptions := containerizer.GetTargetOptions(plan, sourcepath); len(targetOptions) != 0 {
			cops = append(cops, ContainerizationOption{
				ContainerizationType: containerizer.GetContainerBuildStrategy(),
				TargetOptions:        targetOptions,
			})
		}
	}
	return cops
}

// GetContainer get the container for a service
func GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	for _, containerizer := range containerizers {
		if containerizer.GetContainerBuildStrategy() != service.ContainerBuildType {
			continue
		}
		log.Debugf("Containerizing %s using %s", service.ServiceName, service.ContainerBuildType)
		container, err := containerizer.GetContainer(plan, service)
		if err != nil {
			log.Errorf("Error during containerization : %s", err)
			return container, err
		}
		return container, nil
	}
	return irtypes.Container{}, fmt.Errorf("service %s has an invalid containerization strategy %s", service.ServiceName, service.ContainerBuildType)
}
