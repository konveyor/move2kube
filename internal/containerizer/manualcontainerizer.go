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

	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// ManualContainerizer implements Containerizer interface
type ManualContainerizer struct {
}

// Init initializes the containerizer
func (d *ManualContainerizer) Init(path string) {
}

// GetTargetOptions returns empty for Manual
func (d ManualContainerizer) GetTargetOptions(plan plantypes.Plan, path string) []string {
	return []string{}
}

// GetContainerBuildStrategy returns the containerbuildstrategy
func (d ManualContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.ManualContainerBuildTypeValue
}

// GetContainer returns the container for a service
func (d ManualContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	// TODO: Fix exposed ports too
	if service.ContainerBuildType == d.GetContainerBuildStrategy() {
		container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)
		return container, nil
	}
	return irtypes.Container{}, fmt.Errorf("unsupported service type for Containerization or insufficient information in service")
}
