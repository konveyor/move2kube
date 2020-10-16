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
	"github.com/docker/cli/cli/config/types"
	dockerclitypes "github.com/docker/cli/cli/config/types"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

const otherRegistry = "Other"

// registryCustomizer customizes image registry related configurations
type registryCustomizer struct {
}

// customize modifies image paths and secret
func (rc *registryCustomizer) customize(ir *irtypes.IR) error {

	usedRegistries := []string{}
	registryList := []string{otherRegistry}
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

	if ir.Kubernetes.RegistryURL == "" && len(newimages) != 0 {
		if !common.IsStringPresent(registryList, common.DefaultRegistryURL) {
			registryList = append(registryList, common.DefaultRegistryURL)
		}
		if defreg == "" {
			defreg = common.DefaultRegistryURL
		}
		problem, err := qatypes.NewSelectProblem("Select the registry where your images are hosted:", []string{"You can always change it later by changing the yamls."}, defreg, registryList)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}
		problem, err = qaengine.FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}
		reg, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}
		if reg != otherRegistry {
			ir.Kubernetes.RegistryURL = reg
		}
	}

	if ir.Kubernetes.RegistryURL == "" && len(newimages) != 0 {
		problem, err := qatypes.NewInputProblem("Enter the name of the registry : ", []string{"Ex : " + common.DefaultRegistryURL}, common.DefaultRegistryURL)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}
		problem, err = qaengine.FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}
		reg, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}
		if reg != "" {
			ir.Kubernetes.RegistryURL = reg
		} else {
			ir.Kubernetes.RegistryURL = common.DefaultRegistryURL
		}
	}

	if ir.Kubernetes.RegistryNamespace == "" && len(newimages) != 0 {
		problem, err := qatypes.NewInputProblem("Enter the namespace where the new images are pushed : ", []string{"Ex : " + ir.Name}, ir.Name)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}
		problem, err = qaengine.FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}
		ns, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}
		if ns != "" {
			ir.Kubernetes.RegistryNamespace = ns
		} else {
			ir.Kubernetes.RegistryNamespace = ir.Name
		}
	}

	if !common.IsStringPresent(usedRegistries, ir.Kubernetes.RegistryURL) {
		usedRegistries = append(usedRegistries, ir.Kubernetes.RegistryURL)
	}

	imagePullSecrets := map[string]string{} // registryurl, pull secret

	for _, registry := range usedRegistries {
		dauth := dockerclitypes.AuthConfig{}
		const dockerConfigLogin = "Docker login from config"
		const noAuthLogin = "No authentication"
		const userLogin = "UserName/Password"
		const useExistingPullSecret = "Use existing pull secret"
		authOptions := []string{useExistingPullSecret, noAuthLogin, userLogin}
		if auth, ok := registryAuthList[ir.Kubernetes.RegistryURL]; ok {
			imagePullSecrets[registry] = common.ImagePullSecretPrefix + common.MakeFileNameCompliant(imagePullSecrets[registry])
			dauth.Auth = auth
			authOptions = append(authOptions, dockerConfigLogin)
		}

		problem, err := qatypes.NewSelectProblem(fmt.Sprintf("[%s] What type of container registry login do you want to use?", registry), []string{"Docker login from config mode, will use the default config from your local machine."}, noAuthLogin, authOptions)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}
		problem, err = qaengine.FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}
		auth, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}
		if auth == noAuthLogin {
			dauth.Auth = ""
		} else if auth == useExistingPullSecret {
			problem, err := qatypes.NewInputProblem(fmt.Sprintf("[%s] Enter the name of the pull secret : ", registry), []string{"The pull secret should exist in the namespace where you will be deploying the application."}, "")
			if err != nil {
				log.Fatalf("Unable to create problem : %s", err)
			}
			problem, err = qaengine.FetchAnswer(problem)
			if err != nil {
				log.Fatalf("Unable to fetch answer : %s", err)
			}
			ps, err := problem.GetStringAnswer()
			if err != nil {
				log.Fatalf("Unable to get answer : %s", err)
			}
			imagePullSecrets[registry] = ps
		} else if auth != dockerConfigLogin {
			problem, err := qatypes.NewInputProblem(fmt.Sprintf("[%s] Enter the container registry username : ", registry), []string{"Enter username for container registry login"}, "iamapikey")
			if err != nil {
				log.Fatalf("Unable to create problem : %s", err)
			}
			problem, err = qaengine.FetchAnswer(problem)
			if err != nil {
				log.Fatalf("Unable to fetch answer : %s", err)
			}
			un, err := problem.GetStringAnswer()
			if err != nil {
				log.Fatalf("Unable to get answer : %s", err)
			}
			dauth.Username = un
			problem, err = qatypes.NewPasswordProblem(fmt.Sprintf("[%s] Enter the container registry password : ", registry), []string{"Enter password for container registry login."})
			if err != nil {
				log.Fatalf("Unable to create problem : %s", err)
			}
			problem, err = qaengine.FetchAnswer(problem)
			if err != nil {
				log.Fatalf("Unable to fetch answer : %s", err)
			}
			pwd, err := problem.GetStringAnswer()
			if err != nil {
				log.Fatalf("Unable to get answer : %s", err)
			}
			dauth.Password = pwd
		}
		if dauth != (types.AuthConfig{}) {
			dconfigfile := dockercliconfigfile.ConfigFile{
				AuthConfigs: map[string]dockerclitypes.AuthConfig{ir.Kubernetes.RegistryURL: dauth},
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

	ir.Values.RegistryNamespace = ir.Kubernetes.RegistryNamespace
	ir.Values.RegistryURL = ir.Kubernetes.RegistryURL
	for _, service := range ir.Services {
		for i, serviceContainer := range service.Containers {
			if common.IsStringPresent(newimages, serviceContainer.Image) {
				parts := strings.Split(serviceContainer.Image, "/")
				image, tag := common.GetImageNameAndTag(parts[len(parts)-1])
				if ir.Kubernetes.RegistryURL != "" && ir.Kubernetes.RegistryNamespace != "" {
					serviceContainer.Image = ir.Kubernetes.RegistryURL + "/" + ir.Kubernetes.RegistryNamespace + "/" + image + ":" + tag
				} else if ir.Kubernetes.RegistryNamespace != "" {
					serviceContainer.Image = ir.Kubernetes.RegistryNamespace + "/" + image + ":" + tag
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
						service.ImagePullSecrets = append(service.ImagePullSecrets, corev1.LocalObjectReference{Name: ps})
					}
				}
			}
		}
		ir.Services[service.Name] = service
	}
	return nil
}
