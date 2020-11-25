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
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	corev1 "k8s.io/api/core/v1"
)

//PortMergeOptimizer implements Optimizer interface
type portMergeOptimizer struct {
}

//Optimize uses data from ir containers to fill ir.services
func (opt *portMergeOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {
	for _, service := range ir.Services {
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
		selectedPortToContainerIdx := opt.askQuestion(service, portToContainerIdx)
		if len(selectedPortToContainerIdx) == 0 {
			log.Infof("User deselected all ports. Not adding any ports to the service %s", service.Name)
			continue
		}
		opt.exposePorts(service, selectedPortToContainerIdx)
	}

	return ir, nil
}

func (*portMergeOptimizer) gatherPorts(ir irtypes.IR, service irtypes.Service) map[int]int {
	portToContainerIdx := map[int]int{}
	for coreV1ContainerIdx, coreV1Container := range service.Containers {
		if irContainer, ok := ir.GetContainer(coreV1Container.Image); ok {
			for _, exposedPort := range irContainer.ExposedPorts {
				if oldCoreV1ContainerIdx, ok := portToContainerIdx[exposedPort]; ok {
					log.Warnf("The port %d is eligible to be exposed by both container %s and container %s",
						exposedPort, service.Containers[oldCoreV1ContainerIdx].Name, coreV1Container.Name)
				}
				portToContainerIdx[exposedPort] = coreV1ContainerIdx
			}
		}
	}
	if len(portToContainerIdx) == 0 {
		if len(service.Containers) == 0 {
			log.Infof("The service %s has no ports because it has no containers.", service.Name)
		} else {
			log.Infof("Could not find any eligibile ports for the service %s . Adding default port %d", service.Name, common.DefaultServicePort)
			portToContainerIdx[common.DefaultServicePort] = 0 // the first container index
		}
	}
	return portToContainerIdx
}

func (*portMergeOptimizer) askQuestion(service irtypes.Service, portToContainerIdx map[int]int) map[int]int {
	eligiblePorts := []string{}
	for eligiblePort := range portToContainerIdx {
		eligiblePorts = append(eligiblePorts, cast.ToString(eligiblePort))
	}
	message := fmt.Sprintf("Service %s has no ports. Please select the ports that should be added to it:", service.Name)
	hint := []string{"If this is a headless service deselect all the ports."}
	problem, err := qatypes.NewMultiSelectProblem(message, hint, eligiblePorts, eligiblePorts)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	eligiblePorts, err = problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	if len(eligiblePorts) == 0 {
		return nil
	}
	selectedPortToContainerIdx := map[int]int{}
	for _, eligiblePort := range eligiblePorts {
		selectedPort, err := cast.ToIntE(eligiblePort)
		if err != nil {
			log.Debugf("Failed to parse %q as an integer port. Error: %q", eligiblePort, err)
			continue
		}
		selectedPortToContainerIdx[selectedPort] = portToContainerIdx[selectedPort]
	}
	return selectedPortToContainerIdx
}

func (*portMergeOptimizer) exposePorts(service irtypes.Service, portToContainerIdx map[int]int) {
	for port, coreV1ContainerIdx := range portToContainerIdx {
		coreV1Port := corev1.ContainerPort{ContainerPort: int32(port)}
		service.Containers[coreV1ContainerIdx].Ports = append(service.Containers[coreV1ContainerIdx].Ports, coreV1Port)
	}
}
