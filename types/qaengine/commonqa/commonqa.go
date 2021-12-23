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
	"fmt"
	"net/url"
	"strconv"
	"strings"

	dockercliconfig "github.com/docker/cli/cli/config"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	"github.com/sirupsen/logrus"
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

// MinimumReplicaCount returns minimum replica count
func MinimumReplicaCount(defaultminreplicas string) string {
	return qaengine.FetchStringAnswer(common.ConfigMinReplicasKey, "Provide the minimum number of replicas each service should have", []string{"If the value is 0 pods won't be started by default"}, defaultminreplicas)
}

// GetPortsForService returns ports used by a service
func GetPortsForService(detectedPorts []int32, serviceName string) []int32 {
	var selectedPortsStr, detectedPortsStr []string
	var exposePorts []int32
	if len(detectedPorts) != 0 {
		for _, detectedPort := range detectedPorts {
			detectedPortsStr = append(detectedPortsStr, strconv.Itoa(int(detectedPort)))
		}
		allDetectedPortsStr := append(detectedPortsStr, qatypes.OtherAnswer)
		selectedPortsStr = qaengine.FetchMultiSelectAnswer(common.ConfigServicesKey+common.Delim+serviceName+common.Delim+common.ConfigPortsForServiceKeySegment, fmt.Sprintf("Select ports to be exposed for the service %s :", serviceName), []string{"Select Other if you want to add more ports"}, detectedPortsStr, allDetectedPortsStr)
	}
	for _, portStr := range selectedPortsStr {
		portStr = strings.TrimSpace(portStr)
		if portStr != "" {
			port, err := strconv.ParseInt(portStr, 10, 32)
			if err != nil {
				logrus.Errorf("Error while converting the selected port from string to int : %s", err)
			} else {
				exposePorts = append(exposePorts, int32(port))
			}
		}
	}
	return exposePorts
}

// GetPortForService returns port used by a service
func GetPortForService(detectedPorts []int32, serviceName string) int32 {
	var detectedPortsStr []string
	var exposePortStr string
	var exposePort int32
	if len(detectedPorts) != 0 {
		for _, detectedPort := range detectedPorts {
			detectedPortsStr = append(detectedPortsStr, strconv.Itoa(int(detectedPort)))
		}
		allDetectedPortsStr := append(detectedPortsStr, qatypes.OtherAnswer)
		exposePortStr = qaengine.FetchSelectAnswer(common.ConfigServicesKey+common.Delim+serviceName+common.Delim+common.ConfigPortForServiceKeySegment, fmt.Sprintf("Select port to be exposed for the service %s :", serviceName), []string{fmt.Sprintf("Select Other if you want to expose the service %s to some other port", serviceName)}, allDetectedPortsStr[0], allDetectedPortsStr)
	} else {
		exposePortStr = qaengine.FetchStringAnswer(common.ConfigServicesKey+common.Delim+serviceName+common.Delim+common.ConfigPortForServiceKeySegment, fmt.Sprintf("Enter the port to be exposed for the service %s: ", serviceName), []string{fmt.Sprintf("The service %s will be exposed to the specified port", serviceName)}, fmt.Sprintf("%d", common.DefaultServicePort))
	}
	exposePortStr = strings.TrimSpace(exposePortStr)
	if exposePortStr != "" {
		port, err := strconv.ParseInt(exposePortStr, 10, 32)
		if err != nil {
			logrus.Errorf("Error while converting the selected port from string to int : %s", err)
		}
		return int32(port)
	}
	return exposePort
}

// GetContainerRuntime returns the container runtime
func GetContainerRuntime() string {
	containerRuntimes := []string{"docker", "podman"}
	return qaengine.FetchSelectAnswer(common.ConfigContainerRuntimeKey, "Select the container runtime to use :", []string{"The container runtime selected will be used in the scripts"}, containerRuntimes[0], containerRuntimes)
}
