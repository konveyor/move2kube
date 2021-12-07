/*
 *  Copyright IBM Corporation 2020, 2021
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

package lib

import (
	"context"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/configuration"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CreatePlan creates the plan from all planners
func CreatePlan(ctx context.Context, inputPath, outputPath string, customizationsPath, transformerSelector, prjName string) plantypes.Plan {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	p := plantypes.NewPlan()
	p.Name = prjName
	p.Spec.RootDir = inputPath
	p.Spec.CustomizationsDir = customizationsPath
	if customizationsPath != "" {
		CheckAndCopyCustomizations(customizationsPath)
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
	tc, err := (&configuration.ClusterMDLoader{}).GetTargetClusterMetadataForPlan(p)
	if err != nil {
		logrus.Errorf("Unable to load cluster metadata : %s", err)
	}
	transformerSelectorObj, err := metav1.ParseToLabelSelector(transformerSelector)
	if err != nil {
		logrus.Errorf("Unable to parse the transformer selector string : %s", err)
	} else {
		p.Spec.TransformerSelector = *transformerSelectorObj
	}
	lblSelector, err := metav1.LabelSelectorAsSelector(transformerSelectorObj)
	if err != nil {
		logrus.Errorf("Unable to conver label selector to selector : %s", err)
	}
	transformer.Init(common.AssetsPath, inputPath, lblSelector, tc, outputPath, p.Name)
	ts := transformer.GetInitializedTransformers()
	for tn, t := range ts {
		config, _ := t.GetConfig()
		p.Spec.Configuration.Transformers[tn] = config.Spec.FilePath
	}
	logrus.Infoln("Configuration loading done")

	p.Spec.Services, err = transformer.GetServices(p.Name, inputPath)
	if err != nil {
		logrus.Errorf("Unable to create plan : %s", err)
	}
	logrus.Infof("No of services identified : %d", len(p.Spec.Services))
	return p
}

// CuratePlan allows curation the plan with the qa engine
func CuratePlan(p plantypes.Plan, outputPath, transformerSelector string) plantypes.Plan {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	transformers := []string{}
	for tn := range p.Spec.Configuration.Transformers {
		if !common.IsStringPresent(transformers, tn) {
			transformers = append(transformers, tn)
		}
	}
	serviceNames := []string{}
	transformers = qaengine.FetchMultiSelectAnswer(common.ConfigTransformerTypesKey, "Select all transformer types that you are interested in:", []string{"Services that don't support any of the transformer types you are interested in will be ignored."}, transformers, transformers)
	for sn, st := range p.Spec.Services {
		sArtifacts := []transformertypes.Artifact{}
		for _, t := range st {
			if common.IsStringPresent(transformers, t.Artifact) {
				sArtifacts = append(sArtifacts, t)
				break
			}
			logrus.Debugf("Ignoring transformer %+v for service %s due to deselected transformer", t, sn)
		}
		if len(sArtifacts) == 0 {
			logrus.Warnf("No transformers selected for service %s. Ignoring.", sn)
			delete(p.Spec.Services, sn)
			continue
		}
		p.Spec.Services[sn] = sArtifacts
		serviceNames = append(serviceNames, sn)
	}

	// Choose cluster type to target
	clusters := new(configuration.ClusterMDLoader).GetClusters(p)
	clusterTypeList := []string{}
	for c := range clusters {
		clusterTypeList = append(clusterTypeList, c)
	}
	clusterType := qaengine.FetchSelectAnswer(common.ConfigTargetClusterTypeKey, "Choose the cluster type:", []string{"Choose the cluster type you would like to target"}, string(common.DefaultClusterType), clusterTypeList)
	p.Spec.TargetCluster.Type = clusterType
	p.Spec.TargetCluster.Path = ""

	tc, err := (&configuration.ClusterMDLoader{}).GetTargetClusterMetadataForPlan(p)
	if err != nil {
		logrus.Errorf("Unable to load cluster metadata : %s", err)
	}
	transformerSelectorObj, err := common.ConvertStringSelectorsToSelectors(transformerSelector)
	if err != nil {
		logrus.Errorf("Unable to parse the transformer selector string : %s", err)
	}
	selectorsInPlan, err := metav1.LabelSelectorAsSelector(&p.Spec.TransformerSelector)
	if err != nil {
		logrus.Errorf("Unable to convert label selector to selector : %s", err)
	} else {
		requirements, _ := selectorsInPlan.Requirements()
		transformerSelectorObj = transformerSelectorObj.Add(requirements...)
	}

	transformer.InitTransformers(p.Spec.Configuration.Transformers, transformerSelectorObj, tc, p.Spec.RootDir, outputPath, p.Name, true)
	selectedServices := qaengine.FetchMultiSelectAnswer(common.ConfigServicesNamesKey, "Select all services that are needed:", []string{"The services unselected here will be ignored."}, serviceNames, serviceNames)
	planServices := map[string][]transformertypes.Artifact{}
	for _, s := range selectedServices {
		planServices[s] = p.Spec.Services[s]
	}
	p.Spec.Services = planServices

	logrus.Debugf("Plan : %+v", p)
	return p
}
