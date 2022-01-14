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
	"bytes"
	"fmt"
	"net/url"
	"strings"

	dockercliconfig "github.com/docker/cli/cli/config"
	dockercliconfigfile "github.com/docker/cli/cli/config/configfile"
	dockerclitypes "github.com/docker/cli/cli/config/types"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	// imagePullSecretSuffix is the suffix that will be appended to pull secret name
	imagePullSecretSuffix = "-imagepullsecret"
)

// registryPreProcessor preprocesses image registry related configurations
type registryPreProcessor struct {
}

func (p registryPreProcessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	usedRegistries := []string{}
	registryList := []string{qatypes.OtherAnswer}
	newimages := []string{}
	for in, container := range ir.ContainerImages {
		if container.Build.ContainerBuildType != "" {
			newimages = append(newimages, in)
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
	reg := commonqa.ImageRegistry()
	if !common.IsStringPresent(usedRegistries, reg) {
		usedRegistries = append(usedRegistries, reg)
	}
	imagePullSecrets := map[string]string{} // registryurl, pull secret
	registryAuthList := map[string]string{} //Registry url and auth
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
					registryAuthList[regurl] = regauth.Auth
				}
			}
		}
	}
	ns := commonqa.ImageRegistryNamespace()
	for _, registry := range usedRegistries {
		dauth := dockerclitypes.AuthConfig{}
		const dockerConfigLogin = "Docker login from config"
		const noAuthLogin = "No authentication"
		const userLogin = "UserName/Password"
		const useExistingPullSecret = "Use existing pull secret"
		authOptions := []string{useExistingPullSecret, noAuthLogin, userLogin}
		if _, ok := imagePullSecrets[registry]; !ok {
			imagePullSecrets[registry] = common.NormalizeForMetadataName(registry + imagePullSecretSuffix)
		}
		if auth, ok := registryAuthList[registry]; ok {
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
				AuthConfigs: map[string]dockerclitypes.AuthConfig{registry: dauth},
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
				logrus.Warnf("Unable to create auth string : %s", err)
			}
		}
	}

	for sn, service := range ir.Services {
		for i, serviceContainer := range service.Containers {
			if common.IsStringPresent(newimages, serviceContainer.Image) {
				image, tag := common.GetImageNameAndTag(serviceContainer.Image)
				if reg != "" && ns != "" {
					serviceContainer.Image = reg + "/" + ns + "/" + image + ":" + tag
				} else if ns != "" {
					serviceContainer.Image = ns + "/" + image + ":" + tag
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
		ir.Services[sn] = service
	}
	return ir, nil
}
