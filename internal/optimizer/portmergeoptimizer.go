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
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

//PortMergeOptimizer implements Optimizer interface
type portMergeOptimizer struct {
}

//Optimize uses data from ir containers to fill ir.services
func (opt *portMergeOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {
	for serviceName, service := range ir.Services {
		serviceHasNoPorts := true
		for _, coreV1Container := range service.Containers {
			if len(coreV1Container.Ports) > 0 {
				serviceHasNoPorts = false
				break
			}
		}
		if !serviceHasNoPorts {
			continue
		}

		// The service has no ports so we gather some eligible ports and ask a question.
		log.Debugf("The service %s has no ports", service.Name)
		portToContainerIdx := opt.gatherPorts(ir, service)
		if len(portToContainerIdx) == 0 {
			continue
		}
		opt.exposePorts(&service, portToContainerIdx)
		ir.Services[serviceName] = service
	}

	return ir, nil
}

func (*portMergeOptimizer) gatherPorts(ir irtypes.IR, service irtypes.Service) map[int]int {
	portToContainerIdx := map[int]int{}
	for coreV1ContainerIdx, coreV1Container := range service.Containers {
		if irContainer, ok := ir.GetContainer(coreV1Container.Image); ok {
			for _, exposedPort := range irContainer.ExposedPorts {
				if oldCoreV1ContainerIdx, ok := portToContainerIdx[exposedPort]; ok {
					log.Debugf("The port %d is eligible to be exposed by both container %s and container %s of service %s",
						exposedPort, service.Containers[oldCoreV1ContainerIdx].Name, coreV1Container.Name, service.Name)
					continue
				}
				portToContainerIdx[exposedPort] = coreV1ContainerIdx
			}
		}
	}
	if len(portToContainerIdx) == 0 {
		if len(service.Containers) == 0 {
			log.Infof("The service %s has no ports because it has no containers.", service.Name)
		} else {
			log.Infof("No ports detected for service %s . Adding default port %d", service.Name, common.DefaultServicePort)
			portToContainerIdx[common.DefaultServicePort] = 0 // the first container index
		}
	}
	return portToContainerIdx
}

func (*portMergeOptimizer) exposePorts(service *irtypes.Service, portToContainerIdx map[int]int) {
	for port, coreV1ContainerIdx := range portToContainerIdx {
		// Add the port to the k8s pod.
		service.Containers[coreV1ContainerIdx].Ports = append(service.Containers[coreV1ContainerIdx].Ports, corev1.ContainerPort{ContainerPort: int32(port)})
		// Forward the port on the k8s service to the k8s pod.
		podPort := irtypes.Port{Number: int32(port)}
		servicePort := podPort
		service.AddPortForwarding(servicePort, podPort)
	}
}
