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

// DockerfileImageBuildScript implements Transformer interface
type DockerfileImageBuildScript struct {
	TConfig transformertypes.Transformer
	Env     environment.Environment
}

// DockerfileImageBuildScriptTemplateConfig represents template config used by ImagePush script
type DockerfileImageBuildScriptTemplateConfig struct {
	DockerfileName string
	ImageName      string
	Context        string
}

// Init Initializes the transformer
func (t *DockerfileImageBuildScript) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *DockerfileImageBuildScript) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *DockerfileImageBuildScript) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *DockerfileImageBuildScript) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *DockerfileImageBuildScript) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	dfs := []DockerfileImageBuildScriptTemplateConfig{}
	for _, a := range newArtifacts {
		if a.Artifact == artifacts.DockerfileArtifactType {
			for _, path := range a.Paths[artifacts.DockerfilePathType] {
				relPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), filepath.Dir(path))
				if err != nil {
					logrus.Errorf("Unable to make path relative : %s", err)
					continue
				}
				df := DockerfileImageBuildScriptTemplateConfig{
					ImageName:      a.Name,
					Context:        filepath.Join(common.DefaultSourceDir, relPath),
					DockerfileName: filepath.Base(path),
				}
				dfs = append(dfs, df)
			}
		}
	}
	if len(dfs) == 0 {
		return nil, nil, nil
	}
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.TConfig.Spec.TemplatesDir),
		DestPath:       common.ScriptsDir,
		TemplateConfig: dfs,
	})
	artifacts := []transformertypes.Artifact{{
		Name:     artifacts.DockerImageBuildScriptArtifactType,
		Artifact: artifacts.DockerImageBuildScriptArtifactType,
		Paths: map[string][]string{artifacts.DockerImageBuildShScriptPathType: {filepath.Join(common.ScriptsDir, "builddockerimages.sh")},
			artifacts.DockerImageBuildBatScriptPathType: {filepath.Join(common.ScriptsDir, "builddockerimages.bat")}},
	}}
	return pathMappings, artifacts, nil
}
