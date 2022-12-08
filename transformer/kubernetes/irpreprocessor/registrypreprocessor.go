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
	// find all the new images that we are going to create

	newImageNames := []string{}
	for imageName, container := range ir.ContainerImages {
		if container.Build.ContainerBuildType != "" {
			newImageNames = append(newImageNames, imageName)
		}
	}

	// find all the registries that we use for our images

	usedRegistries := []string{}
	for _, service := range ir.Services {
		for _, container := range service.Containers {
			if !common.IsPresent(newImageNames, container.Image) {

				// if it's a pre-existing image then find the registry where the image exists

				parts := strings.Split(container.Image, "/")
				if len(parts) == 3 {
					usedRegistries = append(usedRegistries, parts[0])
				}
			}
		}
	}

	// ask the user for the registry url where new images should be pushed

	registryToPushImagesTo := commonqa.ImageRegistry()
	usedRegistries = common.AppendIfNotPresent(usedRegistries, registryToPushImagesTo)

	// get the login credentials for each registry we use by parsing the docker config.json file

	registryAuthList := map[string]dockerclitypes.AuthConfig{} // registry url -> login credentials
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
				registryAuthList[regurl] = regauth
			}
		}
	}

	// ask the user what type of login to use for each registry that we use

	imagePullSecrets := map[string]string{} // registry url -> pull secret name
	registryNamespace := commonqa.ImageRegistryNamespace()
	useExistingPullSecret := false
	for _, registry := range usedRegistries {
		if _, ok := imagePullSecrets[registry]; !ok {
			imagePullSecrets[registry] = common.NormalizeForMetadataName(strings.ReplaceAll(registry, ".", "-") + imagePullSecretSuffix)
		}
		regAuth := dockerclitypes.AuthConfig{}
		authOptions := []string{string(existingPullSecretLogin), string(noLogin), string(usernamePasswordLogin)}
		defaultOption := noLogin
		if auth, err := fuzzyMatch(registry, registryAuthList); err == nil {

			// if we find the credentials for the registry in the docker config.json then use that as the default option

			regAuth = auth
			authOptions = append(authOptions, string(dockerConfigLogin))
			defaultOption = dockerConfigLogin
		}
		quesKey := fmt.Sprintf(common.ConfigImageRegistryLoginTypeKey, `"`+registry+`"`)
		desc := fmt.Sprintf("[%s] What type of container registry login do you want to use?", registry)
		hints := []string{"Docker login from config mode, will use the default config from your local machine."}
		auth := qaengine.FetchSelectAnswer(quesKey, desc, hints, string(defaultOption), authOptions, nil)
		createPullSecret := false
		switch registryLoginOption(auth) {
		case noLogin:
			regAuth.Auth = ""
		case existingPullSecretLogin:
			useExistingPullSecret = true
			qaKey := fmt.Sprintf(common.ConfigImageRegistryPullSecretKey, `"`+registry+`"`)
			ps := qaengine.FetchStringAnswer(qaKey, fmt.Sprintf("[%s] Enter the name of the pull secret : ", registry), []string{"The pull secret should exist in the namespace where you will be deploying the application."}, "", nil)
			imagePullSecrets[registry] = ps
		case usernamePasswordLogin:
			createPullSecret = true
			qaUsernameKey := fmt.Sprintf(common.ConfigImageRegistryUserNameKey, `"`+registry+`"`)
			regAuth.Username = qaengine.FetchStringAnswer(qaUsernameKey, fmt.Sprintf("[%s] Enter the username to login into the registry : ", registry), nil, "iamapikey", nil)
			qaPasswordKey := fmt.Sprintf(common.ConfigImageRegistryPasswordKey, `"`+registry+`"`)
			regAuth.Password = qaengine.FetchPasswordAnswer(qaPasswordKey, fmt.Sprintf("[%s] Enter the password to login into the registry : ", registry), nil, nil)
		case dockerConfigLogin:
			createPullSecret = true
			logrus.Debugf("using the credentials from the docker config.json file")
		}
		if createPullSecret {

			// create a valid docker config.json file using the credentials

			configFile := dockercliconfigfile.ConfigFile{AuthConfigs: map[string]dockerclitypes.AuthConfig{registry: regAuth}}
			configFileContents := new(bytes.Buffer)
			if err := configFile.SaveToWriter(configFileContents); err != nil {
				logrus.Warnf("failed to create auth string. Error: %q", err)
			} else {
				ir.AddStorage(irtypes.Storage{
					Name:        imagePullSecrets[registry],
					StorageType: irtypes.PullSecretKind,
					Content:     map[string][]byte{".dockerconfigjson": configFileContents.Bytes()},
				})
			}
		}
	}

	for serviceName, service := range ir.Services {
		for i, container := range service.Containers {
			if common.IsPresent(newImageNames, container.Image) {
				image, tag := common.GetImageNameAndTag(container.Image)
				if registryToPushImagesTo != "" && registryNamespace != "" {
					container.Image = registryToPushImagesTo + "/" + registryNamespace + "/" + image + ":" + tag
				} else if registryNamespace != "" {
					container.Image = registryNamespace + "/" + image + ":" + tag
				} else {
					container.Image = image + ":" + tag
				}
				service.Containers[i] = container
			}
			parts := strings.Split(container.Image, "/")
			if len(parts) != 3 {
				continue
			}
			reg := parts[0]
			pullSecretName, ok := imagePullSecrets[reg]
			if !ok {
				continue
			}
			found := false
			for _, eps := range service.ImagePullSecrets {
				if eps.Name == pullSecretName {
					found = true
				}
			}
			if !found {
				if useExistingPullSecret {
					service.ImagePullSecrets = append(service.ImagePullSecrets, core.LocalObjectReference{Name: pullSecretName})
				}
			}
		}
		ir.Services[serviceName] = service
	}
	return ir, nil
}

func fuzzyMatch(regUrl string, regAuthMap map[string]dockerclitypes.AuthConfig) (dockerclitypes.AuthConfig, error) {
	for k, v := range regAuthMap {
		if strings.EqualFold(k, regUrl) {
			return v, nil
		}
	}
	if regUrl == "docker.io" {
		if v, ok := regAuthMap["index.docker.io"]; ok {
			return v, nil
		}
	} else if regUrl == "index.docker.io" {
		if v, ok := regAuthMap["docker.io"]; ok {
			return v, nil
		}
	}
	return dockerclitypes.AuthConfig{}, fmt.Errorf("failed to find the creds for the registry url '%s'", regUrl)
}
