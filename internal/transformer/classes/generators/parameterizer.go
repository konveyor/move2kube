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
	"github.com/konveyor/move2kube/parameterizer"
	parameterizertypes "github.com/konveyor/move2kube/types/parameterizer"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// Parameterizer implements Transformer interface
type Parameterizer struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *Parameterizer) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	return nil
}

// GetConfig returns the transformer config
func (t *Parameterizer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *Parameterizer) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each subdirectory
func (t *Parameterizer) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// Transform transforms artifacts
func (t *Parameterizer) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	psmap, err := parameterizer.CollectParamsFromPath(t.Env.Context)
	if err != nil {
		logrus.Errorf("Error while parsing for params : %s", err)
		return nil, nil, err
	}
	ps := []parameterizertypes.ParameterizerT{}
	for _, p := range psmap {
		ps = append(ps, p...)
	}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.KubernetesYamlsArtifactType {
			continue
		}
		yamlsPath := a.Paths[artifacts.KubernetesYamlsPathType][0]
		destPath := yamlsPath + "-parameterized"
		filesWritten, err := parameterizer.Parameterize(yamlsPath, destPath, parameterizertypes.PackagingSpecPathT{}, ps)
		if err != nil {
			logrus.Errorf("Unable to parameterize : %s", err)
		}
		for _, f := range filesWritten {
			rel, err := filepath.Rel(t.Env.GetEnvironmentOutput(), f)
			if err != nil {
				logrus.Errorf("Unable to make parameterized file path relative : %s", err)
				continue
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  f,
				DestPath: rel,
			})
		}
	}
	return pathMappings, nil, nil
}
