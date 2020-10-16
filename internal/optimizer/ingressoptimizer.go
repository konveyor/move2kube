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
	"log"

	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

// ingressOptimizer identifies the services that needs to be externally exposed
type ingressOptimizer struct {
}

func (ingresso ingressOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {

	if len(ir.Services) == 0 {
		return ir, nil
	}

	// Obtain a listing of services.
	i := 0
	servicesList := make([]string, len(ir.Services))
	for k := range ir.Services {
		servicesList[i] = k
		i++
	}

	problem, err := qatypes.NewMultiSelectProblem("Which services should we expose?", []string{"An Ingress object will be created for every exposed service."}, servicesList, servicesList)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	services, err := problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}

	for _, k := range services {
		tempService := ir.Services[k]

		// Set the line in annotations
		if tempService.Annotations == nil {
			tempService.Annotations = map[string]string{}
		}
		tempService.Annotations[types.GroupName+"/expose"] = "true"
		// Also set the special field
		tempService.ExposeService = true
		ir.Services[k] = tempService
	}

	return ir, nil
}
