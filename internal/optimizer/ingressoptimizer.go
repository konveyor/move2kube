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
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	log "github.com/sirupsen/logrus"
)

// ingressOptimizer optimizes the ingress options of the application
type ingressOptimizer struct {
}

// customize modifies image paths and secret
func (opt *ingressOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {
	if len(ir.Services) == 0 {
		log.Debugf("No services to optimize")
		return ir, nil
	}

	// Obtain a listing of services.
	serviceNames := []string{}
	exposedServiceNames := []string{}
	for serviceName, service := range ir.Services {
		serviceNames = append(serviceNames, serviceName)
		if service.ServiceRelPath != "" {
			exposedServiceNames = append(exposedServiceNames, serviceName)
		}
	}

	exposedServiceNames = qaengine.FetchMultiSelectAnswer(common.ConfigServicesExposeKey, "Select all services that should be exposed:", []string{"Exposed services will be reachable from outside the cluster."}, exposedServiceNames, serviceNames)
	if len(exposedServiceNames) == 0 {
		log.Debugf("User deselected all services. Not exposing anything.")
		return ir, nil
	}

	for _, exposedServiceName := range exposedServiceNames {
		key := common.ConfigServicesKey + common.Delim + `"` + exposedServiceName + `"` + common.Delim + "urlpath"
		message := fmt.Sprintf("What URL/path should we expose the service %s on?", exposedServiceName)
		hints := []string{"By default we expose the service on /<service name>:"}
		exposedServiceRelPath := "/" + exposedServiceName
		if len(exposedServiceNames) == 1 {
			hints = []string{"Since there's only one exposed service, the default path is /"}
			exposedServiceRelPath = "/"
		}
		exposedServiceRelPath = qaengine.FetchStringAnswer(key, message, hints, exposedServiceRelPath)
		log.Debugf("Exposing service %s on path %s", exposedServiceName, exposedServiceRelPath)

		exposedServiceRelPath = opt.normalizeServiceRelPath(exposedServiceRelPath)

		tempService := ir.Services[exposedServiceName]
		tempService.ServiceRelPath = exposedServiceRelPath
		if tempService.Annotations == nil {
			tempService.Annotations = map[string]string{}
		}
		tempService.Annotations[common.ExposeSelector] = common.AnnotationLabelValue
		ir.Services[exposedServiceName] = tempService
	}

	return ir, nil
}

func (opt *ingressOptimizer) normalizeServiceRelPath(exposedServiceRelPath string) string {
	exposedServiceRelPath = strings.TrimSpace(exposedServiceRelPath)
	if len(exposedServiceRelPath) == 0 {
		log.Warnf("User gave an empty service path. Assuming it should be exposed on /")
	}
	if !strings.HasPrefix(exposedServiceRelPath, "/") {
		exposedServiceRelPath = "/" + exposedServiceRelPath
	}
	return exposedServiceRelPath
}
