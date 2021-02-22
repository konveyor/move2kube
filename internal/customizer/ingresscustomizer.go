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
	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
)

//ingressCustomizer customizes ingress host
type ingressCustomizer struct {
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

	host := qaengine.FetchStringAnswer(common.ConfigIngressHostKey, "Provide the ingress host domain", []string{"Ingress host domain is part of service URL"}, defaultSubDomain)
	host = name + "." + host

	defaultSecret := ""
	secret := qaengine.FetchStringAnswer(common.ConfigIngressTLSKey, "Provide the TLS secret for ingress", []string{"Enter TLS secret name"}, defaultSecret)

	return host, secret
}
