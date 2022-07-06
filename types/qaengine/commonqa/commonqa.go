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
	// DefaultRegistryURL points to the default registry url that will be used
	defaultRegistryURL := "quay.io"
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
	if !common.IsStringPresent(registryList, defaultRegistryURL) {
		registryList = append(registryList, defaultRegistryURL)
	}
	if defreg == "" {
		defreg = defaultRegistryURL
	}
	return qaengine.FetchSelectAnswer(common.ConfigImageRegistryURLKey, "Enter the URL of the image registry : ", []string{"You can always change it later by changing the yamls."}, defreg, registryList)
}

// ImageRegistryNamespace returns Image Registry Namespace
func ImageRegistryNamespace() string {
	return qaengine.FetchStringAnswer(common.ConfigImageRegistryNamespaceKey, "Enter the namespace where the new images should be pushed : ", []string{"Ex : " + common.ProjectName}, common.ProjectName)
}

// IngressHost returns Ingress host
func IngressHost(defaulthost string, clusterQaLabel string) string {
	key := common.ConfigTargetKey + common.Delim + clusterQaLabel + common.Delim + common.ConfigIngressHostKey
	return qaengine.FetchStringAnswer(key, "Provide the ingress host domain", []string{"Ingress host domain is part of service URL"}, defaulthost)
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
		quesKey := common.JoinQASubKeys(common.ConfigServicesKey, serviceName, common.ConfigPortsForServiceKeySegment)
		desc := fmt.Sprintf("Select ports to be exposed for the service %s :", serviceName)
		hints := []string{"Select Other if you want to add more ports"}
		selectedPortsStr = qaengine.FetchMultiSelectAnswer(quesKey, desc, hints, detectedPortsStr, allDetectedPortsStr)
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

// GetPortForService returns the port to expose the service on.
func GetPortForService(detectedPorts []int32, serviceName string) int32 {
	quesKey := common.JoinQASubKeys(common.ConfigServicesKey, serviceName, common.ConfigPortForServiceKeySegment)
	desc := fmt.Sprintf("Select the port to be exposed for the service %s :", serviceName)
	hints := []string{"Select 'Other' if you want to expose the service using a different port."}
	detectedPortStrs := []string{}
	for _, detectedPort := range detectedPorts {
		detectedPortStrs = append(detectedPortStrs, fmt.Sprintf("%d", detectedPort))
	}
	if len(detectedPortStrs) == 0 {
		detectedPortStrs = append(detectedPortStrs, fmt.Sprintf("%d", common.DefaultServicePort))
	}
	detectedPortStrs = append(detectedPortStrs, qatypes.OtherAnswer)
	selectedPortStr := qaengine.FetchSelectAnswer(quesKey, desc, hints, detectedPortStrs[0], detectedPortStrs)
	selectedPortStr = strings.TrimSpace(selectedPortStr)
	selectedPort, err := strconv.ParseInt(selectedPortStr, 10, 32)
	if err != nil {
		logrus.Errorf("got the string '%s' which is not a valid integer/port. Error: %q", selectedPortStr, err)
		return common.DefaultServicePort
	}
	if selectedPort < 0 || selectedPort > 65535 {
		logrus.Errorf("got the integer '%d' which is outside the range for a valid port (0 to 65535).", selectedPort)
		return common.DefaultServicePort
	}
	return int32(selectedPort)
}

// GetContainerRuntime returns the container runtime
func GetContainerRuntime() string {
	containerRuntimes := []string{"docker", "podman"}
	return qaengine.FetchSelectAnswer(common.ConfigContainerRuntimeKey, "Select the container runtime to use :", []string{"The container runtime selected will be used in the scripts"}, containerRuntimes[0], containerRuntimes)
}
