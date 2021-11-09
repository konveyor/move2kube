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

package cloudfoundry

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/cli/util/manifest"
	"github.com/cloudfoundry/bosh-cli/director/template"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
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

// CloudFoundry implements Transformer interface
type CloudFoundry struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
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
func (t *CloudFoundry) DirectoryDetect(dir string) (services map[string][]transformertypes.TransformerPlan, err error) {
	services = map[string][]transformertypes.TransformerPlan{}

	filePaths, err := common.GetFilesByExt(dir, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", dir, err)
		return services, err
	}
	services = map[string][]transformertypes.TransformerPlan{}
	// Load instance apps, if available
	cfInstanceApps := map[string][]collecttypes.CfApplication{} //path
	for _, filePath := range filePaths {
		fileCfInstanceApps := collecttypes.CfInstanceApps{}
		if err := common.ReadMove2KubeYaml(filePath, &fileCfInstanceApps); err != nil {
			logrus.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		if fileCfInstanceApps.Kind != string(collecttypes.CfInstanceAppsMetadataKind) {
			logrus.Debugf("%q is not a valid apps file. Expected kind: %s Actual Kind: %s", filePath, string(collecttypes.CfInstanceAppsMetadataKind), fileCfInstanceApps.Kind)
			continue
		}
		cfInstanceApps[filePath] = append(cfInstanceApps[filePath], fileCfInstanceApps.Spec.CfApplications...)
	}
	logrus.Debugf("Cf Instances %+v", cfInstanceApps)
	for _, filePath := range filePaths {
		applications, _, err := t.readApplicationManifest(filePath, "")
		if err != nil {
			logrus.Debugf("Failed to parse the manifest file at path %q Error: %q", filePath, err)
			continue
		}
		for _, application := range applications {
			fullbuilddirectory := filepath.Dir(filePath)
			if application.Path != "" {
				fullappdirectory := filepath.Join(filepath.Dir(filePath), application.Path)
				if _, err := os.Stat(fullappdirectory); !os.IsNotExist(err) {
					fullbuilddirectory = fullappdirectory
				} else {
					logrus.Debugf("Path to app directory %s does not exist, assuming manifest directory as app path", fullappdirectory)
				}
			}
			applicationName := application.Name
			if applicationName == "" {
				basename := filepath.Base(filePath)
				applicationName = strings.TrimSuffix(basename, filepath.Ext(basename))
			}
			ct := transformertypes.TransformerPlan{
				Mode:              transformertypes.ModeContainer,
				ArtifactTypes:     []transformertypes.ArtifactType{irtypes.IRArtifactType},
				BaseArtifactTypes: []transformertypes.ArtifactType{irtypes.IRArtifactType},
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.CloudFoundryConfigType: artifacts.CloudFoundryConfig{
						ServiceName: applicationName,
					}},
				Paths: map[transformertypes.PathType][]string{
					artifacts.CfManifestPathType:  {filePath},
					artifacts.ProjectPathPathType: {fullbuilddirectory},
				},
			}
			_, appinstance := getCfInstanceApp(cfInstanceApps, applicationName)
			if application.DockerImage != "" || appinstance.DockerImage != "" {
				dockerImageName := application.DockerImage
				if dockerImageName == "" {
					dockerImageName = appinstance.DockerImage
				}
				ct.ArtifactTypes = append(ct.ArtifactTypes, artifacts.ContainerBuildArtifactType)
				ct.BaseArtifactTypes = append(ct.ArtifactTypes, artifacts.ContainerBuildArtifactType)
				ctConfig := ct.Configs[artifacts.CloudFoundryConfigType].(artifacts.CloudFoundryConfig)
				ctConfig.ImageName = dockerImageName
				ct.Configs[artifacts.CloudFoundryConfigType] = ctConfig
				continue
			}
			services[applicationName] = []transformertypes.TransformerPlan{ct}
		}
	}
	return services, nil
}

// Transform transforms the artifacts
func (t *CloudFoundry) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
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

		ir := irtypes.NewIR()
		logrus.Debugf("Transforming %s", config.ServiceName)
		var cfinstanceapp collecttypes.CfApplication
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
			serviceConfig := irtypes.Service{Name: config.ServiceName}
			serviceContainer := core.Container{Name: config.ServiceName}
			serviceContainer.Image = config.ImageName
			if serviceContainer.Image == "" {
				serviceContainer.Image = sConfig.ServiceName
			}
			for varname, value := range application.EnvironmentVariables {
				serviceContainer.Env = append(serviceContainer.Env, core.EnvVar{Name: varname, Value: value})
			}
			//TODO: Add support for services, health check, memory
			if application.Instances.IsSet {
				serviceConfig.Replicas = application.Instances.Value
			} else if cfinstanceapp.Instances != 0 {
				serviceConfig.Replicas = cfinstanceapp.Instances
			}
			for varname, value := range cfinstanceapp.Env {
				serviceContainer.Env = append(serviceContainer.Env, core.EnvVar{Name: varname, Value: value})
			}
			if len(cfinstanceapp.Ports) > 0 {
				for _, port := range cfinstanceapp.Ports {
					// Add the port to the k8s pod.
					serviceContainer.Ports = append(serviceContainer.Ports, core.ContainerPort{ContainerPort: port})
					// Forward the port on the k8s service to the k8s pod.
					podPort := networking.ServiceBackendPort{Number: port}
					servicePort := podPort
					serviceConfig.AddPortForwarding(servicePort, podPort, "")
				}
				envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(cfinstanceapp.Ports[0])}
				serviceContainer.Env = append(serviceContainer.Env, envvar)
			} else {
				port := common.DefaultServicePort
				// Add the port to the k8s pod.
				serviceContainer.Ports = []core.ContainerPort{{ContainerPort: port}}
				// Forward the port on the k8s service to the k8s pod.
				podPort := networking.ServiceBackendPort{Number: port}
				servicePort := podPort
				serviceConfig.AddPortForwarding(servicePort, podPort, "")
				envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(port)}
				serviceContainer.Env = append(serviceContainer.Env, envvar)
			}
			serviceConfig.Containers = []core.Container{serviceContainer}
			ir.Services[config.ServiceName] = serviceConfig
		}
		artifactsCreated = append(artifactsCreated, transformertypes.Artifact{
			Name:     t.Env.GetProjectName(),
			Artifact: irtypes.IRArtifactType,
			Configs: map[transformertypes.ConfigType]interface{}{
				irtypes.IRConfigType: ir,
			},
		})
	}
	return nil, artifactsCreated, nil
}

// readApplicationManifest reads an application manifest
func (t *CloudFoundry) readApplicationManifest(path string, serviceName string) ([]manifest.Application, []string, error) { // manifest, parameters
	trimmedvariables, err := getMissingVariables(path)
	if err != nil {
		logrus.Debugf("Unable to read as cf manifest %s : %s", path, err)
		return nil, nil, err
	}

	rawManifest, err := ioutil.ReadFile(path)
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

func getCfInstanceApp(fileApps map[string][]collecttypes.CfApplication, name string) (string, collecttypes.CfApplication) {
	for path, apps := range fileApps {
		for _, app := range apps {
			if app.Name == name {
				return path, app
			}
		}
	}
	return "", collecttypes.CfApplication{}
}

func getCfAppInstance(path string, appname string) (collecttypes.CfApplication, error) {
	cfinstanceappsfile := collecttypes.CfInstanceApps{}
	if err := common.ReadMove2KubeYaml(path, &cfinstanceappsfile); err != nil {
		return collecttypes.CfApplication{}, err
	}
	for _, app := range cfinstanceappsfile.Spec.CfApplications {
		if app.Name == appname {
			return app, nil
		}
	}
	return collecttypes.CfApplication{}, fmt.Errorf("failed to find the app %s in the cf apps file at path %s", appname, path)
}
