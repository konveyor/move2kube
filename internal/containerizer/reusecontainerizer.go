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

// ReuseContainerizer implements Containerizer interface
type ReuseContainerizer struct {
}

// Init initializes the containerizer
func (d *ReuseContainerizer) Init(path string) {
}

// GetTargetOptions does nothing for reuse containerizer
func (d ReuseContainerizer) GetTargetOptions(plan plantypes.Plan, path string) []string {
	return []string{}
}

// GetContainerBuildStrategy returns the containerbuildstrategy that is supported
func (d ReuseContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.ReuseContainerBuildTypeValue
}

// GetContainer returns the container for a service
func (d ReuseContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	// TODO: Fix exposed ports too
	if service.ContainerBuildType == d.GetContainerBuildStrategy() {
		container := irtypes.NewContainer(service.Image, false)
		return container, nil
	}
	return irtypes.Container{}, fmt.Errorf("Unsupported service type for Containerization or insufficient information in service")
}
