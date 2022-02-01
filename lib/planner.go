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
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//CreatePlan creates the plan from all planners
func CreatePlan(ctx context.Context, inputPath, outputPath string, customizationsPath, transformerSelector, prjName string) plantypes.Plan {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	p := plantypes.NewPlan()
	p.Name = prjName
	common.ProjectName = prjName
	p.Spec.SourceDir = inputPath
	p.Spec.CustomizationsDir = customizationsPath
	if customizationsPath != "" {
		CheckAndCopyCustomizations(customizationsPath)
	}
	transformerSelectorObj, err := metav1.ParseToLabelSelector(transformerSelector)
	if err != nil {
		logrus.Errorf("Unable to parse the transformer selector string : %s", err)
	} else {
		p.Spec.TransformerSelector = *transformerSelectorObj
	}
	lblSelector, err := metav1.LabelSelectorAsSelector(transformerSelectorObj)
	if err != nil {
		logrus.Errorf("Unable to convert label selector to selector : %s", err)
	}
	transformer.Init(common.AssetsPath, inputPath, lblSelector, outputPath, p.Name)
	ts := transformer.GetInitializedTransformers()
	for _, t := range ts {
		config, _ := t.GetConfig()
		p.Spec.Transformers[config.Name] = config.Spec.FilePath
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
	common.ProjectName = p.Name
	logrus.Debugf("Temp Dir : %s", common.TempPath)

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
	transformer.InitTransformers(p.Spec.Transformers, transformerSelectorObj, p.Spec.SourceDir, outputPath, p.Name, true)
	serviceNames := []string{}
	for sn, st := range p.Spec.Services {
		sArtifacts := []plantypes.PlanArtifact{}
		for _, t := range st {
			if _, err := transformer.GetTransformerByName(t.TransformerName); err == nil {
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
	selectedServices := qaengine.FetchMultiSelectAnswer(common.ConfigServicesNamesKey, "Select all services that are needed:", []string{"The services unselected here will be ignored."}, serviceNames, serviceNames)
	planServices := map[string][]plantypes.PlanArtifact{}
	for _, s := range selectedServices {
		planServices[s] = p.Spec.Services[s]
	}
	p.Spec.Services = planServices

	logrus.Debugf("Plan : %+v", p)
	return p
}
