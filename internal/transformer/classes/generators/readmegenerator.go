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
	"github.com/konveyor/move2kube/internal/common"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// ReadMeGenerator implements Transformer interface
type ReadMeGenerator struct {
	TConfig transformertypes.Transformer
	Env     environment.Environment
}

// Init initializes the translator
func (t *ReadMeGenerator) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	return nil
}

// GetConfig returns the config of the transformer
func (t *ReadMeGenerator) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

// BaseDirectoryDetect executes detect in base directory
func (t *ReadMeGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect executes detect in directories respecting the m2kignore
func (t *ReadMeGenerator) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *ReadMeGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	images := []string{}
	for _, a := range newArtifacts {
		if a.Artifact == artifacts.ImagePushScriptArtifactType {
			image := artifacts.NewImage{}
			err := a.GetConfig(artifacts.NewImageConfigType, &image)
			if err != nil {
				logrus.Errorf("Unable to read Image config : %s", err)
			}
			if !common.IsStringPresent(images, image.ImageName) {
				images = append(images, image.ImageName)
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.TemplatePathMappingType,
				SrcPath:        filepath.Join(t.Env.Context, t.TConfig.Spec.TemplatesDir),
				DestPath:       common.ScriptsDir,
				TemplateConfig: images,
			})
		}
	}
	return pathMappings, nil, nil
}
