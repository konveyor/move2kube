/*
Copyright IBM Corporation 2021

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

package classes

/*
import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/cli/util/manifest"
	"github.com/cloudfoundry/bosh-cli/director/template"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/translator/executable/containerizer"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	// CfManifestArtifactType defines the source artifact type of cf manifest
	CfManifestArtifactType = "CfManifest"
	// CfRunningManifestArtifactType defines the source artifact type of a manifest of a running instance
	CfRunningManifestArtifactType = "CfRunningManifest"
)

// CloudFoundry implements Translator interface
type CloudFoundry struct {
}

type CloudFoundryConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
}

func (t *CloudFoundry) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	namedServices = map[string]plantypes.Service{}

	filePaths, err := common.GetFilesByExt(dir, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", dir, err)
		return namedServices, nil, err
	}

	// Load buildpack mappings, if available
	cfContainerizers := collecttypes.CfContainerizers{}
	err = yaml.Unmarshal([]byte(data.Cfbuildpacks_yaml), &cfContainerizers)
	if err != nil {
		logrus.Debugf("Not valid containerizer option data : %s", err)
	}
	for _, filePath := range filePaths {
		containerizersMetadata := collecttypes.CfContainerizers{}
		err := common.ReadMove2KubeYaml(filePath, &containerizersMetadata)
		if err != nil {
			logrus.Debugf("Not a valid containerizer option file at path %q Error: %q", filePath, err)
			continue
		}
		cfContainerizers.Spec.BuildpackContainerizers = append(cfContainerizers.Spec.BuildpackContainerizers, containerizersMetadata.Spec.BuildpackContainerizers...)
	}
	logrus.Debugf("Containerizers %+v", cfContainerizers)

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

	appsCovered := []string{}

	for _, filePath := range filePaths {
		applications, _, err := ReadApplicationManifest(filePath, "")
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
			appinstancefilepath, appinstance := getCfInstanceApp(cfInstanceApps, applicationName)
			if application.DockerImage != "" || appinstance.DockerImage != "" {
				service := plantypes.Service{}
				service.GenerationOptions = []plantypes.GenerationOption{{
					Mode: plantypes.GenerationModeContainer,
					Name: string(plantypes.ReuseContainerBuildTypeValue),
				}}
				services = append(services, service)
				appsCovered = append(appsCovered, applicationName)
				continue
			}
			containerizationoptionsfound := false
			for _, cop := range containerizer.GetContainerizationOptions(plan, fullbuilddirectory) {
				service := plantypes.Service{}
				service.GenerationOptions = []plantypes.GenerationOption{{
					Mode: cop.ContainerizationType,
					Name: string(plantypes.ReuseContainerBuildTypeValue),
				}}
				service.ContainerizationOptions = cop.TargetOptions
				service.AddSourceArtifact(plantypes.CfManifestArtifactType, filePath)
				if appinstance.Name != "" {
					service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appinstancefilepath)
				}
				if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], fullbuilddirectory) {
					service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, fullbuilddirectory)
					service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, fullbuilddirectory)
				}
				services = append(services, service)
				appsCovered = append(appsCovered, applicationName)
				containerizationoptionsfound = true
			}
			for _, containerizer := range cfContainerizers.Spec.BuildpackContainerizers {
				isbuildpackmatched := false
				if application.Buildpack.IsSet && containerizer.BuildpackName == application.Buildpack.Value {
					isbuildpackmatched = true
				}
				for _, bpname := range application.Buildpacks {
					if bpname == containerizer.BuildpackName {
						isbuildpackmatched = true
						break
					}
				}
				if !isbuildpackmatched {
					if (appinstance.Buildpack != "" && containerizer.BuildpackName == appinstance.Buildpack) || (appinstance.DetectedBuildpack != "" && containerizer.BuildpackName == appinstance.DetectedBuildpack) {
						isbuildpackmatched = true
					}
				}
				if !isbuildpackmatched {
					continue
				}
				service := cfManifestTranslator.newService(applicationName)
				service.ContainerBuildType = containerizer.ContainerBuildType
				service.ContainerizationOptions = containerizer.ContainerizationTargetOptions
				service.AddSourceArtifact(plantypes.CfManifestArtifactType, filePath)
				if appinstance.Name != "" {
					service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appinstancefilepath)
				}
				if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], fullbuilddirectory) {
					service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, fullbuilddirectory)
					service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, fullbuilddirectory)
				}
				services = append(services, service)
				appsCovered = append(appsCovered, applicationName)
				containerizationoptionsfound = true
			}
			if !containerizationoptionsfound {
				logrus.Warnf("No known containerization approach for %s even though it has a cf manifest %s; Defaulting to manual", fullbuilddirectory, filepath.Base(filePath))
				service := cfManifestTranslator.newService(applicationName)
				service.ContainerBuildType = plantypes.ManualContainerBuildTypeValue
				service.AddSourceArtifact(plantypes.CfManifestArtifactType, filePath)
				if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], fullbuilddirectory) {
					service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, fullbuilddirectory)
					service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, fullbuilddirectory)
				}
				appsCovered = append(appsCovered, applicationName)
				services = append(services, service)
			}
		}

		for appfilepath, apps := range cfInstanceApps {
			for _, application := range apps {
				applicationName := application.Name
				if !common.IsStringPresent(appsCovered, applicationName) {
					fullbuilddirectory := filepath.Dir(appfilepath)
					applicationName := application.Name
					if applicationName == "" {
						continue
					}
					if application.DockerImage != "" {
						service := cfManifestTranslator.newService(applicationName)
						service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue
						if application.DockerImage != "" {
							service.Image = application.DockerImage
						}
						services = append(services, service)
					} else {
						containerizationoptionsfound := false
						//TODO: Think whether we should include this for only runtime manifest file
						for _, cop := range containerizer.GetContainerizationOptions(plan, fullbuilddirectory) {
							service := cfManifestTranslator.newService(applicationName)
							service.ContainerBuildType = cop.ContainerizationType
							service.ContainerizationOptions = cop.TargetOptions
							service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appfilepath)
							if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], fullbuilddirectory) {
								service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, fullbuilddirectory)
								service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, fullbuilddirectory)
							}
							services = append(services, service)
							containerizationoptionsfound = true
						}
						for _, containerizer := range cfContainerizers.Spec.BuildpackContainerizers {
							isbuildpackmatched := false
							if !isbuildpackmatched {
								if (application.Buildpack != "" && containerizer.BuildpackName == application.Buildpack) || (application.DetectedBuildpack != "" && containerizer.BuildpackName == application.DetectedBuildpack) {
									isbuildpackmatched = true
								}
							}
							if isbuildpackmatched {
								service := cfManifestTranslator.newService(applicationName)
								service.ContainerBuildType = containerizer.ContainerBuildType
								service.ContainerizationOptions = containerizer.ContainerizationTargetOptions
								service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appfilepath)
								if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], fullbuilddirectory) {
									service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, fullbuilddirectory)
									service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, fullbuilddirectory)
								}
								services = append(services, service)
								containerizationoptionsfound = true
							}
						}
						if !containerizationoptionsfound {
							logrus.Warnf("No known containerization approach for %s even though it has a cf manifest %s; Defaulting to manual", fullbuilddirectory, filepath.Base(filePath))
							service := cfManifestTranslator.newService(applicationName)
							service.ContainerBuildType = plantypes.ManualContainerBuildTypeValue
							service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appfilepath)
							if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], fullbuilddirectory) {
								service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, fullbuilddirectory)
								service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, fullbuilddirectory)
							}
							services = append(services, service)
						}
					}

				}
			}
		}
	}
	return services, nil, nil
}

func (t *CloudFoundry) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *CloudFoundry) KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *CloudFoundry) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *CloudFoundry) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *CloudFoundry) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error) {
	ir := irtypes.NewIR(plan.Name)
	logrus.Debugf("Translating %s", serviceName)
	var cfinstanceapp collecttypes.CfApplication
	if runninginstancefile, ok := translatorPlan.Paths[CfRunningManifestArtifactType]; ok {
		var err error
		cfinstanceapp, err = getCfAppInstance(runninginstancefile[0], service.ServiceName)
		if err != nil {
			logrus.Debugf("The file at path %s is not a valid cf apps file. Error: %q", runninginstancefile[0], err)
		}
	}

	if paths, ok := service.SourceArtifacts[CfManifestArtifactType]; ok {
		path := paths[0] // TODO: what about the rest of the manifests?
		applications, variables, err := ReadApplicationManifest(path, service.ServiceName)
		if err != nil {
			logrus.Debugf("Error while trying to parse manifest : %s", err)
			continue
		}
		logrus.Debugf("Using cf manifest file at path %s to translate service %s", path, service.ServiceName)
		container, err := containerizer.GetContainer(plan, service)
		if err != nil {
			logrus.Errorf("Failed to containerize service %s in cf manifest file at path %s Error: %q", service.ServiceName, path, err)
			continue
		}
		ir.AddContainer(container)
		application := applications[0]
		serviceConfig := irtypes.NewServiceFromPlanService(service)
		serviceContainer := core.Container{Name: service.ServiceName}
		serviceContainer.Image = service.Image
		for varname, value := range application.EnvironmentVariables {
			serviceContainer.Env = append(serviceContainer.Env, core.EnvVar{Name: varname, Value: value})
		}
		for _, variable := range variables {
			ir.Values.GlobalVariables[variable] = variable
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
				podPort := irtypes.Port{Number: int32(port)}
				servicePort := podPort
				serviceConfig.AddPortForwarding(servicePort, podPort)
			}
			envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(cfinstanceapp.Ports[0])}
			serviceContainer.Env = append(serviceContainer.Env, envvar)
		} else {
			if len(container.ExposedPorts) > 0 {
				for _, port := range container.ExposedPorts {
					// Add the port to the k8s pod.
					serviceContainer.Ports = append(serviceContainer.Ports, core.ContainerPort{ContainerPort: int32(port)})
					// Forward the port on the k8s service to the k8s pod.
					podPort := irtypes.Port{Number: int32(port)}
					servicePort := podPort
					serviceConfig.AddPortForwarding(servicePort, podPort)
				}
				envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(container.ExposedPorts[0])}
				serviceContainer.Env = append(serviceContainer.Env, envvar)
			} else {
				port := common.DefaultServicePort
				// Add the port to the k8s pod.
				serviceContainer.Ports = []core.ContainerPort{{ContainerPort: int32(port)}}
				// Forward the port on the k8s service to the k8s pod.
				podPort := irtypes.Port{Number: int32(port)}
				servicePort := podPort
				serviceConfig.AddPortForwarding(servicePort, podPort)
				envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(port)}
				serviceContainer.Env = append(serviceContainer.Env, envvar)
			}
		}
		serviceConfig.Containers = []core.Container{serviceContainer}
		ir.Services[service.ServiceName] = serviceConfig
	} else {
		logrus.Debugf("No cf manifest file found for service %s", service.ServiceName)
		container, err := containerizer.GetContainer(plan, service)
		if err != nil {
			logrus.Errorf("Failed to containerize service %s using cfmanifest translator. Error: %q", service.ServiceName, err)
			continue
		}
		ir.AddContainer(container)
		serviceConfig := irtypes.NewServiceFromPlanService(service)
		serviceContainer := core.Container{Name: service.ServiceName, Image: service.Image}
		if cfinstanceapp.Instances != 0 {
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
				podPort := irtypes.Port{Number: port}
				servicePort := podPort
				serviceConfig.AddPortForwarding(servicePort, podPort)
			}
			envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(cfinstanceapp.Ports[0])}
			serviceContainer.Env = append(serviceContainer.Env, envvar)
		} else {
			if len(container.ExposedPorts) > 0 {
				for _, port := range container.ExposedPorts {
					// Add the port to the k8s pod.
					serviceContainer.Ports = append(serviceContainer.Ports, core.ContainerPort{ContainerPort: int32(port)})
					// Forward the port on the k8s service to the k8s pod.
					podPort := irtypes.Port{Number: int32(port)}
					servicePort := podPort
					serviceConfig.AddPortForwarding(servicePort, podPort)
				}
				envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(container.ExposedPorts[0])}
				serviceContainer.Env = append(serviceContainer.Env, envvar)
			} else {
				port := int32(common.DefaultServicePort)
				// Add the port to the k8s pod.
				serviceContainer.Ports = []core.ContainerPort{{ContainerPort: port}}
				// Forward the port on the k8s service to the k8s pod.
				podPort := irtypes.Port{Number: int32(port)}
				servicePort := podPort
				serviceConfig.AddPortForwarding(servicePort, podPort)
				envvar := core.EnvVar{Name: "PORT", Value: cast.ToString(port)}
				serviceContainer.Env = append(serviceContainer.Env, envvar)
			}
		}
		serviceConfig.Containers = []core.Container{serviceContainer}
		ir.Services[service.ServiceName] = serviceConfig
	}

	p := translatortypes.Patch{
		IR: ir,
	}
	return []translatortypes.Patch{p}, nil
}

func (t *CloudFoundry) TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.PathMapping, error) {
	return nil, nil
}

// ReadApplicationManifest reads an application manifest
func ReadApplicationManifest(path string, serviceName string) ([]manifest.Application, []string, error) { // manifest, parameters
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
	return collecttypes.CfApplication{}, fmt.Errorf("Failed to find the app %s in the cf apps file at path %s", appname, path)
}
*/
