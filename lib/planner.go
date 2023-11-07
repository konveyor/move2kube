package lib

import (
	"context"
	"fmt"
	"github.com/konveyor/move2kube-wasm/transformer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// "fmt"

	"github.com/konveyor/move2kube-wasm/common"
	// "github.com/konveyor/move2kube/common/vcs"
	plantypes "github.com/konveyor/move2kube-wasm/types/plan"
	"github.com/sirupsen/logrus"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreatePlan creates the plan using all the tranformers.
func CreatePlan(ctx context.Context, inputPath, outputPath string, customizationsPath, transformerSelector, prjName string) (plantypes.Plan, error) {
	logrus.Trace("CreatePlan start")
	defer logrus.Trace("CreatePlan end")
	// remoteInputFSPath := vcs.GetClonedPath(inputPath, common.RemoteSourcesFolder, true)
	// remoteOutputFSPath := vcs.GetClonedPath(outputPath, common.RemoteOutputsFolder, true)
	// logrus.Debugf("common.TempPath: '%s' inputPath: '%s' remoteInputFSPath '%s'", common.TempPath, inputPath, remoteInputFSPath)
	plan := plantypes.NewPlan()
	plan.Name = prjName
	common.ProjectName = prjName
	plan.Spec.SourceDir = inputPath
	plan.Spec.CustomizationsDir = customizationsPath
	inputFSPath := inputPath
	outputFSPath := outputPath
	// if remoteInputFSPath != "" {
	// 	inputFSPath = remoteInputFSPath
	// }
	// if remoteOutputFSPath != "" {
	// 	outputFSPath = remoteOutputFSPath
	// }
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
	deselectedTransformers, err := transformer.Init(common.AssetsPath, inputFSPath, lblSelector, outputFSPath, plan.Name)
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
	if inputFSPath != "" {
		var err error
		plan.Spec.Services, err = transformer.GetServices(plan.Name, inputFSPath, nil)
		if err != nil {
			return plan, fmt.Errorf("failed to get services from the input directory '%s' . Error: %w", inputFSPath, err)
		}
	}
	logrus.Infof("Planning done. Number of services identified: %d", len(plan.Spec.Services))
	return plan, nil
}
