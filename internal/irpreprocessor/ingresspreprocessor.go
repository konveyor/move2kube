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

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
)

// ingressOptimizer optimizes the ingress options of the application
type ingressPreprocessor struct {
}

// customize modifies image paths and secret
func (opt *ingressPreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	if len(ir.Services) == 0 {
		logrus.Debugf("No services")
		return ir, nil
	}

	for sn, s := range ir.Services {
		tempService := ir.Services[sn]
		expose := false
		for pfi, pf := range s.ServiceToPodPortForwardings {
			if pf.ServicePort.Number == 0 {
				continue
			}
			key := common.ConfigServicesKey + common.Delim + `"` + sn + `"` + common.Delim + `"` + fmt.Sprintf("%d", pf.ServicePort.Number) + `"` + common.Delim + "urlpath"
			message := fmt.Sprintf("What URL/path should we expose the service %s's %d port on?", sn, pf.ServicePort.Number)
			hints := []string{"Enter :- not expose the service", "Leave out leading / to use first part as subdomain", "Add :N as suffix for NodePort service type", "Add :L for Load Balancer service type"}
			exposedServiceRelPath := ""
			if pf.ServiceRelPath != "" {
				exposedServiceRelPath = pf.ServiceRelPath
			} else {
				exposedServiceRelPath = "/" + sn
			}
			exposedServiceRelPath = strings.TrimSpace(qaengine.FetchStringAnswer(key, message, hints, exposedServiceRelPath))
			pf.ServiceRelPath = exposedServiceRelPath
			if exposedServiceRelPath != "" {
				expose = true
			}
			tempService.ServiceToPodPortForwardings[pfi] = pf
		}
		if tempService.Annotations == nil {
			tempService.Annotations = map[string]string{}
		}
		if expose {
			tempService.Annotations[common.ExposeSelector] = common.AnnotationLabelValue
		} else {
			delete(tempService.Annotations, common.ExposeSelector)
		}
		ir.Services[sn] = tempService
	}
	return ir, nil
}
