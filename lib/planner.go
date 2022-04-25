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

	logrus.Infoln("Start planning")
	p.Spec.Services, err = transformer.GetServices(p.Name, inputPath)
	if err != nil {
		logrus.Errorf("Unable to create plan : %s", err)
	}
	logrus.Infoln("Planning done")
	logrus.Infof("No of services identified : %d", len(p.Spec.Services))
	return p
}
