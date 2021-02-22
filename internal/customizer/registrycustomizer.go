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
	"bytes"
	"fmt"
	"net/url"
	"strings"

	dockercliconfig "github.com/docker/cli/cli/config"
	dockercliconfigfile "github.com/docker/cli/cli/config/configfile"
	dockerclitypes "github.com/docker/cli/cli/config/types"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// registryCustomizer customizes image registry related configurations
type registryCustomizer struct {
}

// customize modifies image paths and secret
func (rc *registryCustomizer) customize(ir *irtypes.IR) error {

	usedRegistries := []string{}
	registryList := []string{qatypes.OtherAnswer}
	newimages := []string{}

	for _, container := range ir.Containers {
		if container.New {
			newimages = append(newimages, container.ImageNames...)
		}
	}

	for _, service := range ir.Services {
		for _, container := range service.Containers {
			if !common.IsStringPresent(newimages, container.Image) {
				parts := strings.Split(container.Image, "/")
				if len(parts) == 3 {
					registryList = append(registryList, parts[0])
					usedRegistries = append(usedRegistries, parts[0])
				}
			}
		}
	}

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

	if ir.RegistryURL == "" && len(newimages) != 0 {
		if !common.IsStringPresent(registryList, common.DefaultRegistryURL) {
			registryList = append(registryList, common.DefaultRegistryURL)
		}
		if defreg == "" {
			defreg = common.DefaultRegistryURL
		}
		reg := qaengine.FetchSelectAnswer(common.ConfigImageRegistryURLKey, "Enter the name of the image registry : ", []string{"You can always change it later by changing the yamls."}, defreg, registryList)
		if reg != "" {
			ir.RegistryURL = reg
		} else {
			ir.RegistryURL = common.DefaultRegistryURL
		}
	}

	if ir.RegistryNamespace == "" && len(newimages) != 0 {
		ns := qaengine.FetchStringAnswer(common.ConfigImageRegistryNamespaceKey, "Enter the namespace where the new images should be pushed : ", []string{"Ex : " + ir.Name}, ir.Name)
		if ns != "" {
			ir.RegistryNamespace = ns
		} else {
			ir.RegistryNamespace = ir.Name
		}
	}

	if !common.IsStringPresent(usedRegistries, ir.RegistryURL) {
		usedRegistries = append(usedRegistries, ir.RegistryURL)
	}

	imagePullSecrets := map[string]string{} // registryurl, pull secret

	for _, registry := range usedRegistries {
		dauth := dockerclitypes.AuthConfig{}
		const dockerConfigLogin = "Docker login from config"
		const noAuthLogin = "No authentication"
		const userLogin = "UserName/Password"
		const useExistingPullSecret = "Use existing pull secret"
		authOptions := []string{useExistingPullSecret, noAuthLogin, userLogin}
		if auth, ok := registryAuthList[ir.RegistryURL]; ok {
			imagePullSecrets[registry] = common.ImagePullSecretPrefix + common.MakeFileNameCompliant(imagePullSecrets[registry])
			dauth.Auth = auth
			authOptions = append(authOptions, dockerConfigLogin)
		}

		auth := qaengine.FetchSelectAnswer(common.ConfigImageRegistryLoginTypeKey, fmt.Sprintf("[%s] What type of container registry login do you want to use?", registry), []string{"Docker login from config mode, will use the default config from your local machine."}, noAuthLogin, authOptions)
		if auth == noAuthLogin {
			dauth.Auth = ""
		} else if auth == useExistingPullSecret {
			ps := qaengine.FetchStringAnswer(common.ConfigImageRegistryPullSecretKey, fmt.Sprintf("[%s] Enter the name of the pull secret : ", registry), []string{"The pull secret should exist in the namespace where you will be deploying the application."}, "")
			imagePullSecrets[registry] = ps
		} else if auth != dockerConfigLogin {
			un := qaengine.FetchStringAnswer(common.ConfigImageRegistryUserNameKey, fmt.Sprintf("[%s] Enter the container registry username : ", registry), []string{"Enter username for container registry login"}, "iamapikey")
			dauth.Username = un
			dauth.Password = qaengine.FetchPasswordAnswer(common.ConfigImageRegistryPasswordKey, fmt.Sprintf("[%s] Enter the container registry password : ", registry), []string{"Enter password for container registry login."})
		}
		if dauth != (dockerclitypes.AuthConfig{}) {
			dconfigfile := dockercliconfigfile.ConfigFile{
				AuthConfigs: map[string]dockerclitypes.AuthConfig{ir.RegistryURL: dauth},
			}
			dconfigbuffer := new(bytes.Buffer)
			err := dconfigfile.SaveToWriter(dconfigbuffer)
			if err == nil {
				data := map[string][]byte{}
				data[".dockerconfigjson"] = dconfigbuffer.Bytes()
				ir.AddStorage(irtypes.Storage{
					Name:        imagePullSecrets[registry],
					StorageType: irtypes.PullSecretKind,
					Content:     data,
				})
			} else {
				log.Warnf("Unable to create auth string : %s", err)
			}
		}
	}

	for si, service := range ir.Services {
		for i, serviceContainer := range service.Containers {
			if common.IsStringPresent(newimages, serviceContainer.Image) {
				image, tag := common.GetImageNameAndTag(serviceContainer.Image)
				if ir.RegistryURL != "" && ir.RegistryNamespace != "" {
					serviceContainer.Image = ir.RegistryURL + "/" + ir.RegistryNamespace + "/" + image + ":" + tag
				} else if ir.RegistryNamespace != "" {
					serviceContainer.Image = ir.RegistryNamespace + "/" + image + ":" + tag
				} else {
					serviceContainer.Image = image + ":" + tag
				}
				service.Containers[i] = serviceContainer
			}
			parts := strings.Split(serviceContainer.Image, "/")
			if len(parts) == 3 {
				reg := parts[0]
				if ps, ok := imagePullSecrets[reg]; ok {
					found := false
					for _, eps := range service.ImagePullSecrets {
						if eps.Name == ps {
							found = true
						}
					}
					if !found {
						service.ImagePullSecrets = append(service.ImagePullSecrets, core.LocalObjectReference{Name: ps})
					}
				}
			}
		}
		ir.Services[si] = service
	}
	return nil
}
