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
	log "github.com/sirupsen/logrus"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/metadata"
	"github.com/konveyor/move2kube/internal/qaengine"
	"github.com/konveyor/move2kube/internal/source"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

//CreatePlan creates the plan from all planners
func CreatePlan(inputPath string, prjName string) plantypes.Plan {
	var p = plantypes.NewPlan()
	p.Name = prjName
	// Setup rootdir.
	if err := p.Spec.Inputs.SetRootDir(inputPath); err != nil {
		log.Errorf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
	}

	translationPlanners := source.GetSourceLoaders()
	log.Infoln("Planning Translation")
	for _, l := range translationPlanners {
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
	// Load qa caches
	cachepaths := []string{}
	for i := len(p.Spec.Inputs.QACaches) - 1; i >= 0; i-- {
		cachepaths = append(cachepaths, p.GetFullPath(p.Spec.Inputs.QACaches[i]))
	}
	qaengine.AddCaches(cachepaths)
	// Identify services of interest
	servicenames := []string{}
	for sn := range p.Spec.Inputs.Services {
		servicenames = append(servicenames, sn)
	}
	problem, err := qatypes.NewMultiSelectProblem("Select all services that are needed:", []string{"The services unselected here will be ignored."}, servicenames, servicenames)
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
	planservices := map[string][]plantypes.Service{}
	for _, s := range selectedServices {
		planservices[s] = p.Spec.Inputs.Services[s]
	}
	p.Spec.Inputs.Services = planservices

	// Identify containerization techniques of interest
	conTypes := []string{}
	for _, s := range p.Spec.Inputs.Services {
		for _, so := range s {
			if !common.IsStringPresent(conTypes, string(so.ContainerBuildType)) {
				conTypes = append(conTypes, string(so.ContainerBuildType))
			}
		}
	}
	problem, err = qatypes.NewMultiSelectProblem("Select all containerization modes that is of interest:", []string{"The services which does not support any of the containerization technique you are interested will be ignored."}, conTypes, conTypes)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	selectedConTypes, err := problem.GetSliceAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	if len(selectedConTypes) == 0 {
		log.Fatalf("No containerization technique was selected; Terminating.")
	}
	services := map[string][]plantypes.Service{}
	for sn, s := range p.Spec.Inputs.Services {
		sConTypes := []string{}
		for _, so := range s {
			if common.IsStringPresent(selectedConTypes, string(so.ContainerBuildType)) {
				sConTypes = append(sConTypes, string(so.ContainerBuildType))
			}
		}
		if len(sConTypes) == 0 {
			log.Warnf("Ignoring service %s, since it does not support any selected containerization technique.", sn)
			continue
		}
		selectedSConType := sConTypes[0]
		if len(sConTypes) > 1 {
			problem, err := qatypes.NewSelectProblem("Select containerization technique for service "+sn+":", []string{"Choose the containerization technique of interest."}, selectedSConType, sConTypes)
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
		for _, so := range s {
			if selectedSConType == string(so.ContainerBuildType) {
				if len(so.ContainerizationTargetOptions) > 1 {
					problem, err := qatypes.NewSelectProblem("Select containerization technique's mode for service "+sn+":", []string{"Choose the containerization technique mode of interest."}, so.ContainerizationTargetOptions[0], so.ContainerizationTargetOptions)
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
					so.ContainerizationTargetOptions = []string{selectedSConMode}
				}
				services[sn] = []plantypes.Service{so}
				break
			}
		}
	}
	p.Spec.Inputs.Services = services

	// Choose output artifact type
	artifactTypeList := make([]string, 3)
	artifactTypeList[0] = string(plantypes.Yamls)
	artifactTypeList[1] = string(plantypes.Helm)
	artifactTypeList[2] = string(plantypes.Knative)
	problem, err = qatypes.NewSelectProblem("Choose the artifact type:", []string{"Yamls - Generate Kubernetes Yamls", "Helm - Generate Helm chart", "Knative - Create Knative artifacts"}, string(plantypes.Yamls), artifactTypeList)
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
	clusters := metadata.ClusterMDLoader{}.GetClusters(p)
	clusterTypeList := []string{}
	for c := range clusters {
		clusterTypeList = append(clusterTypeList, c)
	}
	problem, err = qatypes.NewSelectProblem("Choose the cluster type:", []string{"Choose the cluster type you would like to target"}, string(common.DefaultClusterType), clusterTypeList)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	p.Spec.Outputs.Kubernetes.ClusterType, err = problem.GetStringAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}

	return p
}

//WritePlan writes the plan
func WritePlan(plan plantypes.Plan, outputPath string) error {
	err := common.WriteYaml(outputPath, plan)
	return err
}

//ReadPlan reads the plan
func ReadPlan(planFile string) (plantypes.Plan, error) {
	var plan plantypes.Plan
	err := common.ReadYaml(planFile, &plan)
	return plan, err
}
