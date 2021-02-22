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
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

//go:generate go run ../../scripts/generator/generator.go scripts

// Containerizer can be used to containerize applications given path to the source code
type Containerizer interface {
	Init(path string)
	GetTargetOptions(plan plantypes.Plan, path string) []string
	GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue
	GetContainer(serviceName string, containerizationOption plantypes.ContainerizationOption, plan plantypes.Plan) (irtypes.Container, error)
}

const (
	containerizerJSONPort      = "port"
	containerizerJSONBuilder   = "builder"
	containerizerJSONImageName = "image_name"
)

var containerizers map[plantypes.ContainerBuildTypeValue]Containerizer

// InitContainerizers initializes the containerizers
func InitContainerizers(path string, containerizerTypes []string) {
	containerizers = make(map[plantypes.ContainerBuildTypeValue]Containerizer)
	for _, containerizer := range getAllContainerizers() {
		cbs := containerizer.GetContainerBuildStrategy()
		if containerizerTypes == nil || common.IsStringPresent(containerizerTypes, string(cbs)) {
			containerizer.Init(path)
			containerizer.Init(common.AssetsPath)
			containerizers[cbs] = containerizer
		}
	}
}

// ComesBefore returns true if x < y i.e. x comes before y
func ComesBefore(x, y plantypes.ContainerBuildTypeValue) bool {
	xidx := -1
	yidx := -1
	buildTypes := GetAllContainerBuildStrategies()
	for i, buildType := range buildTypes {
		if buildType == string(x) {
			xidx = i
		}
		if buildType == string(y) {
			yidx = i
		}
	}
	return xidx < yidx
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
	cbs = append(cbs, string((new(ReuseDockerfileContainerizer)).GetContainerBuildStrategy()), string((new(ManualContainerizer)).GetContainerBuildStrategy()))
	return cbs
}

// GetContainerizationOptions returns ContainerizerOptions for given sourcepath
func GetContainerizationOptions(plan plantypes.Plan, sourcepath string) []plantypes.ContainerizationOption {
	cops := []plantypes.ContainerizationOption{}
	for _, containerizer := range containerizers {
		for _, option := range containerizer.GetTargetOptions(plan, sourcepath) {
			cops = append(cops, plantypes.ContainerizationOption{
				BuildType:   containerizer.GetContainerBuildStrategy(),
				ID:          option,
				ContextPath: sourcepath,
			})
		}
	}
	return cops
}

// GetContainer get the container for a service
func GetContainer(serviceName string, containerizationOption plantypes.ContainerizationOption, plan plantypes.Plan) (irtypes.Container, error) {
	log.Debugf("Containerizing %s using %s", serviceName, containerizationOption.BuildType)
	return containerizers[containerizationOption.BuildType].GetContainer(serviceName, containerizationOption, plan)
}
