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

// S2IGenerator implements Transformer interface
type S2IGenerator struct {
	TConfig transformertypes.Transformer
	Env     environment.Environment
}

// Init Initializes the transformer
func (t *S2IGenerator) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *S2IGenerator) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

// BaseDirectoryDetect runs detect in the base directory
func (t *S2IGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *S2IGenerator) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *S2IGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	newartifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		tc := artifacts.S2IMetadataConfig{}
		err := a.GetConfig(artifacts.S2IMetadataConfigType, &tc)
		if err != nil {
			logrus.Errorf("Unable to read S2I Template config : %s", err)
		}
		relSrcPath, err := filepath.Rel(t.Env.GetWorkspaceSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		tc.ImageName = a.Configs[artifacts.ServiceArtifactType].(artifacts.ServiceConfig).ServiceName
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.TConfig.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: tc,
		})
		newartifacts = append(newartifacts, transformertypes.Artifact{
			Name:     tc.ImageName,
			Artifact: artifacts.NewImageArtifactType,
			Configs: map[string]interface{}{
				artifacts.NewImageConfigType: artifacts.NewImage{
					ImageName: tc.ImageName,
				},
			},
		})
	}
	return pathMappings, newartifacts, nil
}
