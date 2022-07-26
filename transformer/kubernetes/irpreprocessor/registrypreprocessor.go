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

// registryPreProcessor preprocesses image registry related configurations
type registryPreProcessor struct {
}

type registryLoginOption string

const (
	noLogin                 registryLoginOption = "no authentication"
	usernamePasswordLogin   registryLoginOption = "username and password"
	existingPullSecretLogin registryLoginOption = "use an existing pull secret"
	dockerConfigLogin       registryLoginOption = "use the credentials from the docker config.json file"
)

const (
	// imagePullSecretSuffix is the suffix that will be appended to pull secret name
	imagePullSecretSuffix = "-imagepullsecret"
)

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
			if !common.IsPresent(newimages, container.Image) {
				parts := strings.Split(container.Image, "/")
				if len(parts) == 3 {
					registryList = append(registryList, parts[0])
					usedRegistries = append(usedRegistries, parts[0])
				}
			}
		}
	}
	reg := commonqa.ImageRegistry()
	usedRegistries = common.AppendIfNotPresent(usedRegistries, reg)
	imagePullSecrets := map[string]string{}                    // registryurl, pull secret
	registryAuthList := map[string]dockerclitypes.AuthConfig{} //Registry url and auth
	if !common.IgnoreEnvironment {
		defaultDockerConfigDir := dockercliconfig.Dir()
		configFile, err := dockercliconfig.Load(defaultDockerConfigDir)
		if err != nil {
			logrus.Errorf("failed to load the docker config.json file in the directory %s . Error: %q", defaultDockerConfigDir, err)
		} else {
			for regurl, regauth := range configFile.AuthConfigs {
				u, err := url.Parse(regurl)
				if err != nil {
					logrus.Errorf("failed to parse the string '%s' as a url. Error: %q", regurl, err)
					continue
				}
				if u.Host == "" {
					if u.Scheme != "" {
						continue
					}
					u, err = url.Parse("http://" + regurl)
					if err != nil || u.Host == "" {
						continue
					}
				}
				registryList = common.AppendIfNotPresent(registryList, u.Host)
				registryAuthList[regurl] = regauth
			}
		}
	}
	ns := commonqa.ImageRegistryNamespace()
	for _, registry := range usedRegistries {
		if _, ok := imagePullSecrets[registry]; !ok {
			imagePullSecrets[registry] = common.NormalizeForMetadataName(strings.ReplaceAll(registry, ".", "-") + imagePullSecretSuffix)
		}
		regAuth := dockerclitypes.AuthConfig{}
		authOptions := []string{string(existingPullSecretLogin), string(noLogin), string(usernamePasswordLogin)}
		defaultOption := noLogin
		if auth, ok := registryAuthList[registry]; ok {
			regAuth = auth
			authOptions = append(authOptions, string(dockerConfigLogin))
			defaultOption = dockerConfigLogin
		}
		quesKey := fmt.Sprintf(common.ConfigImageRegistryLoginTypeKey, `"`+registry+`"`)
		desc := fmt.Sprintf("[%s] What type of container registry login do you want to use?", registry)
		hints := []string{"Docker login from config mode, will use the default config from your local machine."}
		auth := qaengine.FetchSelectAnswer(quesKey, desc, hints, string(defaultOption), authOptions)
		switch registryLoginOption(auth) {
		case noLogin:
			regAuth.Auth = ""
		case existingPullSecretLogin:
			qaKey := fmt.Sprintf(common.ConfigImageRegistryPullSecretKey, `"`+registry+`"`)
			ps := qaengine.FetchStringAnswer(qaKey, fmt.Sprintf("[%s] Enter the name of the pull secret : ", registry), []string{"The pull secret should exist in the namespace where you will be deploying the application."}, "")
			imagePullSecrets[registry] = ps
		case usernamePasswordLogin:
			qaUsernameKey := fmt.Sprintf(common.ConfigImageRegistryUserNameKey, `"`+registry+`"`)
			regAuth.Username = qaengine.FetchStringAnswer(qaUsernameKey, fmt.Sprintf("[%s] Enter the username to login into the registry : ", registry), nil, "iamapikey")
			qaPasswordKey := fmt.Sprintf(common.ConfigImageRegistryPasswordKey, `"`+registry+`"`)
			regAuth.Password = qaengine.FetchPasswordAnswer(qaPasswordKey, fmt.Sprintf("[%s] Enter the password to login into the registry : ", registry), nil)
		case dockerConfigLogin:
			logrus.Debugf("using the credentials from the docker config.json file")
		}
		if regAuth != (dockerclitypes.AuthConfig{}) {
			// create a valid docker config json file using the credentials
			dconfigfile := dockercliconfigfile.ConfigFile{AuthConfigs: map[string]dockerclitypes.AuthConfig{registry: regAuth}}
			dconfigbuffer := new(bytes.Buffer)
			if err := dconfigfile.SaveToWriter(dconfigbuffer); err == nil {
				ir.AddStorage(irtypes.Storage{
					Name:        imagePullSecrets[registry],
					StorageType: irtypes.PullSecretKind,
					Content:     map[string][]byte{".dockerconfigjson": dconfigbuffer.Bytes()},
				})
			} else {
				logrus.Warnf("Unable to create auth string : %s", err)
			}
		}
	}

	for sn, service := range ir.Services {
		for i, serviceContainer := range service.Containers {
			if common.IsPresent(newimages, serviceContainer.Image) {
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
