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

package dockerfile

import (
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	rubyFileExt = ".rb"
)

// RubyDockerfileGenerator implements the Transformer interface
type RubyDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// RubyTemplateConfig implements Ruby config interface
type RubyTemplateConfig struct {
	Port    int32
	AppName string
}

// Init Initializes the transformer
func (t *RubyDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *RubyDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *RubyDockerfileGenerator) BaseDirectoryDetect(dir string) (services map[string][]transformertypes.TransformerPlan, err error) {
	return nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *RubyDockerfileGenerator) DirectoryDetect(dir string) (services map[string][]transformertypes.TransformerPlan, err error) {
	Gemfiles, err := common.GetFilesByName(dir, []string{"Gemfile"}, nil)
	if err != nil {
		logrus.Debugf("Cannot get the Gemfile: %s", err)
		return nil, nil
	}
	if len(Gemfiles) > 0 {
		rubyFiles, err := common.GetFilesByExt(dir, []string{rubyFileExt})
		if err != nil {
			logrus.Errorf("Error while finding ruby files %s", err)
		}
		var serviceName string
		if len(rubyFiles) == 1 {
			serviceName = strings.TrimSuffix(filepath.Base(rubyFiles[0]), rubyFileExt)
		}
		services = map[string][]transformertypes.TransformerPlan{
			serviceName: {{
				Mode:              t.Config.Spec.Mode,
				ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
				BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
				Paths: map[string][]string{
					artifacts.ProjectPathPathType: {dir},
				},
			}},
		}
		return services, nil
	}
	return nil, nil
}

// Transform transforms the artifacts
func (t *RubyDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	artifactsCreated := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		var rubyConfig RubyTemplateConfig
		var detectedPorts []int32
		detectedPorts = append(detectedPorts, 8080) //TODO: Write parser to parse and identify port
		rubyConfig.Port = commonqa.GetPortForService(detectedPorts, a.Name)
		rubyConfig.AppName = a.Name
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: rubyConfig,
		})
		paths := a.Paths
		paths[artifacts.DockerfilePathType] = []string{filepath.Join(common.DefaultSourceDir, relSrcPath, common.DefaultDockerfileName)}
		p := transformertypes.Artifact{
			Name:     sImageName.ImageName,
			Artifact: artifacts.DockerfileArtifactType,
			Paths:    paths,
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:     sConfig.ServiceName,
			Artifact: artifacts.DockerfileForServiceArtifactType,
			Paths:    a.Paths,
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
				artifacts.ServiceConfigType:   sConfig,
			},
		}
		artifactsCreated = append(artifactsCreated, p, dfs)
	}
	return pathMappings, artifactsCreated, nil
}
