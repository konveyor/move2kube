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

package api

import (
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/metadata"
	"github.com/konveyor/move2kube/internal/translator"
	"github.com/konveyor/move2kube/qaengine"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

//CreatePlan creates the plan from all planners
func CreatePlan(inputPath string, configurationsPath, prjName string) plantypes.Plan {
	p := plantypes.NewPlan()
	p.Name = prjName
	p.Spec.RootDir = inputPath
	p.Spec.ConfigurationsDir = configurationsPath
	if configurationsPath != "" {
		common.CheckAndCopyConfigurations(configurationsPath)
	}
	translator.Init(common.AssetsPath)
	ts := translator.GetTranslators()
	for tn, t := range ts {
		p.Spec.Configuration.Translators[tn] = t.GetConfig().Spec.FilePath
	}
	p.Spec.Services = translator.GetServices(p.Name, inputPath)
	log.Infof("No of services identified : %d", len(p.Spec.Services))
	p.Spec.IRTranslators, _ = translator.GetIRTranslators(p)

	log.Infoln("Planning Metadata")
	metadataPlanners := metadata.GetLoaders()
	for _, l := range metadataPlanners {
		log.Infof("[%T] Planning metadata", l)
		err := l.UpdatePlan(&p)
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
	// Choose cluster type to target
	clusters := new(metadata.ClusterMDLoader).GetClusters(p)
	clusterTypeList := []string{}
	for c := range clusters {
		clusterTypeList = append(clusterTypeList, c)
	}
	clusterType := qaengine.FetchSelectAnswer(common.ConfigTargetClusterTypeKey, "Choose the cluster type:", []string{"Choose the cluster type you would like to target"}, string(common.DefaultClusterType), clusterTypeList)
	p.Spec.TargetCluster.Type = clusterType
	p.Spec.TargetCluster.Path = ""

	return p
}
