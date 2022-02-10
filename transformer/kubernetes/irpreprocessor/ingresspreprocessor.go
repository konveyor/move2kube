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
	core "k8s.io/kubernetes/pkg/apis/core"
)

// ingressPreprocessor optimizes the ingress options of the application
type ingressPreprocessor struct {
}

func (opt *ingressPreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	for sn, s := range ir.Services {
		tempService := ir.Services[sn]
		for pfi, pf := range s.ServiceToPodPortForwardings {
			if pf.ServicePort.Number == 0 {
				continue
			}
			if pf.ServiceRelPath == "" {
				pf.ServiceRelPath = "/" + sn
			}
			noneServiceType := "Don't create service"
			portKeyPart := common.ConfigServicesKey + common.Delim + `"` + sn + `"` + common.Delim + `"` + fmt.Sprintf("%d", pf.ServicePort.Number) + `"` + common.Delim
			options := []string{common.IngressKind, string(core.ServiceTypeLoadBalancer), string(core.ServiceTypeNodePort), string(core.ServiceTypeClusterIP), noneServiceType}
			pf.ServiceType = core.ServiceType(qaengine.FetchSelectAnswer(portKeyPart+"servicetype", fmt.Sprintf("What kind of service/ingress to create for %s's %d port?", sn, pf.ServicePort.Number), []string{"Choose " + common.IngressKind + " if you want a ingress/route resource to be created"}, common.IngressKind, options))
			if string(pf.ServiceType) == noneServiceType {
				pf.ServiceType = ""
			}
			if string(pf.ServiceType) == common.IngressKind {
				pf.ServiceRelPath = strings.TrimSpace(qaengine.FetchStringAnswer(portKeyPart+"urlpath", fmt.Sprintf("Specify the ingress path to expose %s's %d port?", sn, pf.ServicePort.Number), []string{"Leave out leading / to use first part as subdomain"}, pf.ServiceRelPath))
				pf.ServiceType = core.ServiceTypeClusterIP
			} else {
				pf.ServiceRelPath = ""
			}
			tempService.ServiceToPodPortForwardings[pfi] = pf
		}
		ir.Services[sn] = tempService
	}
	return ir, nil
}
