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

package irpreprocessor

import (
	irtypes "github.com/konveyor/move2kube/types/ir"
	core "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/networking"
)

//mergePreprocessor implements Optimizer interface
type mergePreprocessor struct {
}

//Optimize uses data from ir containers to fill ir.services
func (opt *mergePreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	for serviceName, service := range ir.Services {
		service.Containers = opt.mergeContainers(service.Containers)
		pfs := service.ServiceToPodPortForwardings
		service.ServiceToPodPortForwardings = []irtypes.ServiceToPodPortForwarding{}
		for _, pf := range pfs {
			service.AddPortForwarding(pf.ServicePort, pf.PodPort, pf.ServiceRelPath)
		}
		for _, c := range service.Containers {
			for _, p := range c.Ports {
				service.AddPortForwarding(networking.ServiceBackendPort{Number: p.ContainerPort}, networking.ServiceBackendPort{Number: p.ContainerPort}, "")
			}
		}
		tolerations := service.Tolerations
		service.Tolerations = []core.Toleration{}
		for _, t := range tolerations {
			if (t == core.Toleration{}) {
				continue
			}
			for _, ot := range service.Tolerations {
				if ot == t {
					continue
				}
			}
			service.Tolerations = append(service.Tolerations, t)
		}
		ir.Services[serviceName] = service
	}
	return ir, nil
}

func (opt *mergePreprocessor) mergeContainers(sContainers []core.Container) []core.Container {
	containers := map[string]core.Container{}
	for _, coreContainer := range sContainers {
		var container core.Container
		var ok bool
		if coreContainer.Name == "" {
			continue
		}
		if container, ok = containers[coreContainer.Name]; !ok {
			container = coreContainer
			uniquePorts := []core.ContainerPort{}
			for _, ccp := range coreContainer.Ports {
				found := false
				for _, cp := range uniquePorts {
					if ccp.ContainerPort == cp.ContainerPort {
						found = true
						break
					}
				}
				if !found && ccp.ContainerPort != 0 {
					uniquePorts = append(uniquePorts, ccp)
				}
			}
			container.Ports = uniquePorts
		}
		uniquePorts := container.Ports
		for _, ccp := range coreContainer.Ports {
			found := false
			for _, cp := range uniquePorts {
				if ccp.ContainerPort == cp.ContainerPort {
					found = true
					break
				}
			}
			if !found && ccp.ContainerPort != 0 {
				uniquePorts = append(uniquePorts, ccp)
			}
		}
		container.Ports = uniquePorts
		containers[coreContainer.Name] = container
		break
	}
	sContainers = []core.Container{}
	for _, c := range containers {
		sContainers = append(sContainers, c)
	}
	return sContainers
}
