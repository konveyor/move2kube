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
	Env     *environment.Environment
}

// DockerfileImageBuildScriptTemplateConfig represents template config used by ImagePush script
type DockerfileImageBuildScriptTemplateConfig struct {
	DockerfileName   string
	ImageName        string
	ContextToSlash   string
	ContextFromSlash string
}

// Init Initializes the transformer
func (t *DockerfileImageBuildScript) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *DockerfileImageBuildScript) GetConfig() (transformertypes.Transformer, *environment.Environment) {
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
	nartifacts := []transformertypes.Artifact{}
	processedImages := map[string]bool{}
	for _, a := range append(newArtifacts, oldArtifacts...) {
		if a.Artifact != artifacts.DockerfileArtifactType {
			continue
		}
		sImageName := artifacts.ImageName{}
		err := a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
		}
		if processedImages[sImageName.ImageName] {
			continue
		}
		processedImages[sImageName.ImageName] = true
		for _, path := range a.Paths[artifacts.DockerfilePathType] {
			relPath := filepath.Dir(path)
			if common.IsParent(path, t.Env.GetEnvironmentSource()) {
				relPath, err = filepath.Rel(t.Env.GetEnvironmentSource(), filepath.Dir(path))
				if err != nil {
					logrus.Errorf("Unable to make path relative : %s", err)
					continue
				}
				dfs = append(dfs, DockerfileImageBuildScriptTemplateConfig{
					ImageName:        sImageName.ImageName,
					ContextToSlash:   filepath.ToSlash(filepath.Join(common.DefaultSourceDir, relPath)),
					ContextFromSlash: filepath.FromSlash(filepath.Join(common.DefaultSourceDir, relPath)),
					DockerfileName:   filepath.Base(path),
				})
			} else if common.IsParent(path, t.Env.GetEnvironmentOutput()) {
				relPath, err = filepath.Rel(t.Env.GetEnvironmentOutput(), filepath.Dir(path))
				if err != nil {
					logrus.Errorf("Unable to make path relative : %s", err)
					continue
				}
				dfs = append(dfs, DockerfileImageBuildScriptTemplateConfig{
					ImageName:        sImageName.ImageName,
					ContextToSlash:   filepath.ToSlash(relPath),
					ContextFromSlash: filepath.FromSlash(relPath),
					DockerfileName:   filepath.Base(path),
				})
			} else {
				dfs = append(dfs, DockerfileImageBuildScriptTemplateConfig{
					ImageName:        sImageName.ImageName,
					ContextToSlash:   filepath.ToSlash(filepath.Join(common.DefaultSourceDir, relPath)),
					ContextFromSlash: filepath.FromSlash(filepath.Join(common.DefaultSourceDir, relPath)),
					DockerfileName:   filepath.Base(path),
				})
			}
			nartifacts = append(nartifacts, transformertypes.Artifact{
				Name:     t.Env.ProjectName,
				Artifact: artifacts.NewImagesArtifactType,
				Configs: map[string]interface{}{
					artifacts.NewImagesConfigType: artifacts.NewImages{
						ImageNames: []string{sImageName.ImageName},
					},
				},
			})
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
	nartifacts = append(nartifacts, transformertypes.Artifact{
		Name:     artifacts.ContainerImageBuildScriptArtifactType,
		Artifact: artifacts.ContainerImageBuildScriptArtifactType,
		Paths: map[string][]string{artifacts.ContainerImageBuildShScriptPathType: {filepath.Join(common.ScriptsDir, "builddockerimages.sh")},
			artifacts.ContainerImageBuildBatScriptPathType: {filepath.Join(common.ScriptsDir, "builddockerimages.bat")}},
	})
	return pathMappings, nartifacts, nil
}
