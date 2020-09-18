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

package optimize

import (
	irtypes "github.com/konveyor/move2kube/internal/types"
	corev1 "k8s.io/api/core/v1"
)

//PortMergeOptimizer implements Optimizer interface
type portMergeOptimizer struct {
}

//Optimize uses data from ir containers to fill ir.services
func (ep portMergeOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {
	for k, scObj := range ir.Services {
		for csIndex, containerSection := range scObj.Containers {
			if c, ok := ir.GetContainer(containerSection.Image); ok {
				for _, exposedPort := range c.ExposedPorts {
					containerSection.Ports = append(containerSection.Ports, corev1.ContainerPort{ContainerPort: int32(exposedPort)})
				}
				scObj.Containers[csIndex] = containerSection
			}
		}

		ir.Services[k] = scObj
	}

	return ir, nil
}
