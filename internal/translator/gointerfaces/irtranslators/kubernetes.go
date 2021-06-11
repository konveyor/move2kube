/*
Copyright IBM Corporation 2021

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

package irtranslators

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/configuration"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	K8sArtifacts = "KubernetesYamls"
)

// Kubernetes implements Translator interface
type Kubernetes struct {
}

type KubernetesConfig struct {
	OutputRelPath string `yaml:"outputRelPath"`
}

func (t *Kubernetes) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Kubernetes) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Kubernetes) KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Kubernetes) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *Kubernetes) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	ts := []plantypes.Translator{{
		Mode:          plantypes.ModeContainer,
		ArtifactTypes: []string{K8sArtifacts},
		Config: KubernetesConfig{
			OutputRelPath: filepath.Join(common.DeployDir, "yamls"),
		},
	}}
	return ts, nil
}

func (t *Kubernetes) TranslateService(serviceName string, translatorPlan plantypes.Translator, tempOutputDir string) ([]translatortypes.Patch, error) {
	return nil, nil
}

func (t *Kubernetes) TranslateIR(ir irtypes.IR, destDir string, plan plantypes.Plan, tempOutputDir string) (pathMappings []translatortypes.PathMapping, err error) {
	logrus.Debugf("Translating IR using Kubernetes translator")
	var targetCluster collecttypes.ClusterMetadata
	if plan.Spec.TargetCluster.Path != "" {
		targetCluster, err = new(configuration.ClusterMDLoader).GetClusterMetadata(plan.Spec.TargetCluster.Path)
		if err != nil {
			logrus.Errorf("Unable to load cluster metadata from %s : %s", plan.Spec.TargetCluster.Path, err)
			return nil, err
		}
	} else if plan.Spec.TargetCluster.Type != "" {
		var ok bool
		targetCluster, ok = new(configuration.ClusterMDLoader).GetClusters(plan)[plan.Spec.TargetCluster.Type]
		if !ok {
			err = fmt.Errorf("unable to load cluster metadata from %s", plan.Spec.TargetCluster.Type)
			logrus.Errorf("%s", err)
			return nil, err
		}
	} else {
		err := fmt.Errorf("unable to find target cluster : %+v", plan.Spec.TargetCluster)
		logrus.Errorf("%s", err)
		return nil, err
	}

	transformPaths := []string{}
	for _, t := range plan.Spec.Configuration.Transformers {
		transformPaths = append(transformPaths, t)
	}

	logrus.Debugf("Starting Kubernetes transform")
	logrus.Debugf("Total services to be transformed : %d", len(ir.Services))
	apis := []apiresource.IAPIResource{&apiresource.Deployment{}, &apiresource.Storage{}, &apiresource.Service{}, &apiresource.ImageStream{}, &apiresource.NetworkPolicy{}}
	targetObjs := []runtime.Object{}
	for _, apiResource := range apis {
		newObjs := (&apiresource.APIResource{IAPIResource: apiResource}).ConvertIRToObjects(irtypes.NewEnhancedIRFromIR(ir), targetCluster)
		targetObjs = append(targetObjs, newObjs...)
	}
	tempDest := filepath.Join(tempOutputDir, destDir)
	if err := os.MkdirAll(tempDest, common.DefaultDirectoryPermission); err != nil {
		logrus.Errorf("Unable to create deploy directory at path %s Error: %q", tempOutputDir, err)
	}

	// deploy/yamls/
	logrus.Debugf("Total %d services to be serialized.", len(targetObjs))
	fixedConvertedTransformedObjs, err := fixConvertAndTransformObjs(targetObjs, targetCluster.Spec, transformPaths)
	if err != nil {
		logrus.Errorf("Failed to fix, convert and transform the objects. Error: %q", err)
	}
	if filesWritten, err := writeObjects(tempDest, fixedConvertedTransformedObjs); err != nil {
		logrus.Errorf("Failed to write the transformed objects to the directory at path %s . Error: %q", destDir, err)
		return nil, err
	} else {
		for _, f := range filesWritten {
			if destPath, err := filepath.Rel(tempOutputDir, f); err != nil {
				logrus.Errorf("Invalid yaml path : %s", destPath)
			} else {
				pathMappings = append(pathMappings, translatortypes.PathMapping{
					Type:     translatortypes.DefaultPathMappingType,
					SrcPath:  f,
					DestPath: destPath,
				})
			}
		}
	}
	logrus.Debugf("Total transformed objects : %d", len(targetObjs))
	return pathMappings, nil
}

func (t *Kubernetes) PathForIR(patch translatortypes.Patch, planTranslator plantypes.Translator) string {
	config := KubernetesConfig{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &config,
		TagName:  "yaml",
	})
	if err := decoder.Decode(planTranslator.Config); err != nil {
		logrus.Errorf("unable to load config for Translator %+v into %T : %s", planTranslator, config, err)
		return ""
	}
	return config.OutputRelPath
}
