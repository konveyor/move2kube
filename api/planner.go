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
	"github.com/konveyor/move2kube/internal/configuration"
	"github.com/konveyor/move2kube/internal/transformer"
	"github.com/konveyor/move2kube/qaengine"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
)

//CreatePlan creates the plan from all planners
func CreatePlan(inputPath string, configurationsPath, prjName string) plantypes.Plan {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	p := plantypes.NewPlan()
	p.Name = prjName
	p.Spec.RootDir = inputPath
	p.Spec.ConfigurationsDir = configurationsPath
	if configurationsPath != "" {
		common.CheckAndCopyConfigurations(configurationsPath)
	}
	logrus.Infoln("Loading Configuration")
	configurationLoaders := configuration.GetLoaders()
	for _, l := range configurationLoaders {
		logrus.Infof("[%T] Loading configuration", l)
		err := l.UpdatePlan(&p)
		if err != nil {
			logrus.Warnf("[%T] Failed : %s", l, err)
		} else {
			logrus.Infof("[%T] Done", l)
		}
	}
	transformer.Init(common.AssetsPath, inputPath)
	ts := transformer.GetTransformers()
	for tn, t := range ts {
		config, _ := t.GetConfig()
		p.Spec.Configuration.Transformers[tn] = config.Spec.FilePath
	}
	logrus.Infoln("Configuration loading done")

	var err error
	p.Spec.Services, err = transformer.GetServices(p.Name, inputPath)
	if err != nil {
		logrus.Errorf("Unable to create plan : %s", err)
	}
	logrus.Infof("No of services identified : %d", len(p.Spec.Services))
	return p
}

// CuratePlan allows curation the plan with the qa engine
func CuratePlan(p plantypes.Plan) plantypes.Plan {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	modes := []string{}
	transformers := []string{}
	for s, st := range p.Spec.Services {
		for _, t := range st {
			if t.Mode == "" {
				logrus.Warnf("Ignoring transformer %+v for service %s due to empty mode", t, s)
				continue
			}
			if !common.IsStringPresent(modes, t.Mode) {
				modes = append(modes, t.Mode)
			}
		}
	}
	for tn := range p.Spec.Configuration.Transformers {
		if !common.IsStringPresent(transformers, tn) {
			transformers = append(transformers, tn)
		}
	}
	modes = qaengine.FetchMultiSelectAnswer(common.ConfigModesKey, "Choose modes to use:", []string{"Modes generally specify the deployment model"}, modes, modes)
	transformers = qaengine.FetchMultiSelectAnswer(common.ConfigTransformerTypesKey, "Select all transformer types that you are interested in:", []string{"Services that don't support any of the transformer types you are interested in will be ignored."}, transformers, transformers)
	for sn, st := range p.Spec.Services {
		mode := ""
		exclusiveArtifactTypes := []string{}
		sTransformers := []plantypes.Transformer{}
		for _, t := range st {
			if mode == "" {
				if t.Mode == "" {
					logrus.Warnf("Ignoring transformer %+v for service %s due to empty mode", t, sn)
					continue
				}
				if !common.IsStringPresent(modes, t.Mode) {
					logrus.Debugf("Ignoring transformer %+v for service %s due to deselected mode %s", t, sn, t.Mode)
					continue
				}
				if !common.IsStringPresent(transformers, t.Name) {
					logrus.Debugf("Ignoring transformer %+v for service %s due to deselected transformer %s", t, sn, t.Mode)
					continue
				}
				mode = t.Mode
			} else if mode != t.Mode {
				logrus.Debugf("Ingoring %+v for service %s due to differing mode", t, sn)
			}
			if !common.IsStringPresent(transformers, t.Name) {
				logrus.Debugf("Ignoring transformer %+v for service %s due to deselected transformer %s", t, sn, t.Mode)
				continue
			}
			artifactsToUse := []plantypes.ArtifactType{}
			for _, at := range t.ArtifactTypes {
				if common.IsStringPresent(exclusiveArtifactTypes, string(at)) {
					continue
				}
				artifactsToUse = append(artifactsToUse, at)
			}
			if len(artifactsToUse) == 0 {
				continue
			}
			t.ArtifactTypes = artifactsToUse
			for _, e := range t.ExclusiveArtifactTypes {
				exclusiveArtifactTypes = append(exclusiveArtifactTypes, string(e))
			}
			sTransformers = append(sTransformers, t)
		}
		if mode != "" {
			modes = append(modes, mode)
		}
		if len(sTransformers) == 0 {
			logrus.Errorf("No transformers selected for service %s. Ignoring.", sn)
			continue
		}
		p.Spec.Services[sn] = sTransformers
	}
	transformer.InitTransformers(p.Spec.Configuration.Transformers, p.Spec.RootDir, true)

	// Choose cluster type to target
	clusters := new(configuration.ClusterMDLoader).GetClusters(p)
	clusterTypeList := []string{}
	for c := range clusters {
		clusterTypeList = append(clusterTypeList, c)
	}
	clusterType := qaengine.FetchSelectAnswer(common.ConfigTargetClusterTypeKey, "Choose the cluster type:", []string{"Choose the cluster type you would like to target"}, string(common.DefaultClusterType), clusterTypeList)
	p.Spec.TargetCluster.Type = clusterType
	p.Spec.TargetCluster.Path = ""

	logrus.Debugf("Plan : %+v", p)
	return p
}
