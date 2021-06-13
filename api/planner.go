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
	"github.com/konveyor/move2kube/internal/translator"
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
	translator.Init(common.AssetsPath, inputPath)
	ts := translator.GetTranslators()
	for tn, t := range ts {
		p.Spec.Configuration.Translators[tn] = t.GetConfig().Spec.FilePath
	}
	logrus.Infoln("Configuration loading done")

	p.Spec.Services = translator.GetServices(p.Name, inputPath)
	logrus.Infof("No of services identified : %d", len(p.Spec.Services))
	p.Spec.PlanTranslators, _ = translator.GetPlanTranslators(p)
	return p
}

// CuratePlan allows curation the plan with the qa engine
func CuratePlan(p plantypes.Plan) plantypes.Plan {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	modes := []string{}
	translators := []string{}
	for s, st := range p.Spec.Services {
		for _, t := range st {
			if t.Mode == "" {
				logrus.Warnf("Ignoring translator %+v for service %s due to empty mode", t, s)
				continue
			}
			if _, ok := p.Spec.Configuration.Translators[t.Name]; !ok {
				logrus.Debugf("Unable to find translator %s in configuration. Ignoring.", t.Name)
				continue
			}
			if !common.IsStringPresent(modes, t.Mode) {
				modes = append(modes, t.Mode)
			}
			if !common.IsStringPresent(translators, t.Name) {
				translators = append(translators, t.Name)
			}
		}
	}
	for _, t := range p.Spec.PlanTranslators {
		if _, ok := p.Spec.Configuration.Translators[t.Name]; !ok {
			logrus.Debugf("Unable to find translator %s in configuration. Ignoring.", t.Name)
			continue
		}
		if !common.IsStringPresent(translators, t.Name) {
			translators = append(translators, t.Name)
		}
	}
	modes = qaengine.FetchMultiSelectAnswer(common.ConfigModesKey, "Choose modes to use:", []string{"Modes generally specify the deployment model"}, modes, modes)
	translators = qaengine.FetchMultiSelectAnswer(common.ConfigTranslatorsKey, "Choose translators to use:", []string{"Translators are those that does the conversion"}, translators, translators)
	uTranslators := []string{}
	for sn, st := range p.Spec.Services {
		mode := ""
		exclusiveArtifactTypes := []string{}
		sTranslators := []plantypes.Translator{}
		for _, t := range st {
			if mode == "" {
				if t.Mode == "" {
					logrus.Warnf("Ignoring translator %+v for service %s due to empty mode", t, sn)
					continue
				}
				if !common.IsStringPresent(modes, t.Mode) {
					logrus.Debugf("Ignoring translator %+v for service %s due to deselected mode %s", t, sn, t.Mode)
					continue
				}
				if !common.IsStringPresent(translators, t.Name) {
					logrus.Debugf("Ignoring translator %+v for service %s due to deselected translator %s", t, sn, t.Mode)
					continue
				}
				mode = t.Mode
			} else if mode != t.Mode {
				logrus.Debugf("Ingoring %+v for service %s due to differing mode", t, sn)
			}
			if !common.IsStringPresent(translators, t.Name) {
				logrus.Debugf("Ignoring translator %+v for service %s due to deselected translator %s", t, sn, t.Mode)
				continue
			}
			if !common.IsStringPresent(uTranslators, t.Name) {
				uTranslators = append(translators, t.Name)
			}
			artifactsToUse := []string{}
			for _, at := range t.ArtifactTypes {
				if common.IsStringPresent(exclusiveArtifactTypes, at) {
					continue
				}
				artifactsToUse = append(artifactsToUse, at)
			}
			t.ArtifactTypes = artifactsToUse
			exclusiveArtifactTypes = append(exclusiveArtifactTypes, t.ExclusiveArtifactTypes...)
			sTranslators = append(sTranslators, t)
		}
		if mode != "" {
			modes = append(modes, mode)
		}
		if len(sTranslators) == 0 {
			logrus.Errorf("No translators selected for service %s. Ignoring.", sn)
			continue
		}
		p.Spec.Services[sn] = sTranslators
	}
	for _, t := range p.Spec.PlanTranslators {
		if _, ok := p.Spec.Configuration.Translators[t.Name]; !ok {
			logrus.Debugf("Unable to find translator %s in configuration. Ignoring.", t.Name)
			continue
		}
		if !common.IsStringPresent(translators, t.Name) {
			logrus.Debugf("Plan translator %s has been unselected. Ignoring.", t.Name)
			continue
		}
		uTranslators = append(uTranslators, t.Name)
	}
	tcs := map[string]string{}
	for _, tn := range uTranslators {
		if t, ok := p.Spec.Configuration.Translators[tn]; ok {
			tcs[tn] = t
		} else {
			logrus.Errorf("Unable to find configuration for translator %s", tn)
		}
	}
	translator.InitTranslators(tcs, p.Spec.RootDir)

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
