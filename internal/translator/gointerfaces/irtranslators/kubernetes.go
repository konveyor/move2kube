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
	K8sArtifacts = "KubernetesYamls"
)

// Kubernetes implements Translator interface
type Kubernetes struct {
}

type KubernetesConfig struct {
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
	}}
	return ts, nil
}

func (t *Kubernetes) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error) {
	return nil, nil
}

func (t *Kubernetes) TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) (pathMappings []translatortypes.PathMapping, err error) {
	logrus.Debugf("Translating IR using Kubernetes translator")
	targetCluster, err := new(configuration.ClusterMDLoader).GetTargetClusterMetadataForPlan(plan)
	if err != nil {
		err := fmt.Errorf("unable to find target cluster : %+v", plan.Spec.TargetCluster)
		logrus.Errorf("%s", err)
		return nil, err
	}
	tempDest := filepath.Join(tempOutputDir, common.DeployDir, "yamls")
	logrus.Debugf("Starting Kubernetes transform")
	logrus.Debugf("Total services to be transformed : %d", len(ir.Services))
	apis := []apiresource.IAPIResource{&apiresource.Deployment{}, &apiresource.Storage{}, &apiresource.Service{}, &apiresource.ImageStream{}, &apiresource.NetworkPolicy{}}
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
