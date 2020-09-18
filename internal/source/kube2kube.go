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

package source

import (
	"github.com/konveyor/move2kube/internal/apiresourceset"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// KubeTranslator implements Translator interface
type KubeTranslator struct {
}

// GetTranslatorType returns the translator type
func (c KubeTranslator) GetTranslatorType() plantypes.TranslationTypeValue {
	return plantypes.Kube2KubeTranslation
}

// GetServiceOptions returns the possible service options for the inputPath
func (c KubeTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	return (&apiresourceset.K8sAPIResourceSet{}).GetServiceOptions(inputPath, plan)
}

// Translate returns the IR representation for the plan service
func (c KubeTranslator) Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error) {
	return (&apiresourceset.K8sAPIResourceSet{}).Translate(services, p)
}

func (c KubeTranslator) newService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, c.GetTranslatorType())
	service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue
	return service
}
