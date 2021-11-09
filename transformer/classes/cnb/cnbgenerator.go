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

package cnb

import (
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// CNBGenerator implements Transformer interface
type CNBGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *CNBGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *CNBGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *CNBGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.TransformerPlan, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *CNBGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	newartifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		tc := artifacts.CNBMetadataConfig{}
		err := a.GetConfig(artifacts.CNBMetadataConfigType, &tc)
		if err != nil {
			logrus.Errorf("Unable to read CNB Template config : %s", err)
		}
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		if tc.ImageName == "" {
			tc.ImageName = common.MakeStringContainerImageNameCompliant(a.Configs[artifacts.ServiceArtifactType].(artifacts.ServiceConfig).ServiceName)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: tc,
		})
		newartifacts = append(newartifacts, transformertypes.Artifact{
			Name:     t.Env.ProjectName,
			Artifact: artifacts.NewImagesArtifactType,
			Configs: map[string]interface{}{
				artifacts.NewImagesConfigType: artifacts.NewImages{
					ImageNames: []string{tc.ImageName},
				},
			},
		}, transformertypes.Artifact{
			Name:     tc.ImageName,
			Artifact: artifacts.ContainerImageBuildScriptArtifactType,
			Paths: map[string][]string{
				artifacts.ContainerImageBuildShScriptPathType:  {filepath.Join(common.DefaultSourceDir, relSrcPath, "cnbbuild.sh")},
				artifacts.ContainerImageBuildBatScriptPathType: {filepath.Join(common.DefaultSourceDir, relSrcPath, "cnbbuild.bat")},
			},
		})
	}
	return pathMappings, newartifacts, nil
}
