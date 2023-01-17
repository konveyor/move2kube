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
	"fmt"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/transformer"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePlan creates the plan using all the tranformers.
func CreatePlan(ctx context.Context, inputPath, outputPath string, customizationsPath, transformerSelector, prjName string) (plantypes.Plan, error) {
	logrus.Trace("CreatePlan start")
	defer logrus.Trace("CreatePlan end")
	logrus.Debugf("common.TempPath: '%s' inputPath: '%s'", common.TempPath, inputPath)
	plan := plantypes.NewPlan()
	plan.Name = prjName
	common.ProjectName = prjName
	plan.Spec.SourceDir = inputPath
	plan.Spec.CustomizationsDir = customizationsPath
	if customizationsPath != "" {
		if err := CheckAndCopyCustomizations(customizationsPath); err != nil {
			return plan, fmt.Errorf("failed to check and copy the customizations. Error: %w", err)
		}
	}
	transformerSelectorObj, err := metav1.ParseToLabelSelector(transformerSelector)
	if err != nil {
		return plan, fmt.Errorf("failed to parse the string '%s' as a transformer selector. Error: %w", transformerSelector, err)
	}
	plan.Spec.TransformerSelector = *transformerSelectorObj

	lblSelector, err := metav1.LabelSelectorAsSelector(transformerSelectorObj)
	if err != nil {
		return plan, fmt.Errorf("failed to convert the label selector to a selector. Error: %w", err)
	}
	deselectedTransformers, err := transformer.Init(common.AssetsPath, inputPath, lblSelector, outputPath, plan.Name)
	if err != nil {
		return plan, fmt.Errorf("failed to initialize the transformers. Error: %w", err)
	}
	plan.Spec.DisabledTransformers = deselectedTransformers
	transformers := transformer.GetInitializedTransformers()
	for _, transformer := range transformers {
		config, _ := transformer.GetConfig()
		plan.Spec.Transformers[config.Name] = config.Spec.TransformerYamlPath
		if config.Spec.InvokedByDefault.Enabled {
			logrus.Debugf("adding a default transformer to the plan file: %+v", config)
			plan.Spec.InvokedByDefaultTransformers = append(plan.Spec.InvokedByDefaultTransformers, config.Name)
		}
	}
	logrus.Info("Configuration loading done")

	logrus.Info("Start planning")
	if inputPath != "" {
		plan.Spec.Services, err = transformer.GetServices(plan.Name, inputPath)
		if err != nil {
			return plan, fmt.Errorf("failed to get services from the input directory '%s' . Error: %w", inputPath, err)
		}
	}
	logrus.Infof("Planning done. Number of services identified: %d", len(plan.Spec.Services))
	return plan, nil
}
