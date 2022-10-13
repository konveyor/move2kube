/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package irpreprocessor

import (
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/spf13/cast"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// ingressPreprocessor optimizes the ingress options of the application
type ingressPreprocessor struct {
}

func (opt *ingressPreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	for serviceName, service := range ir.Services {
		tempService := ir.Services[serviceName]
		for portForwardingIdx, portForwarding := range service.ServiceToPodPortForwardings {
			if portForwarding.ServicePort.Number == 0 {
				continue
			}
			if portForwarding.ServiceRelPath == "" {
				portForwarding.ServiceRelPath = "/" + serviceName
			}
			noneServiceType := "Don't create service"
			portKeyPart := common.JoinQASubKeys(common.ConfigServicesKey, `"`+serviceName+`"`, `"`+cast.ToString(portForwarding.ServicePort.Number)+`"`)
			options := []string{common.IngressKind, string(core.ServiceTypeLoadBalancer), string(core.ServiceTypeNodePort), string(core.ServiceTypeClusterIP), noneServiceType}
			desc := fmt.Sprintf("What kind of service/ingress should be created for the service %s's %d port?", serviceName, portForwarding.ServicePort.Number)
			hints := []string{"Choose " + common.IngressKind + " if you want a ingress/route resource to be created"}
			quesKey := common.JoinQASubKeys(portKeyPart, "servicetype")
			portForwarding.ServiceType = core.ServiceType(qaengine.FetchSelectAnswer(quesKey, desc, hints, common.IngressKind, options, nil))
			if string(portForwarding.ServiceType) == noneServiceType {
				portForwarding.ServiceType = ""
			}
			if string(portForwarding.ServiceType) == common.IngressKind {
				desc := fmt.Sprintf("Specify the ingress path to expose the service %s's %d port on?", serviceName, portForwarding.ServicePort.Number)
				hints := []string{"Leave out leading / to use first part as subdomain"}
				quesKey := common.JoinQASubKeys(portKeyPart, "urlpath")
				portForwarding.ServiceRelPath = strings.TrimSpace(qaengine.FetchStringAnswer(quesKey, desc, hints, portForwarding.ServiceRelPath, nil))
				portForwarding.ServiceType = core.ServiceTypeClusterIP
			} else {
				portForwarding.ServiceRelPath = ""
			}
			tempService.ServiceToPodPortForwardings[portForwardingIdx] = portForwarding
		}
		ir.Services[serviceName] = tempService
	}
	return ir, nil
}
