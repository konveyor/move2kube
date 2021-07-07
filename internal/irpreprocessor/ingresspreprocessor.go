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

	for sn := range ir.Services {
		key := common.ConfigServicesKey + common.Delim + `"` + sn + `"` + common.Delim + "urlpath"
		message := fmt.Sprintf("What URL/path should we expose the service %s on?", sn)
		hints := []string{"Enter empty string to not expose the service"}
		exposedServiceRelPath := "/" + sn
		exposedServiceRelPath = strings.TrimSpace(qaengine.FetchStringAnswer(key, message, hints, exposedServiceRelPath))
		logrus.Debugf("Exposing service %s on path %s", sn, exposedServiceRelPath)
		tempService := ir.Services[sn]
		tempService.ServiceRelPath = exposedServiceRelPath
		if exposedServiceRelPath != "" && !strings.HasPrefix(exposedServiceRelPath, "/") {
			exposedServiceRelPath = "/" + exposedServiceRelPath
		}
		if tempService.Annotations == nil {
			tempService.Annotations = map[string]string{}
		}
		if exposedServiceRelPath != "" {
			tempService.Annotations[common.ExposeSelector] = common.AnnotationLabelValue
		} else {
			delete(tempService.Annotations, common.ExposeSelector)
		}
		ir.Services[sn] = tempService
	}
	return ir, nil
}
