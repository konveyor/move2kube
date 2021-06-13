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
	"path/filepath"

	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/configuration"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
)

const (
	KnativeArtifacts = "KnativeYamls"
)

// Knative implements Translator interface
type Knative struct {
	Config translatortypes.Translator
}

type KnativeConfig struct {
}

func (t *Knative) Init(tc translatortypes.Translator) error {
	t.Config = tc
	return nil
}

func (t *Knative) GetConfig() translatortypes.Translator {
	return t.Config
}

func (t *Knative) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Knative) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Knative) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *Knative) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	ts := []plantypes.Translator{{
		Mode:          plantypes.ModeContainer,
		ArtifactTypes: []string{KnativeArtifacts},
	}}
	return ts, nil
}

func (t *Knative) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error) {
	return nil, nil
}

func (t *Knative) TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) (pathMappings []translatortypes.PathMapping, err error) {
	logrus.Debugf("Translating IR using Kubernetes translator")
	targetCluster, err := new(configuration.ClusterMDLoader).GetTargetClusterMetadataForPlan(plan)
	if err != nil {
		err := fmt.Errorf("unable to find target cluster : %+v", plan.Spec.TargetCluster)
		logrus.Errorf("%s", err)
		return nil, err
	}
	tempDest := filepath.Join(tempOutputDir, common.DeployDir, "knative")
	logrus.Debugf("Starting Kubernetes transform")
	logrus.Debugf("Total services to be transformed : %d", len(ir.Services))
	apis := []apiresource.IAPIResource{&apiresource.KnativeService{}}
	if files, err := apiresource.TransformAndPersist(irtypes.NewEnhancedIRFromIR(ir), tempDest, apis, targetCluster); err == nil {
		for _, f := range files {
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
		logrus.Debugf("Total transformed objects : %d", len(files))
	} else {
		logrus.Errorf("Unable to translate and persist IR : %s", err)
		return nil, err
	}
	return pathMappings, nil
}
