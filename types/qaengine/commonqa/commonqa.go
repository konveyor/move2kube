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

package commonqa

import (
	"net/url"

	dockercliconfig "github.com/docker/cli/cli/config"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/qaengine"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

// ImageRegistry returns Image Registry URL
func ImageRegistry() string {
	registryList := []string{qatypes.OtherAnswer}
	registryAuthList := map[string]string{} //Registry url and auth
	defreg := ""
	if !common.IgnoreEnvironment {
		configFile, err := dockercliconfig.Load(dockercliconfig.Dir())
		if err == nil {
			for regurl, regauth := range configFile.AuthConfigs {
				u, err := url.Parse(regurl)
				if err == nil {
					if u.Host != "" {
						regurl = u.Host
					}
				}
				if regurl == "" {
					continue
				}
				if !common.IsStringPresent(registryList, regurl) {
					registryList = append(registryList, regurl)
				}
				if regauth.Auth != "" {
					defreg = regurl
					registryAuthList[regurl] = regauth.Auth
				}
			}
		}
	}
	if !common.IsStringPresent(registryList, common.DefaultRegistryURL) {
		registryList = append(registryList, common.DefaultRegistryURL)
	}
	if defreg == "" {
		defreg = common.DefaultRegistryURL
	}
	return qaengine.FetchSelectAnswer(common.ConfigImageRegistryURLKey, "Enter the URL of the image registry : ", []string{"You can always change it later by changing the yamls."}, defreg, registryList)
}

// ImageRegistryNamespace returns Image Registry Namespace
func ImageRegistryNamespace(def string) string {
	return qaengine.FetchStringAnswer(common.ConfigImageRegistryNamespaceKey, "Enter the namespace where the new images should be pushed : ", []string{"Ex : " + def}, def)
}

// IngressHost returns Ingress host
func IngressHost(defaulthost string) string {
	return qaengine.FetchStringAnswer(common.ConfigIngressHostKey, "Provide the ingress host domain", []string{"Ingress host domain is part of service URL"}, defaulthost)
}
