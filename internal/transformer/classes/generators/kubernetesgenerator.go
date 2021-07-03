/*
 *  Copyright IBM Corporation 2021
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

package generators

import (
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// Kubernetes implements Transformer interface
type Kubernetes struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

// Init Initializes the transformer
func (t *Kubernetes) Init(tc transformertypes.Transformer, e environment.Environment) error {
	t.Config = tc
	t.Env = e
	return nil
}

// GetConfig returns the transformer config
func (t *Kubernetes) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *Kubernetes) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each subdirectory
func (t *Kubernetes) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// Transform transforms artifacts
func (t *Kubernetes) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Debugf("Translating IR using Kubernetes transformer")
	pathMappings = []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		if a.Artifact != irtypes.IRArtifactType {
			continue
		}
		var ir irtypes.IR
		if err := a.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", ir, err)
			continue
		}
		var pC artifacts.PlanConfig
		if err = a.GetConfig(artifacts.PlanConfigType, &pC); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", pC, err)
			continue
		}
		tempDest := filepath.Join(t.Env.TempPath, common.DeployDir, "yamls")
		logrus.Debugf("Starting Kubernetes transform")
		logrus.Debugf("Total services to be transformed : %d", len(ir.Services))
		apis := []apiresource.IAPIResource{new(apiresource.Deployment), new(apiresource.Storage), new(apiresource.Service), new(apiresource.ImageStream), new(apiresource.NetworkPolicy)}
		files, err := apiresource.TransformAndPersist(irtypes.NewEnhancedIRFromIR(ir), tempDest, apis, pC.TargetCluster)
		if err != nil {
			logrus.Errorf("Unable to transform and persist IR : %s", err)
			return nil, nil, err
		}
		for _, f := range files {
			destPath, err := filepath.Rel(t.Env.TempPath, f)
			if err != nil {
				logrus.Errorf("Invalid yaml path : %s", destPath)
				continue
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  f,
				DestPath: destPath,
			})
		}
		logrus.Debugf("Total transformed objects : %d", len(files))
	}
	return pathMappings, nil, nil
}
