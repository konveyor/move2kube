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

package irpreprocessor

import (
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

//PortMergeOptimizer implements Optimizer interface
type portMergePreprocessor struct {
}

//Optimize uses data from ir containers to fill ir.services
func (opt *portMergePreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	for serviceName, service := range ir.Services {
		serviceHasNoPorts := true
		for _, coreContainer := range service.Containers {
			if len(coreContainer.Ports) > 0 {
				serviceHasNoPorts = false
				break
			}
		}
		if !serviceHasNoPorts {
			continue
		}

		// The service has no ports so we gather some eligible ports and ask a question.
		logrus.Debugf("The service %s has no ports", service.Name)
		portToContainerIdx := opt.gatherPorts(ir, service)
		if len(portToContainerIdx) == 0 {
			continue
		}
		opt.exposePorts(&service, portToContainerIdx)
		ir.Services[serviceName] = service
	}

	return ir, nil
}

func (*portMergePreprocessor) gatherPorts(ir irtypes.IR, service irtypes.Service) map[int]int {
	portToContainerIdx := map[int]int{}
	for coreContainerIdx, coreContainer := range service.Containers {
		if irContainer, ok := ir.ContainerImages[coreContainer.Image]; ok {
			for _, exposedPort := range irContainer.ExposedPorts {
				if oldcoreContainerIdx, ok := portToContainerIdx[exposedPort]; ok {
					logrus.Debugf("The port %d is eligible to be exposed by both container %s and container %s of service %s",
						exposedPort, service.Containers[oldcoreContainerIdx].Name, coreContainer.Name, service.Name)
					continue
				}
				portToContainerIdx[exposedPort] = coreContainerIdx
			}
		}
	}
	if len(portToContainerIdx) == 0 {
		if len(service.Containers) == 0 {
			logrus.Infof("The service %s has no ports because it has no containers.", service.Name)
		} else {
			logrus.Infof("No ports detected for service %s . Adding default port %d", service.Name, common.DefaultServicePort)
			portToContainerIdx[common.DefaultServicePort] = 0 // the first container index
		}
	}
	return portToContainerIdx
}

func (*portMergePreprocessor) exposePorts(service *irtypes.Service, portToContainerIdx map[int]int) {
	for port, coreContainerIdx := range portToContainerIdx {
		// Add the port to the k8s pod.
		service.Containers[coreContainerIdx].Ports = append(service.Containers[coreContainerIdx].Ports, core.ContainerPort{ContainerPort: int32(port)})
		// Forward the port on the k8s service to the k8s pod.
		podPort := irtypes.Port{Number: int32(port)}
		servicePort := podPort
		service.AddPortForwarding(servicePort, podPort)
	}
}
