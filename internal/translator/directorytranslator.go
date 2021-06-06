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

package translator

import (
	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// Any2KubeTranslator implements Translator interface for loading any source folder that can be containerized
type Any2KubeTranslator struct {
}

// GetTranslatorType returns translator type
func (*Any2KubeTranslator) GetTranslatorType() plantypes.TranslationTypeValue {
	return plantypes.Any2KubeTranslation
}

// Translate translates artifacts to IR
func (any2KubeTranslator *Any2KubeTranslator) Translate(services []plantypes.Service, plan plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(plan)
	for _, service := range services {
		if service.TranslationType != any2KubeTranslator.GetTranslatorType() {
			continue
		}
		log.Debugf("Translating %s", service.ServiceName)
		container, err := containerizer.GetContainer(plan, service)
		if err != nil {
			log.Errorf("Unable to translate service %s Error: %q", service.ServiceName, err)
			continue
		}
		ir.AddContainer(container)
		serviceContainer := core.Container{Name: service.ServiceName}
		serviceContainer.Image = service.Image
		irService := irtypes.NewServiceFromPlanService(service)
		serviceContainerPorts := []core.ContainerPort{}
		for _, port := range container.ExposedPorts {
			// Add the port to the k8s pod.
			serviceContainerPort := core.ContainerPort{ContainerPort: int32(port)}
			serviceContainerPorts = append(serviceContainerPorts, serviceContainerPort)
			// Forward the port on the k8s service to the k8s pod.
			podPort := irtypes.Port{Number: int32(port)}
			servicePort := podPort
			irService.AddPortForwarding(servicePort, podPort)
		}
		serviceContainer.Ports = serviceContainerPorts
		irService.Containers = []core.Container{serviceContainer}
		ir.Services[service.ServiceName] = irService
	}
	return ir, nil
}

func (any2KubeTranslator *Any2KubeTranslator) newService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, any2KubeTranslator.GetTranslatorType())
	service.AddSourceType(plantypes.DirectorySourceTypeValue)
	return service
}
