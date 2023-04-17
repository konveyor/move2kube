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
	"os"
	"sort"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/common/vcs"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer"
	"github.com/konveyor/move2kube/transformer/external"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Transform transforms the artifacts and writes output
func Transform(ctx context.Context, plan plantypes.Plan, preExistingPlan bool, outputPath string, transformerSelector string, maxIterations int) error {
	logrus.Infof("Starting transformation")

	common.ProjectName = plan.Name
	logrus.Debugf("common.TempPath: '%s'", common.TempPath)

	transformerSelectorObj, err := common.ConvertStringSelectorsToSelectors(transformerSelector)
	if err != nil {
		return fmt.Errorf("failed to parse the transformer selector string. Error: %w", err)
	}
	selectorsInPlan, err := metav1.LabelSelectorAsSelector(&plan.Spec.TransformerSelector)
	if err != nil {
		return fmt.Errorf("failed to convert label selector to selector. Error: %w", err)
	}
	requirements, _ := selectorsInPlan.Requirements()
	transformerSelectorObj = transformerSelectorObj.Add(requirements...)

	remoteOutputFSPath := vcs.GetInputClonedPath(outputPath, common.RemoteOutputsFolder, true)
	outputFSPath := outputPath
	if remoteOutputFSPath != "" {
		outputFSPath = remoteOutputFSPath
	}

	if _, err := transformer.InitTransformers(
		plan.Spec.Transformers,
		transformerSelectorObj,
		plan.Spec.SourceDir,
		outputFSPath,
		plan.Name,
		true,
		preExistingPlan,
	); err != nil {
		return fmt.Errorf("failed to initialize the transformers. Error: %w", err)
	}

	// select only the services the user is interested in
	serviceNames := []string{}
	for serviceName := range plan.Spec.Services {
		serviceNames = append(serviceNames, serviceName)
	}
	sort.Strings(serviceNames)
	selectedServiceNames := qaengine.FetchMultiSelectAnswer(
		common.ConfigServicesNamesKey,
		"Select all services that are needed:",
		[]string{"The services unselected here will be ignored."},
		serviceNames,
		serviceNames,
		nil,
	)

	// select the first valid transformation option for each selected service
	selectedTransformationOptions := []plantypes.PlanArtifact{}
	for _, selectedServiceName := range selectedServiceNames {
		options := plan.Spec.Services[selectedServiceName]
		found := false
		if len(options) > 1 {
			logrus.Infof("Found multiple transformation options for the service '%s'. Selecting the first valid option.", selectedServiceName)
		}
		for _, option := range options {
			if _, err := transformer.GetTransformerByName(option.TransformerName); err != nil {
				logrus.Errorf("failed to get the transformer named '%s' for the service '%s' . Error: %q", option.TransformerName, selectedServiceName, err)
				continue
			}
			option.ServiceName = selectedServiceName
			selectedTransformationOptions = append(selectedTransformationOptions, option)
			logrus.Infof("Using the transformation option '%s' for the service '%s'.", option.TransformerName, selectedServiceName)
			found = true
			break
		}
		if !found {
			logrus.Warnf("No valid transformers were found for the service '%s'. Skipping.", selectedServiceName)
		}
	}

	// transform the selected services using the selected transformation options
	if err := transformer.Transform(selectedTransformationOptions, plan.Spec.SourceDir, outputFSPath, maxIterations); err != nil {
		return fmt.Errorf("failed to transform using the plan. Error: %w", err)
	}

	logrus.Infof("Transformation done")

	if vcs.IsRemotePath(outputPath) {
		err := vcs.PushVCSRepo(outputPath, common.RemoteOutputsFolder)
		if err != nil {
			logrus.Fatalf("failed to commit and push the output artifacts for the given remote path %s. Errro : %+v", outputPath, err)
		}
		logrus.Infof("move2kube generated artifcats are commited and pushed")
	}
	return nil
}

// Destroy destroys the tranformers
func Destroy() {
	logrus.Debugf("Cleaning up!")
	transformer.Destroy()
	err := os.RemoveAll(common.TempPath)
	if err != nil {
		logrus.Debug("failed to delete temp directory. Error : ", err)
	}

	err = os.RemoveAll(external.DetectContainerOutputDir)
	if err != nil {
		logrus.Debug("failed to delete temp directory. Error : ", err)
	}

	err = os.RemoveAll(external.TransformContainerOutputDir)
	if err != nil {
		logrus.Debug("failed to delete temp directory. Error : ", err)
	}

}
