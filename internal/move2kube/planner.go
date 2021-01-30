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

package move2kube

import (
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	"github.com/konveyor/move2kube/internal/metadata"
	"github.com/konveyor/move2kube/internal/qaengine"
	"github.com/konveyor/move2kube/internal/source"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

//CreatePlan creates the plan from all planners
func CreatePlan(inputPath string, prjName string, interactive bool) plantypes.Plan {
	p := plantypes.NewPlan()
	p.Name = prjName
	p.Spec.Inputs.RootDir = inputPath
	allowKube2Kube := true

	selectedTranslationPlanners := source.GetTranslators()
	if interactive {
		att := source.GetAllTranslatorTypes()
		att = append(att, string(plantypes.Kube2KubeTranslation))
		translationTypes := selectTranslators(att)
		selectedTranslationPlanners = []source.Translator{}
		for _, tp := range source.GetTranslators() {
			tpn := (string)(tp.GetTranslatorType())
			if common.IsStringPresent(translationTypes, tpn) {
				selectedTranslationPlanners = append(selectedTranslationPlanners, tp)
			}
		}
		if !common.IsStringPresent(translationTypes, string(plantypes.Kube2KubeTranslation)) {
			allowKube2Kube = false
		}

		if common.IsStringPresent(translationTypes, string(plantypes.Any2KubeTranslation)) || common.IsStringPresent(translationTypes, string(plantypes.CfManifest2KubeTranslation)) {
			containerizer.InitContainerizers(p.Spec.Inputs.RootDir, selectContainerizationTypes(containerizer.GetAllContainerBuildStrategies()))
		}
	} else {
		containerizer.InitContainerizers(p.Spec.Inputs.RootDir, nil)
	}

	if len(selectedTranslationPlanners) == 0 {
		log.Debugf("No sources selected")
	}

	log.Infoln("Planning Translation")
	for _, l := range selectedTranslationPlanners {
		log.Infof("[%T] Planning translation", l)
		services, err := l.GetServiceOptions(inputPath, p)
		if err != nil {
			log.Warnf("[%T] Failed : %s", l, err)
		} else {
			p.AddServicesToPlan(services)
			log.Infof("[%T] Done", l)
		}
	}
	log.Infoln("Translation planning done")

	log.Infoln("Planning Metadata")
	metadataPlanners := metadata.GetLoaders()
	for _, l := range metadataPlanners {
		if !allowKube2Kube {
			if _, ok := l.(*metadata.K8sFilesLoader); ok {
				continue
			}
		}
		log.Infof("[%T] Planning metadata", l)
		err := l.UpdatePlan(inputPath, &p)
		if err != nil {
			log.Warnf("[%T] Failed : %s", l, err)
		} else {
			log.Infof("[%T] Done", l)
		}
	}
	log.Infoln("Metadata planning done")
	return p
}

// CuratePlan allows curation the plan with the qa engine
func CuratePlan(p plantypes.Plan) plantypes.Plan {
	if len(p.Spec.Inputs.Services) == 0 {
		log.Debugf("No services found.")
	}

	// Identify translation types of interest
	translationTypes := []string{}
	for _, services := range p.Spec.Inputs.Services {
		for _, service := range services {
			translationTypes = append(translationTypes, string(service.TranslationType))
		}
	}
	translationTypes = common.UniqueStrings(translationTypes)
	selectedTranslationTypes := selectTranslators(translationTypes)
	planServices := map[string][]plantypes.Service{}
	for serviceName, services := range p.Spec.Inputs.Services {
		for _, service := range services {
			if common.IsStringPresent(selectedTranslationTypes, string(service.TranslationType)) {
				planServices[serviceName] = append(planServices[serviceName], service)
			}
		}
	}
	p.Spec.Inputs.Services = planServices
	if len(p.Spec.Inputs.Services) == 0 {
		if len(p.Spec.Inputs.K8sFiles) == 0 {
			log.Fatalf("Failed to find any services that support the selected translation types.")
		} else {
			log.Debugf("Failed to find any services that support the selected translation types.")
		}
	}

	// Identify services of interest
	servicenames := []string{}
	for sn := range p.Spec.Inputs.Services {
		servicenames = append(servicenames, sn)
	}
	problem, err := qatypes.NewMultiSelectProblem(common.ConfigServicesNamesKey, "Select all services that are needed:", []string{"The services unselected here will be ignored."}, servicenames, servicenames)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	selectedServices, err := problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	planServices = map[string][]plantypes.Service{}
	for _, s := range selectedServices {
		planServices[s] = p.Spec.Inputs.Services[s]
	}
	if len(p.Spec.Inputs.Services) == 0 {
		if len(p.Spec.Inputs.K8sFiles) == 0 {
			log.Fatalf("All services were deselected. Aborting.")
		} else {
			log.Debugf("All services were deselected however some k8s files were detected.")
		}
	}
	p.Spec.Inputs.Services = planServices

	// Identify containerization techniques of interest
	conTypes := []string{}
	for _, serviceOptions := range p.Spec.Inputs.Services {
		for _, serviceOption := range serviceOptions {
			conTypes = append(conTypes, string(serviceOption.ContainerBuildType))
		}
	}
	conTypes = common.UniqueStrings(conTypes)
	selectedConTypes := selectContainerizationTypes(conTypes)
	if len(selectedConTypes) == 0 {
		log.Infof("No containerization technique was selected; It could mean some services will get ignored.")
	}

	services := map[string][]plantypes.Service{}
	for serviceName, serviceOptions := range p.Spec.Inputs.Services {
		sConTypes := []string{}
		for _, serviceOption := range serviceOptions {
			if common.IsStringPresent(selectedConTypes, string(serviceOption.ContainerBuildType)) {
				sConTypes = append(sConTypes, string(serviceOption.ContainerBuildType))
			}
		}
		// TODO: service options should be have unique container build types already so we don't need to make sConTypes unique.
		// we need this because of a bug where different folders with the same name get detected as the same service.
		sConTypes = common.UniqueStrings(sConTypes)
		if len(sConTypes) == 0 {
			log.Warnf("Ignoring service %s, since it does not support any selected containerization technique.", serviceName)
			continue
		}
		selectedSConType := sConTypes[0]
		if len(sConTypes) > 1 {
			problem, err := qatypes.NewSelectProblem(common.ConfigServicesKey+common.Delim+serviceName+common.Delim+"containerization"+common.Delim+"type", "Select containerization technique for service "+serviceName+":", []string{"Choose the containerization technique of interest."}, selectedSConType, sConTypes)
			if err != nil {
				log.Fatalf("Unable to create problem : %s", err)
			}
			problem, err = qaengine.FetchAnswer(problem)
			if err != nil {
				log.Fatalf("Unable to fetch answer : %s", err)
			}
			selectedSConType, err = problem.GetStringAnswer()
			if err != nil {
				log.Fatalf("Unable to get answer : %s", err)
			}
		}

		for _, serviceOption := range serviceOptions {
			if selectedSConType != string(serviceOption.ContainerBuildType) {
				continue
			}
			if len(serviceOption.ContainerizationTargetOptions) <= 1 {
				if serviceOption.ContainerBuildType != plantypes.ReuseContainerBuildTypeValue &&
					serviceOption.ContainerBuildType != plantypes.ManualContainerBuildTypeValue &&
					len(serviceOption.ContainerizationTargetOptions) == 0 {
					log.Warnf("The selected containerization technique %v has no valid targets.", selectedSConType)
				}
				services[serviceName] = []plantypes.Service{serviceOption}
				break
			}

			// Multiple containerization targets

			// Convert absolute paths to relative. TODO: We are assuming that this won't make it ambiguous.
			// TODO: if we add more build types that require conversion add it here as well.
			buildTypesRequiringConversion := []string{
				string(plantypes.DockerFileContainerBuildTypeValue),
				string(plantypes.ReuseDockerFileContainerBuildTypeValue),
				string(plantypes.S2IContainerBuildTypeValue),
			}
			requiresConversion := common.IsStringPresent(buildTypesRequiringConversion, string(serviceOption.ContainerBuildType))
			options := serviceOption.ContainerizationTargetOptions
			if requiresConversion {
				options = []string{}
				for _, option := range serviceOption.ContainerizationTargetOptions {
					relOptionPath, err := p.GetRelativePath(option)
					if err != nil {
						log.Errorf("Failed to make the option path %q relative to the root directory. Error: %q", option, err)
						continue
					}
					options = append(options, relOptionPath)
				}
			}
			problem, err := qatypes.NewSelectProblem(common.ConfigServicesKey+common.Delim+serviceName+common.Delim+"containerization"+common.Delim+"target", "Select containerization target for service "+serviceName+":", []string{"Choose the target that should be used for containerization."}, options[0], options)
			if err != nil {
				log.Fatalf("Unable to create problem : %s", err)
			}
			problem, err = qaengine.FetchAnswer(problem)
			if err != nil {
				log.Fatalf("Unable to fetch answer : %s", err)
			}
			selectedSConMode, err := problem.GetStringAnswer()
			if err != nil {
				log.Fatalf("Unable to get answer : %s", err)
			}
			if requiresConversion {
				absOptionPath, err := p.GetAbsolutePath(selectedSConMode)
				if err != nil {
					log.Errorf("Failed to make the option path %q absolute. Error: %q", selectedSConMode, err)
				} else {
					selectedSConMode = absOptionPath
				}
			}
			serviceOption.ContainerizationTargetOptions = []string{selectedSConMode}
			services[serviceName] = []plantypes.Service{serviceOption}
			break
		}
	}
	p.Spec.Inputs.Services = services

	// Choose output artifact type
	artifactTypeList := make([]string, 3)
	artifactTypeList[0] = string(plantypes.Yamls)
	artifactTypeList[1] = string(plantypes.Helm)
	artifactTypeList[2] = string(plantypes.Knative)
	problem, err = qatypes.NewSelectProblem(common.ConfigTargetArtifacttTypeKey, "Choose the artifact type:", []string{"Yamls - Generate Kubernetes Yamls", "Helm - Generate Helm chart", "Knative - Create Knative artifacts"}, string(plantypes.Yamls), artifactTypeList)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	artifactType, err := problem.GetStringAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	p.Spec.Outputs.Kubernetes.ArtifactType = plantypes.TargetArtifactTypeValue(artifactType)

	// Choose cluster type to target
	clusters := new(metadata.ClusterMDLoader).GetClusters(p)
	clusterTypeList := []string{}
	for c := range clusters {
		clusterTypeList = append(clusterTypeList, c)
	}
	problem, err = qatypes.NewSelectProblem(common.ConfigTargetClusterTypeKey, "Choose the cluster type:", []string{"Choose the cluster type you would like to target"}, string(common.DefaultClusterType), clusterTypeList)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	clusterType, err := problem.GetStringAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	p.Spec.Outputs.Kubernetes.TargetCluster.Type = clusterType
	p.Spec.Outputs.Kubernetes.TargetCluster.Path = ""

	return p
}

func selectTranslators(translationTypes []string) (selectedTranslationTypes []string) {
	problem, err := qatypes.NewMultiSelectProblem(common.ConfigSourceTypesKey, "Select all source types that you are interested in:", []string{"Services that don't support any of the source types you are interested in will be ignored."}, translationTypes, translationTypes)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	selectedTranslationTypes, err = problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	return selectedTranslationTypes
}

func selectContainerizationTypes(containerizationTypes []string) (selectedConTypes []string) {
	problem, err := qatypes.NewMultiSelectProblem(common.ConfigContainerizationTypesKey, "Select all containerization modes that is of interest:", []string{"Services that don't support any of the containerization techniques you are interested in will be ignored."}, containerizationTypes, containerizationTypes)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	selectedConTypes, err = problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	return selectedConTypes
}
