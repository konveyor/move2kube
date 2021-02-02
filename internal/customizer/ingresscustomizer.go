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

package customizer

import (
	log "github.com/sirupsen/logrus"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

//ingressCustomizer customizes ingress host
type ingressCustomizer struct {
	ir *irtypes.IR
}

//customize customizes the ingress host
func (ic *ingressCustomizer) customize(ir *irtypes.IR) error {

	anyServicesExposed := false
	for _, s := range ir.Services {
		if s.ServiceRelPath != "" {
			anyServicesExposed = true
			break
		}
	}

	if anyServicesExposed {
		host, tlsSecret := ic.configureHostAndTLS(ir.Name)
		ir.TargetClusterSpec.Host = host
		ir.IngressTLSSecretName = tlsSecret
	}
	return nil
}

func (ic ingressCustomizer) configureHostAndTLS(name string) (string, string) {
	defaultSubDomain := name + ".com"

	problem, err := qatypes.NewInputProblem(common.ConfigIngressHostKey, "Provide the ingress host domain", []string{"Ingress host domain is part of service URL"}, defaultSubDomain)
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

	host = name + "." + host

	defaultSecret := ""
	problem, err = qatypes.NewInputProblem(common.ConfigIngressTLSKey, "Provide the TLS secret for ingress", []string{"Enter TLS secret name"}, defaultSecret)
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
