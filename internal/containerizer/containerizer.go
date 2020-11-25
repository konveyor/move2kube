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

	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

const (
	containerizerJSONPort      = "port"
	containerizerJSONBuilder   = "builder"
	containerizerJSONImageName = "image_name"
)

//go:generate go run github.com/konveyor/move2kube/internal/common/generator scripts

// Containerizer interface defines interface for containerizing applications
type Containerizer interface {
	Init(path string)
	GetTargetOptions(plan plantypes.Plan, path string) []string
	GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error)
	GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue
}

// Containerizers keep track of all initialized containerizers
type Containerizers struct {
	containerizers []Containerizer
}

// ContainerizationOption defines the containerization option for a path
type ContainerizationOption struct {
	ContainerizationType plantypes.ContainerBuildTypeValue
	TargetOptions        []string
}

// InitContainerizers initializes the containerizers
func (c *Containerizers) InitContainerizers(path string) {
	c.containerizers = []Containerizer{new(DockerfileContainerizer), new(S2IContainerizer), new(CNBContainerizer), new(ReuseContainerizer)}
	for _, containerizer := range c.containerizers {
		containerizer.Init(path)
		containerizer.Init(common.AssetsPath)
	}
}

// GetContainerizationOptions returns ContainerizerOptions for given sourcepath
func (c *Containerizers) GetContainerizationOptions(plan plantypes.Plan, sourcepath string) []ContainerizationOption {
	cops := []ContainerizationOption{}
	for _, containerizer := range c.containerizers {
		if targetoptions := containerizer.GetTargetOptions(plan, sourcepath); len(targetoptions) != 0 {
			cops = append(cops, ContainerizationOption{
				ContainerizationType: containerizer.GetContainerBuildStrategy(),
				TargetOptions:        targetoptions,
			})
		}
	}
	return cops
}

// GetContainer get the container for a service
func (c *Containerizers) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	for _, containerizer := range c.containerizers {
		if containerizer.GetContainerBuildStrategy() != service.ContainerBuildType {
			continue
		}
		log.Debugf("Containerizing %s using %s", service.ServiceName, service.ContainerBuildType)
		container, err := containerizer.GetContainer(plan, service)
		if err != nil {
			log.Errorf("Error during containerization : %s", err)
			return irtypes.Container{}, err
		}
		return container, nil
	}
	return irtypes.Container{}, fmt.Errorf("service %s has an invalid containerization strategy %s", service.ServiceName, service.ContainerBuildType)
}
