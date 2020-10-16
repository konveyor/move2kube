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

package source

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/cli/util/manifest"
	"github.com/cloudfoundry/bosh-cli/director/template"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	"github.com/konveyor/move2kube/internal/source/data"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

//go:generate go run github.com/konveyor/move2kube/internal/common/generator data

// CfManifestTranslator implements Translator interface for CfManifest files
type CfManifestTranslator struct {
}

// GetTranslatorType returns the translator type
func (c CfManifestTranslator) GetTranslatorType() plantypes.TranslationTypeValue {
	return plantypes.CfManifest2KubeTranslation
}

// GetServiceOptions - output a plan based on the input directory contents
func (c CfManifestTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	services := []plantypes.Service{}
	allcontainerizers := new(containerizer.Containerizers)
	allcontainerizers.InitContainerizers(inputPath)
	containerizers := collecttypes.CfContainerizers{}

	var files, err = common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize cf manifest yamls : %s", err)
	}
	// Load buildpack mappings, if available
	containerizersmetadata := collecttypes.CfContainerizers{}
	err = yaml.Unmarshal([]byte(data.Cfbuildpacks_yaml), &containerizersmetadata)
	if err != nil {
		log.Debugf("Not valid containerizer option data : %s", err)
	}
	containerizers.Spec.BuildpackContainerizers = append(containerizers.Spec.BuildpackContainerizers, containerizersmetadata.Spec.BuildpackContainerizers...)
	for _, fullpath := range files {
		containerizersmetadata := collecttypes.CfContainerizers{}
		err := common.ReadYaml(fullpath, &containerizersmetadata)
		if err != nil {
			log.Debugf("Not a valid containerizer option file : %s %s", fullpath, err)
			continue
		}
		containerizers.Spec.BuildpackContainerizers = append(containerizers.Spec.BuildpackContainerizers, containerizersmetadata.Spec.BuildpackContainerizers...)
	}
	log.Debugf("Containerizers %+v", containerizers)

	// Load instance apps, if available
	cfinstanceapps := map[string][]collecttypes.CfApplication{} //path
	for _, fullpath := range files {
		cfinstanceappsfile := collecttypes.CfInstanceApps{}
		err := common.ReadYaml(fullpath, &cfinstanceappsfile)
		if err != nil || cfinstanceappsfile.Kind != string(collecttypes.CfInstanceAppsMetadataKind) {
			log.Debugf("Not a valid apps file : %s %s", fullpath, err)
			continue
		}
		relpath, _ := plan.GetRelativePath(fullpath)
		cfinstanceapps[relpath] = append(cfinstanceapps[relpath], cfinstanceappsfile.Spec.CfApplications...)
	}
	log.Debugf("Cf Instances %+v", cfinstanceapps)

	appsCovered := []string{}

	for _, fullpath := range files {
		applications, _, err := ReadApplicationManifest(fullpath, "", plantypes.Yamls)
		if err != nil {
			log.Debugf("Error while trying to parse manifest : %s", err)
			continue
		}
		path, _ := plan.GetRelativePath(fullpath)
		for _, application := range applications {
			fullbuilddirectory := filepath.Dir(fullpath)
			if application.Path != "" {
				fullbuilddirectory = filepath.Join(fullpath, application.Path)
			}
			applicationName := application.Name
			if applicationName == "" {
				basename := filepath.Base(fullpath)
				applicationName = strings.TrimSuffix(basename, filepath.Ext(basename))
			}
			appinstancefilepath, appinstance := getCfInstanceApp(cfinstanceapps, applicationName)
			builddirectory, _ := plan.GetRelativePath(fullbuilddirectory)
			if application.DockerImage != "" || appinstance.DockerImage != "" {
				service := c.newService(applicationName)
				service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue
				if application.DockerImage != "" {
					service.Image = application.DockerImage
				} else {
					service.Image = appinstance.DockerImage
				}
				service.UpdateContainerBuildPipeline = false
				services = append(services, service)
				appsCovered = append(appsCovered, applicationName)
			} else {
				containerizationoptionsfound := false
				for _, cop := range allcontainerizers.GetContainerizationOptions(plan, fullbuilddirectory) {
					service := c.newService(applicationName)
					service.ContainerBuildType = cop.ContainerizationType
					service.ContainerizationTargetOptions = cop.TargetOptions
					service.AddSourceArtifact(plantypes.CfManifestArtifactType, path)
					if appinstance.Name != "" {
						service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appinstancefilepath)
					}
					if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], builddirectory) {
						service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, builddirectory)
						service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, builddirectory)
					}
					services = append(services, service)
					appsCovered = append(appsCovered, applicationName)
					containerizationoptionsfound = true
				}
				for _, containerizer := range containerizers.Spec.BuildpackContainerizers {
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
					if isbuildpackmatched {
						service := c.newService(applicationName)
						service.ContainerBuildType = containerizer.ContainerBuildType
						service.ContainerizationTargetOptions = containerizer.ContainerizationTargetOptions
						service.AddSourceArtifact(plantypes.CfManifestArtifactType, path)
						if appinstance.Name != "" {
							service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appinstancefilepath)
						}
						if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], builddirectory) {
							service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, builddirectory)
							service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, builddirectory)
						}
						services = append(services, service)
						appsCovered = append(appsCovered, applicationName)
						containerizationoptionsfound = true
					}
				}
				if !containerizationoptionsfound {
					log.Warnf("No known containerization approach for %s even though it has a cf manifest %s; Defaulting to manual", fullbuilddirectory, filepath.Base(fullpath))
					service := c.newService(applicationName)
					service.ContainerBuildType = plantypes.ManualContainerBuildTypeValue
					service.AddSourceArtifact(plantypes.CfManifestArtifactType, path)
					if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], builddirectory) {
						service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, builddirectory)
						service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, builddirectory)
					}
					appsCovered = append(appsCovered, applicationName)
					services = append(services, service)
				}
			}
		}

		for appfilepath, apps := range cfinstanceapps {
			for _, application := range apps {
				applicationName := application.Name
				if !common.IsStringPresent(appsCovered, applicationName) {
					fullbuilddirectory := filepath.Dir(appfilepath)
					applicationName := application.Name
					if applicationName == "" {
						continue
					}
					builddirectory, _ := plan.GetRelativePath(fullbuilddirectory)
					if application.DockerImage != "" {
						service := c.newService(applicationName)
						service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue
						if application.DockerImage != "" {
							service.Image = application.DockerImage
						}
						service.UpdateContainerBuildPipeline = false
						services = append(services, service)
					} else {
						containerizationoptionsfound := false
						//TODO: Think whether we should include this for only runtime manifest file
						for _, cop := range allcontainerizers.GetContainerizationOptions(plan, fullbuilddirectory) {
							service := c.newService(applicationName)
							service.ContainerBuildType = cop.ContainerizationType
							service.ContainerizationTargetOptions = cop.TargetOptions
							service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appfilepath)
							if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], builddirectory) {
								service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, builddirectory)
								service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, builddirectory)
							}
							services = append(services, service)
							containerizationoptionsfound = true
						}
						for _, containerizer := range containerizers.Spec.BuildpackContainerizers {
							isbuildpackmatched := false
							if !isbuildpackmatched {
								if (application.Buildpack != "" && containerizer.BuildpackName == application.Buildpack) || (application.DetectedBuildpack != "" && containerizer.BuildpackName == application.DetectedBuildpack) {
									isbuildpackmatched = true
								}
							}
							if isbuildpackmatched {
								service := c.newService(applicationName)
								service.ContainerBuildType = containerizer.ContainerBuildType
								service.ContainerizationTargetOptions = containerizer.ContainerizationTargetOptions
								service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appfilepath)
								if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], builddirectory) {
									service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, builddirectory)
									service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, builddirectory)
								}
								services = append(services, service)
								containerizationoptionsfound = true
							}
						}
						if !containerizationoptionsfound {
							log.Warnf("No known containerization approach for %s even though it has a cf manifest %s; Defaulting to manual", fullbuilddirectory, filepath.Base(fullpath))
							service := c.newService(applicationName)
							service.ContainerBuildType = plantypes.ManualContainerBuildTypeValue
							service.AddSourceArtifact(plantypes.CfRunningManifestArtifactType, appfilepath)
							if !common.IsStringPresent(service.BuildArtifacts[plantypes.SourceDirectoryBuildArtifactType], builddirectory) {
								service.AddSourceArtifact(plantypes.SourceDirectoryArtifactType, builddirectory)
								service.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, builddirectory)
							}
							services = append(services, service)
						}
					}

				}
			}
		}
	}
	return services, nil
}

// Translate translates servies to IR
func (c CfManifestTranslator) Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(p)
	containerizers := new(containerizer.Containerizers)
	containerizers.InitContainerizers(p.Spec.Inputs.RootDir)
	for _, service := range services {
		if service.TranslationType == c.GetTranslatorType() {
			log.Debugf("Translating %s", service.ServiceName)

			var cfinstanceapp collecttypes.CfApplication
			if runninginstancefile, ok := service.SourceArtifacts[plantypes.CfRunningManifestArtifactType]; ok {
				cfinstanceapp = getCfAppInstance(p.GetFullPath(runninginstancefile[0]), service.ServiceName)
			}

			if paths, ok := service.SourceArtifacts[plantypes.CfManifestArtifactType]; ok {
				path := paths[0]
				applications, variables, err := ReadApplicationManifest(p.GetFullPath(path), service.ServiceName, p.Spec.Outputs.Kubernetes.ArtifactType)
				if err != nil {
					log.Debugf("Error while trying to parse manifest : %s", err)
					continue
				}
				application := applications[0]
				serviceConfig := irtypes.Service{Name: service.ServiceName}
				serviceContainer := v1.Container{Name: service.ServiceName}
				serviceContainer.Image = service.Image
				//TODO: Add support for services, health check, memory
				if application.Instances.IsSet {
					serviceConfig.Replicas = application.Instances.Value
				} else if cfinstanceapp.Instances != 0 {
					serviceConfig.Replicas = cfinstanceapp.Instances
				}
				for varname, value := range application.EnvironmentVariables {
					envvar := v1.EnvVar{Name: varname, Value: value}
					serviceContainer.Env = append(serviceContainer.Env, envvar)
				}
				for varname, value := range cfinstanceapp.Env {
					envvar := v1.EnvVar{Name: varname, Value: value}
					serviceContainer.Env = append(serviceContainer.Env, envvar)
				}
				for _, variable := range variables {
					ir.Values.GlobalVariables[variable] = variable
				}
				if cfinstanceapp.Ports != nil && len(cfinstanceapp.Ports) > 0 {
					serviceContainer.Ports = []v1.ContainerPort{}
					for _, port := range cfinstanceapp.Ports {
						serviceContainer.Ports = append(serviceContainer.Ports, v1.ContainerPort{ContainerPort: port,
							Protocol: v1.ProtocolTCP,
						})
					}
					envvar := v1.EnvVar{Name: "PORT", Value: string(cfinstanceapp.Ports[0])}
					serviceContainer.Env = append(serviceContainer.Env, envvar)
				} else {
					serviceContainer.Ports = []v1.ContainerPort{
						{ContainerPort: 8080,
							Protocol: v1.ProtocolTCP,
						}}
					envvar := v1.EnvVar{Name: "PORT", Value: "8080"}
					serviceContainer.Env = append(serviceContainer.Env, envvar)
				}
				serviceConfig.Containers = []v1.Container{serviceContainer}
				container, err := containerizers.GetContainer(p, service)
				if err == nil {
					ir.AddContainer(container)
					ir.Services[service.ServiceName] = serviceConfig
				} else {
					log.Errorf("Unable to translate service %s using cfmanifest at %s : %s", service.ServiceName, path, err)
				}
			} else {
				serviceConfig := irtypes.Service{Name: service.ServiceName}
				serviceContainer := v1.Container{Name: service.ServiceName}
				serviceContainer.Image = service.Image
				if cfinstanceapp.Instances != 0 {
					serviceConfig.Replicas = cfinstanceapp.Instances
				}
				for varname, value := range cfinstanceapp.Env {
					envvar := v1.EnvVar{Name: varname, Value: value}
					serviceContainer.Env = append(serviceContainer.Env, envvar)
				}
				if cfinstanceapp.Ports != nil && len(cfinstanceapp.Ports) > 0 {
					serviceContainer.Ports = []v1.ContainerPort{}
					for _, port := range cfinstanceapp.Ports {
						serviceContainer.Ports = append(serviceContainer.Ports, v1.ContainerPort{ContainerPort: port,
							Protocol: v1.ProtocolTCP,
						})
					}
				} else {
					serviceContainer.Ports = []v1.ContainerPort{
						{
							ContainerPort: 8080,
							Protocol:      v1.ProtocolTCP,
						}}
				}
				serviceConfig.Containers = []v1.Container{serviceContainer}
				container, err := containerizers.GetContainer(p, service)
				if err == nil {
					ir.AddContainer(container)
					ir.Services[service.ServiceName] = serviceConfig
				} else {
					log.Errorf("Unable to translate service %s using cfinstancemanifest : %s", service.ServiceName, err)
				}
			}
		}
	}
	return ir, nil
}

func (c CfManifestTranslator) newService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, c.GetTranslatorType())
	service.AddSourceType(plantypes.DirectorySourceTypeValue)
	service.AddSourceType(plantypes.CfManifestSourceTypeValue)
	service.UpdateContainerBuildPipeline = true
	service.UpdateDeployPipeline = true
	return service
}

// ReadApplicationManifest reads an application manifest
func ReadApplicationManifest(path string, serviceName string, artifactType plantypes.TargetArtifactTypeValue) ([]manifest.Application, []string, error) { // manifest, parameters
	trimmedvariables, err := getMissingVariables(path)
	if err != nil {
		log.Debugf("Unable to read as cf manifest %s : %s", path, err)
		return []manifest.Application{}, []string{}, err
	}

	rawManifest, err := ioutil.ReadFile(path)
	if err != nil {
		log.Errorf("Unable to read manifest file %s", path)
		return []manifest.Application{}, []string{}, err
	}
	tpl := template.NewTemplate(rawManifest)
	fileVars := template.StaticVariables{}
	for _, variable := range trimmedvariables {
		if artifactType == plantypes.Helm {
			fileVars[variable] = "{{ index  .Values " + `"globalvariables" "` + variable + `"}}`
		} else {
			fileVars[variable] = "{{ $" + variable + " }}"
		}
	}
	rawManifest, err = tpl.Evaluate(fileVars, nil, template.EvaluateOpts{ExpectAllKeys: true})
	if err != nil {
		log.Errorf("Interpolation Error %s", err)
		return []manifest.Application{}, []string{}, err
	}

	var m manifest.Manifest
	err = yaml.Unmarshal(rawManifest, &m)
	if err != nil {
		log.Errorf("UnMarshalling error %s", err)
		return []manifest.Application{}, []string{}, err
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
			log.Debugf("Error %s", err)
			return []string{}, err
		}
	}
	return trimmedvariables, nil
}

func getCfInstanceApp(apps map[string][]collecttypes.CfApplication, name string) (string, collecttypes.CfApplication) {
	for path, apps := range apps {
		for _, app := range apps {
			if app.Name == name {
				return path, app
			}
		}
	}
	return "", collecttypes.CfApplication{}
}

func getCfAppInstance(path string, appname string) collecttypes.CfApplication {
	cfinstanceappsfile := collecttypes.CfInstanceApps{}
	err := common.ReadYaml(path, &cfinstanceappsfile)
	if err != nil {
		log.Debugf("Not a valid apps file : %s %s", path, err)
	}
	for _, app := range cfinstanceappsfile.Spec.CfApplications {
		if app.Name == appname {
			return app
		}
	}
	return collecttypes.CfApplication{}
}
