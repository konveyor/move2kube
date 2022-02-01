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
	"sort"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Transform transforms the artifacts and writes output
func Transform(ctx context.Context, plan plantypes.Plan, outputPath string, transformerSelector string) {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	logrus.Infof("Starting Plan Transformation")

	common.ProjectName = plan.Name
	logrus.Debugf("Temp Dir : %s", common.TempPath)

	transformerSelectorObj, err := common.ConvertStringSelectorsToSelectors(transformerSelector)
	if err != nil {
		logrus.Errorf("Unable to parse the transformer selector string : %s", err)
	}
	selectorsInPlan, err := metav1.LabelSelectorAsSelector(&plan.Spec.TransformerSelector)
	if err != nil {
		logrus.Errorf("Unable to convert label selector to selector : %s", err)
	} else {
		requirements, _ := selectorsInPlan.Requirements()
		transformerSelectorObj = transformerSelectorObj.Add(requirements...)
	}
	transformer.InitTransformers(plan.Spec.Transformers, transformerSelectorObj, plan.Spec.SourceDir, outputPath, plan.Name, true)
	serviceNames := []string{}
	planServices := map[string]plantypes.PlanArtifact{}
	for sn, st := range plan.Spec.Services {
		sArtifacts := []plantypes.PlanArtifact{}
		for _, t := range st {
			if _, err := transformer.GetTransformerByName(t.TransformerName); err == nil {
				serviceNames = append(serviceNames, sn)
				t.ServiceName = sn
				planServices[sn] = t
				break
			}
			logrus.Debugf("Ignoring transformer %+v for service %s due to deselected transformer", t, sn)
		}
		if len(sArtifacts) == 0 {
			logrus.Warnf("No transformers selected for service %s. Ignoring.", sn)
		}
	}
	sort.Strings(serviceNames)
	selectedServices := qaengine.FetchMultiSelectAnswer(common.ConfigServicesNamesKey, "Select all services that are needed:", []string{"The services unselected here will be ignored."}, serviceNames, serviceNames)
	selectedPlanServices := []plantypes.PlanArtifact{}
	for _, s := range selectedServices {
		selectedPlanServices = append(selectedPlanServices, planServices[s])
	}
	err = transformer.Transform(selectedPlanServices, plan.Spec.SourceDir, outputPath)
	if err != nil {
		logrus.Fatalf("Failed to transform the plan. Error: %q", err)
	}
	logrus.Infof("Plan Transformation done")
}

// Destroy destroys the tranformers
func Destroy() {
	logrus.Debugf("Cleaning up!")
	transformer.Destroy()
}
