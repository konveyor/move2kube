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

package transformer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"code.cloudfoundry.org/cli/util/manifest"
	"github.com/cloudfoundry/bosh-cli/director/template"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
	core "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/networking"
)

// VariableLiteralPattern to identify variable literals in environment names
var VariableLiteralPattern = regexp.MustCompile("[-.+~`!@#$%^&*(){}\\[\\]:;\"',?<>/]")

// CloudFoundry implements Transformer interface
type CloudFoundry struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// VCAPService defines the VCAP service data from JSON
type VCAPService struct {
	ServiceName        string                 `json:"name"`
	ServiceCredentials map[string]interface{} `json:"credentials"`
}

// Init Initializes the transformer
func (t *CloudFoundry) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *CloudFoundry) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect detects cloud foundry projects in various directories
func (t *CloudFoundry) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	services = map[string][]transformertypes.Artifact{}
	filePaths, err := common.GetFilesByExt(dir, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", dir, err)
		return services, err
	}
	services = map[string][]transformertypes.Artifact{}
	// Load instance apps, if available
	cfInstanceApps := map[string][]collecttypes.CfApp{} //path
	for _, filePath := range filePaths {
		fileCfInstanceApps := collecttypes.CfApps{}
		if err := common.ReadMove2KubeYaml(filePath, &fileCfInstanceApps); err != nil {
			logrus.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		if fileCfInstanceApps.Kind != string(collecttypes.CfAppsMetadataKind) {
			logrus.Debugf("%q is not a valid apps file. Expected kind: %s Actual Kind: %s", filePath, string(collecttypes.CfAppsMetadataKind), fileCfInstanceApps.Kind)
			continue
		}
		cfInstanceApps[filePath] = append(cfInstanceApps[filePath], fileCfInstanceApps.Spec.CfApps...)
	}
	logrus.Debugf("Cf Instances %+v", cfInstanceApps)
	for _, filePath := range filePaths {
		applications, _, err := t.readApplicationManifest(filePath, "")
		if err != nil {
			logrus.Debugf("Failed to parse the manifest file at path %q Error: %q", filePath, err)
			continue
		}
		for _, application := range applications {
			servicedirectory := filepath.Dir(filePath)
			buildArtifactDirectory := ""
			if application.Path != "" {
				artifactDirectory := filepath.Join(filepath.Dir(filePath), application.Path)
				if _, err := os.Stat(artifactDirectory); !os.IsNotExist(err) {
					servicedirectory = artifactDirectory
				} else {
					buildArtifactDirectory = artifactDirectory
					logrus.Debugf("Path to app directory %s does not exist, assuming manifest directory as app path", artifactDirectory)
				}
			}
			applicationName := application.Name
			if applicationName == "" {
				basename := filepath.Base(filePath)
				applicationName = strings.TrimSuffix(basename, filepath.Ext(basename))
			}
			ct := transformertypes.Artifact{
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.CloudFoundryConfigType: artifacts.CloudFoundryConfig{
						ServiceName: applicationName,
					}},
				Paths: map[transformertypes.PathType][]string{
					artifacts.CfManifestPathType: {filePath},
					artifacts.ServiceDirPathType: {servicedirectory},
				},
			}
			if buildArtifactDirectory != "" {
				ct.Paths[artifacts.BuildArtifactPathType] = []string{buildArtifactDirectory}
			}
			containerizationOptions := getContainerizationOptions(servicedirectory)
			if len(containerizationOptions) != 0 {
				ct.Configs[artifacts.ContainerizationOptionsConfigType] = artifacts.ContainerizationOptionsConfig(containerizationOptions)
			}
			runningManifestPath, appinstance := getCfInstanceApp(cfInstanceApps, applicationName)
			if application.DockerImage != "" || appinstance.Application.DockerImage != "" {
				dockerImageName := application.DockerImage
				if dockerImageName == "" {
					dockerImageName = appinstance.Application.DockerImage
				}
				ctConfig := ct.Configs[artifacts.CloudFoundryConfigType].(artifacts.CloudFoundryConfig)
				ctConfig.ImageName = dockerImageName
				ct.Configs[artifacts.CloudFoundryConfigType] = ctConfig
			}
			if runningManifestPath != "" {
				ct.Paths[artifacts.CfRunningManifestPathType] = append(ct.Paths[artifacts.CfRunningManifestPathType], runningManifestPath)
			}
			services[applicationName] = []transformertypes.Artifact{ct}
		}
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *CloudFoundry) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		var config artifacts.CloudFoundryConfig
		err := a.GetConfig(artifacts.CloudFoundryConfigType, &config)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", config, err)
			continue
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		var cConfig artifacts.ContainerizationOptionsConfig
		err = a.GetConfig(artifacts.ContainerizationOptionsConfigType, &cConfig)
		if err != nil {
			logrus.Debugf("Unable to get containerization config : %s", err)
		}
		ir := irtypes.NewIR()
		var cfinstanceapp collecttypes.CfApp
		logrus.Debugf("Transforming %s", config.ServiceName)
		if runninginstancefile, ok := a.Paths[artifacts.CfRunningManifestPathType]; ok {
			var err error
			cfinstanceapp, err = getCfAppInstance(runninginstancefile[0], config.ServiceName)
			if err != nil {
				logrus.Debugf("The file at path %s is not a valid cf apps file. Error: %q", runninginstancefile[0], err)
			}
		}
		if paths, ok := a.Paths[artifacts.CfManifestPathType]; ok {
			path := paths[0] // TODO: what about the rest of the manifests?
			applications, _, err := t.readApplicationManifest(path, config.ServiceName)
			if err != nil {
				logrus.Debugf("Error while trying to parse manifest : %s", err)
				continue
			}
			logrus.Debugf("Using cf manifest file at path %s to transform service %s", path, config.ServiceName)
			application := applications[0]
			serviceConfig := irtypes.Service{Name: sConfig.ServiceName}
			serviceContainer := core.Container{Name: sConfig.ServiceName}
			serviceContainer.Image = config.ImageName
			if serviceContainer.Image == "" {
				serviceContainer.Image = sConfig.ServiceName
			}
			//TODO: Add support for services, health check, memory
			if application.Instances.IsSet {
				serviceConfig.Replicas = application.Instances.Value
			} else if cfinstanceapp.Application.Instances != 0 {
				serviceConfig.Replicas = cfinstanceapp.Application.Instances
			}
			serviceContainer.Env = append(serviceContainer.Env,
				t.prioritizeAndAddEnvironmentVariables(cfinstanceapp, application.EnvironmentVariables)...)
			ports := cfinstanceapp.Application.Ports
			if len(ports) == 0 {
				ports = []int{int(common.DefaultServicePort)}
				cfinstanceapp.Application.Ports = ports
			}
			for _, port := range cfinstanceapp.Application.Ports {
				// Add the port to the k8s pod.
				serviceContainer.Ports = append(serviceContainer.Ports, core.ContainerPort{ContainerPort: int32(port)})
				// Forward the port on the k8s service to the k8s pod.
				podPort := networking.ServiceBackendPort{Number: int32(port)}
				servicePort := podPort
				serviceConfig.AddPortForwarding(servicePort, podPort, "")
			}
			envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(ports[0])}
			serviceContainer.Env = append(serviceContainer.Env, envvar)
			serviceConfig.Containers = []core.Container{serviceContainer}
			ir.Services[sConfig.ServiceName] = serviceConfig
		}
		if len(cConfig) != 0 {
			containerizationOption := qaengine.FetchSelectAnswer(common.ConfigServicesKey+common.Delim+sConfig.ServiceName+common.Delim+common.ConfigContainerizationOptionServiceKeySegment, fmt.Sprintf("Select the transformer to use for containerization %s :", sConfig.ServiceName), []string{fmt.Sprintf("Select containerization option to use %s", sConfig.ServiceName)}, cConfig[0], cConfig)
			containerizationArtifact := getContainerizationConfig(sConfig.ServiceName, a.Paths[artifacts.ServiceDirPathType], a.Paths[artifacts.BuildArtifactPathType], containerizationOption)
			if containerizationArtifact.Type == "" {
				if config.ImageName == "" {
					logrus.Errorf("No containerization option found for service %s", sConfig.ServiceName)
				}
			} else {
				containerizationArtifact.Name = sConfig.ServiceName
				if containerizationArtifact.Configs == nil {
					containerizationArtifact.Configs = map[transformertypes.ConfigType]interface{}{}
				}
				containerizationArtifact.Configs[irtypes.IRConfigType] = ir
				containerizationArtifact.Configs[artifacts.ServiceConfigType] = sConfig
				artifactsCreated = append(artifactsCreated, containerizationArtifact)
			}
		}
		artifactsCreated = append(artifactsCreated, transformertypes.Artifact{
			Name: t.Env.GetProjectName(),
			Type: irtypes.IRArtifactType,
			Configs: map[transformertypes.ConfigType]interface{}{
				irtypes.IRConfigType: ir,
			},
		})
	}
	return nil, artifactsCreated, nil
}

// prioritizeAndAddEnvironmentVariables adds relevant environment variables relevant to the application deployment
func (t *CloudFoundry) prioritizeAndAddEnvironmentVariables(cfApp collecttypes.CfApp, manifestEnvMap map[string]string) []core.EnvVar {
	envOrderMap := map[string]core.EnvVar{}
	// Manifest
	for varname, value := range manifestEnvMap {
		envOrderMap[varname] = core.EnvVar{Name: varname, Value: value}
	}
	for varname, value := range cfApp.Environment.StagingEnv {
		envOrderMap[varname] = core.EnvVar{Name: varname, Value: fmt.Sprintf("%s", value)}
	}
	for varname, value := range cfApp.Environment.RunningEnv {
		envOrderMap[varname] = core.EnvVar{Name: varname, Value: fmt.Sprintf("%s", value)}
	}
	for varname, value := range cfApp.Environment.SystemEnv {
		valueStr := fmt.Sprintf("%s", value)
		if varname == "VCAP_SERVICES" && valueStr != "" {
			flattenedEnvList := flattenVcapServiceVariables(valueStr)
			for _, env := range flattenedEnvList {
				envOrderMap[env.Name] = env
			}
		}
		envOrderMap[varname] = core.EnvVar{Name: varname, Value: valueStr}
	}
	for varname, value := range cfApp.Environment.ApplicationEnv {
		envOrderMap[varname] = core.EnvVar{Name: varname, Value: fmt.Sprintf("%s", value)}
	}
	for varname, value := range cfApp.Environment.Environment {
		envOrderMap[varname] = core.EnvVar{Name: varname, Value: fmt.Sprintf("%s", value)}
	}
	var envList []core.EnvVar
	for _, value := range envOrderMap {
		envList = append(envList, value)
	}
	return envList
}

// flattenVariable flattens a given variable defined by <name, credential>
func flattenVariable(prefix string, credential interface{}) []core.EnvVar {
	var credentialList []core.EnvVar
	switch cred := credential.(type) {
	case []interface{}:
		for index, value := range cred {
			envName := fmt.Sprintf("%s_%v", prefix, index)
			credentialList = append(credentialList, flattenVariable(envName, value)...)
		}
	case map[string]interface{}:
		for name, value := range cred {
			envName := fmt.Sprintf("%s_%s", prefix, name)
			credentialList = append(credentialList, flattenVariable(envName, value)...)
		}
	default:
		return []core.EnvVar{{Name: strings.ToUpper(VariableLiteralPattern.ReplaceAllLiteralString(prefix, "_")),
			Value: fmt.Sprintf("%#v", credential)}}
	}
	return credentialList
}

// flattenVcapServiceVariables flattens the variables specified in the "credentials" field of VCAP_SERVICES
func flattenVcapServiceVariables(vcapService string) []core.EnvVar {
	var flattenedEnvList []core.EnvVar
	var serviceInstanceMap map[string][]VCAPService
	err := json.Unmarshal([]byte(vcapService), &serviceInstanceMap)
	if err != nil {
		logrus.Errorf("Could not unmarshal the service map instance (%s) in VCAP_SERVICES: %s", vcapService, err)
		return nil
	}
	for _, serviceInstances := range serviceInstanceMap {
		for _, serviceInstance := range serviceInstances {
			flattenedEnvList = append(flattenedEnvList, flattenVariable(serviceInstance.ServiceName, serviceInstance.ServiceCredentials)...)
		}
	}
	return flattenedEnvList
}

// readApplicationManifest reads an application manifest
func (t *CloudFoundry) readApplicationManifest(path string, serviceName string) ([]manifest.Application, []string, error) { // manifest, parameters
	trimmedvariables, err := getMissingVariables(path)
	if err != nil {
		logrus.Debugf("Unable to read as cf manifest %s : %s", path, err)
		return nil, nil, err
	}

	rawManifest, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("Unable to read manifest file at path %q Error: %q", path, err)
		return nil, nil, err
	}
	tpl := template.NewTemplate(rawManifest)
	fileVars := template.StaticVariables{}
	for _, variable := range trimmedvariables {
		fileVars[variable] = "{{ index  .Values " + `"globalvariables" "` + variable + `"}}`
	}
	rawManifest, err = tpl.Evaluate(fileVars, nil, template.EvaluateOpts{ExpectAllKeys: true})
	if err != nil {
		logrus.Debugf("Interpolation Error %s", err)
		return nil, nil, err
	}

	var m manifest.Manifest
	err = yaml.Unmarshal(rawManifest, &m)
	if err != nil {
		logrus.Debugf("UnMarshalling error %s", err)
		return nil, nil, err
	}
	if len(m.Applications) == 1 {
		//If the service name is missing, use the directory name
		return m.Applications, trimmedvariables, nil
	}
	applications := []manifest.Application{}
	if serviceName != "" {
		for _, application := range m.Applications {
			if application.Name == serviceName {
				applications = append(applications, application)
			}
		}
	} else {
		applications = m.Applications
	}
	return applications, trimmedvariables, nil
}

func getMissingVariables(path string) ([]string, error) {
	trimmedvariables := []string{}
	_, err := manifest.ReadAndInterpolateManifest(path, []string{}, []template.VarKV{})
	if err != nil {
		errstring := err.Error()
		if strings.Contains(errstring, "Expected to find variables:") {
			variablesstr := strings.Split(errstring, ":")[1]
			variables := strings.Split(variablesstr, ",")
			for _, variable := range variables {
				trimmedvariables = append(trimmedvariables, strings.TrimSpace(variable))
			}
		} else {
			logrus.Debugf("Error %s", err)
			return []string{}, err
		}
	}
	return trimmedvariables, nil
}

func getCfInstanceApp(fileApps map[string][]collecttypes.CfApp, name string) (string, collecttypes.CfApp) {
	for path, apps := range fileApps {
		for _, app := range apps {
			if app.Application.Name == name {
				return path, app
			}
		}
	}
	return "", collecttypes.CfApp{}
}

func getCfAppInstance(path string, appname string) (collecttypes.CfApp, error) {
	cfinstanceappsfile := collecttypes.CfApps{}
	if err := common.ReadMove2KubeYaml(path, &cfinstanceappsfile); err != nil {
		return collecttypes.CfApp{}, err
	}
	for _, app := range cfinstanceappsfile.Spec.CfApps {
		if app.Application.Name == appname {
			return app, nil
		}
	}
	return collecttypes.CfApp{}, fmt.Errorf("failed to find the app %s in the cf apps file at path %s", appname, path)
}
