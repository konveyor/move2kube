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

package containerimage

import (
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	buildImagesFileName = "buildimages"
)

// ContainerImagesBuildScript implements Transformer interface
type ContainerImagesBuildScript struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// ImageBuildTemplateConfig represents template config used by ImagePush script
type ImageBuildTemplateConfig struct {
	BuildScript string
	PathUnix    string
	PathWindows string
}

// Init Initializes the transformer
func (t *ContainerImagesBuildScript) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *ContainerImagesBuildScript) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *ContainerImagesBuildScript) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *ContainerImagesBuildScript) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	shScripts := []ImageBuildTemplateConfig{}
	batScripts := []ImageBuildTemplateConfig{}
	for _, a := range append(newArtifacts, alreadySeenArtifacts...) {
		if a.Type != artifacts.ContainerImageBuildScriptArtifactType {
			continue
		}
		for _, shScript := range a.Paths[artifacts.ContainerImageBuildShScriptPathType] {
			contextPath := filepath.Dir(shScript)
			if ctxPath, ok := a.Paths[artifacts.ContainerImageBuildShScriptContextPathType]; ok {
				contextPath = ctxPath[0]
			}
			relPath, err := filepath.Rel(t.Env.GetEnvironmentOutput(), contextPath)
			if err != nil {
				logrus.Errorf("Unable to make path relative : %s", err)
				continue
			}
			scriptPath, err := filepath.Rel(contextPath, shScript)
			if err != nil {
				logrus.Errorf("Unable to make path relative : %s", err)
				continue
			}
			n := ImageBuildTemplateConfig{
				BuildScript: scriptPath,
				PathUnix:    common.GetUnixPath(relPath),
			}
			if t.scriptExists(shScripts, n) {
				continue
			}
			shScripts = append(shScripts, n)
		}
		for _, batScript := range a.Paths[artifacts.ContainerImageBuildBatScriptPathType] {
			contextPath := filepath.Dir(batScript)
			if ctxPath, ok := a.Paths[artifacts.ContainerImageBuildShScriptContextPathType]; ok {
				contextPath = ctxPath[0]
			}
			relPath, err := filepath.Rel(t.Env.GetEnvironmentOutput(), contextPath)
			if err != nil {
				logrus.Errorf("Unable to make path relative : %s", err)
				continue
			}
			scriptPath, err := filepath.Rel(contextPath, batScript)
			if err != nil {
				logrus.Errorf("Unable to make path relative : %s", err)
				continue
			}
			n := ImageBuildTemplateConfig{
				BuildScript: scriptPath,
				PathWindows: common.GetWindowsPath(relPath),
			}
			if t.scriptExists(batScripts, n) {
				continue
			}
			batScripts = append(batScripts, n)
		}
	}
	if len(shScripts) == 0 {
		return nil, nil, nil
	}
	buildImagesShFileName := buildImagesFileName + common.ShExt
	buildImagesBatFileName := buildImagesFileName + common.BatExt
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir, buildImagesShFileName),
		DestPath:       filepath.Join(common.ScriptsDir, buildImagesShFileName),
		TemplateConfig: shScripts,
	}, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir, buildImagesBatFileName),
		DestPath:       filepath.Join(common.ScriptsDir, buildImagesBatFileName),
		TemplateConfig: batScripts,
	})
	as := []transformertypes.Artifact{{
		Name: string(artifacts.ContainerImagesBuildScriptArtifactType),
		Type: artifacts.ContainerImagesBuildScriptArtifactType,
		Paths: map[transformertypes.PathType][]string{
			artifacts.ContainerImagesBuildShScriptPathType:  {filepath.Join(common.ScriptsDir, buildImagesShFileName)},
			artifacts.ContainerImagesBuildBatScriptPathType: {filepath.Join(common.ScriptsDir, buildImagesBatFileName)}},
	}}
	return pathMappings, as, nil
}

func (t *ContainerImagesBuildScript) scriptExists(configs []ImageBuildTemplateConfig, newconfig ImageBuildTemplateConfig) bool {
	for _, config := range configs {
		if config == newconfig {
			return true
		}
	}
	return false
}
