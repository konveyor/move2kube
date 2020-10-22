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
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

// ingressOptimizer optimizes the ingress options of the application
type ingressOptimizer struct {
}

const (
	exposeSelector = types.GroupName + ".service.expose"
	exposeLabel    = "true"
)

// customize modifies image paths and secret
func (ic ingressOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {
	if len(ir.Services) == 0 {
		log.Debugf("No services to optimize")
		return ir, nil
	}

	// Obtain a listing of services.
	i := 0
	servicesList := make([]string, len(ir.Services))
	for k := range ir.Services {
		servicesList[i] = k
		i++
	}

	problem, err := qatypes.NewMultiSelectProblem("Select all services that should be exposed:",
		[]string{"The services unselected here will not be exposed."},
		servicesList,
		servicesList)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	exposedServices, err := problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}

	if len(exposedServices) > 0 {
		host, tlsSecret := ic.configureHostAndTLS(ir.Name)
		ir.TargetClusterSpec.Host = host
		ir.IngressTLSName = tlsSecret
		for _, k := range exposedServices {
			tempService := ir.Services[k]
			log.Infof("exposed service: %s", k)
			// Set the line in annotations
			if tempService.Annotations == nil {
				tempService.Annotations = make(map[string]string)
			}
			tempService.Annotations[exposeSelector] = exposeLabel
			if !tempService.IsServiceExposed() {
				tempService.ServiceRelPath = "/" + tempService.Name
			}
			ir.Services[k] = tempService
		}
	} else {
		log.Infof("No service exposed. skippig domain and TLS configuration")
	}

	return ir, nil
}

func (ic ingressOptimizer) configureHostAndTLS(name string) (string, string) {
	defaultSubDomain := name + ".com"

	problem, err := qatypes.NewInputProblem("Provide the ingress host domain",
		[]string{"Ingress host domain is part of service URL"},
		defaultSubDomain)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	host, err := problem.GetStringAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}

	defaultSecret := ""
	problem, err = qatypes.NewInputProblem("Provide the TLS secret for ingress", []string{"Enter TLS secret name"}, defaultSecret)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	secret, err := problem.GetStringAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}

	return host, secret
}
