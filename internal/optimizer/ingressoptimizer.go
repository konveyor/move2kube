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
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

// ingressOptimizer optimizes the ingress options of the application
type ingressOptimizer struct {
}

// customize modifies image paths and secret
func (ic ingressOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {
	if len(ir.Services) == 0 {
		log.Debugf("No services to optimize")
		return ir, nil
	}

	// Obtain a listing of services.
	i := 0
	exposedServices := make([]string, 0)
	servicesList := make([]string, 0)
	for sn, s := range ir.Services {
		servicesList = append(servicesList, sn)
		if s.ServiceRelPath != "" {
			exposedServices = append(exposedServices, sn)
		}
		i++
	}

	problem, err := qatypes.NewMultiSelectProblem("Select all services that should be exposed:",
		[]string{"The services unselected here will not be exposed."},
		servicesList,
		exposedServices)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	exposedServices, err = problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}

	for _, k := range exposedServices {
		tempService := ir.Services[k]
		log.Debugf("Exposed service: %s", k)
		// Set the line in annotations
		if tempService.Annotations == nil {
			tempService.Annotations = make(map[string]string)
		}
		tempService.Annotations[common.ExposeSelector] = common.AnnotationLabelValue
		if tempService.ServiceRelPath == "" {
			tempService.ServiceRelPath = "/"
		}
		ir.Services[k] = tempService
	}

	return ir, nil
}
