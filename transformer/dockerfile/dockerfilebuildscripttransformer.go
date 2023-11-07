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
	"fmt"
	"path/filepath"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	"github.com/konveyor/move2kube-wasm/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	buildImagesFileName                 = "buildimages"
	buildAndPushImagesFileName          = "buildandpushimages_multiarch"
	defaultDockerBuildScriptsOutputPath = common.ScriptsDir
)

// DockerfileImageBuildScript implements Transformer interface
type DockerfileImageBuildScript struct {
	Config                           transformertypes.Transformer
	Env                              *environment.Environment
	DockerfileImageBuildScriptConfig *DockerfileImageBuildScriptConfig
}

// DockerfileImageBuildScriptConfig stores the transformer specific configuration
type DockerfileImageBuildScriptConfig struct {
	OutputPath string `yaml:"outputPath"`
}

// DockerfileImageBuildScriptTemplateConfig represents the data used to fill the build script generator template
type DockerfileImageBuildScriptTemplateConfig struct {
	RelParentOfSourceDir string
	DockerfilesConfig    []DockerfileImageBuildConfig
	RegistryURL          string
	RegistryNamespace    string
}

// DockerfileImageBuildConfig contains the Dockerfile image build config to be used in the ImageBuild script
type DockerfileImageBuildConfig struct {
	DockerfileName string
	ImageName      string
	ContextUnix    string
	ContextWindows string
}

// Init Initializes the transformer
func (t *DockerfileImageBuildScript) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.DockerfileImageBuildScriptConfig = &DockerfileImageBuildScriptConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.DockerfileImageBuildScriptConfig); err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.DockerfileImageBuildScriptConfig, err)
		return err
	}
	if t.DockerfileImageBuildScriptConfig.OutputPath == "" {
		t.DockerfileImageBuildScriptConfig.OutputPath = defaultDockerBuildScriptsOutputPath
	}
	return nil
}

// GetConfig returns the transformer config
func (t *DockerfileImageBuildScript) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *DockerfileImageBuildScript) DirectoryDetect(dir string) (namedServices map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *DockerfileImageBuildScript) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	dockerfilesImageBuildConfig := []DockerfileImageBuildConfig{}
	createdArtifacts := []transformertypes.Artifact{}
	processedImages := map[string]bool{}
	for _, artifact := range append(alreadySeenArtifacts, newArtifacts...) {
		if artifact.Type != artifacts.DockerfileArtifactType {
			continue
		}
		imageName := artifacts.ImageName{}
		if err := artifact.GetConfig(artifacts.ImageNameConfigType, &imageName); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T . Error: %q", imageName, err)
			continue
		}
		if imageName.ImageName == "" {
			imageName.ImageName = common.MakeStringContainerImageNameCompliant(artifact.Name)
		}
		if processedImages[imageName.ImageName] {
			continue
		}
		processedImages[imageName.ImageName] = true
		var dockerfileImageBuildConfig DockerfileImageBuildConfig
		dockerfileImageBuildConfig.ImageName = imageName.ImageName
		for _, dockerfilePath := range artifact.Paths[artifacts.DockerfilePathType] {
			dockerContextPath := filepath.Dir(dockerfilePath)
			relDockerfilePath := filepath.Base(dockerfilePath)
			if len(artifact.Paths[artifacts.DockerfileContextPathType]) > 0 {
				dockerContextPath = artifact.Paths[artifacts.DockerfileContextPathType][0]
				var err error
				relDockerfilePath, err = filepath.Rel(dockerContextPath, dockerfilePath)
				if err != nil {
					logrus.Errorf("failed to make the path %s relative to the base path %s . Error: %q", dockerfilePath, dockerContextPath, err)
					continue
				}
			}
			dockerfileImageBuildConfig.DockerfileName = relDockerfilePath
			if common.IsParent(dockerfilePath, t.Env.GetEnvironmentSource()) {
				relDockerContextPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), filepath.Dir(dockerfilePath))
				if err != nil {
					logrus.Errorf("failed to make the path %s relative to the base path %s . Error: %q", filepath.Dir(dockerfilePath), t.Env.GetEnvironmentSource(), err)
					continue
				}
				dockerfileImageBuildConfig.ContextUnix = common.GetUnixPath(filepath.Join(common.DefaultSourceDir, relDockerContextPath))
				dockerfileImageBuildConfig.ContextWindows = common.GetWindowsPath(filepath.Join(common.DefaultSourceDir, relDockerContextPath))
				dockerfilesImageBuildConfig = append(dockerfilesImageBuildConfig, dockerfileImageBuildConfig)
			} else if common.IsParent(dockerfilePath, t.Env.GetEnvironmentOutput()) {
				relDockerContextPath, err := filepath.Rel(t.Env.GetEnvironmentOutput(), filepath.Dir(dockerfilePath))
				if err != nil {
					logrus.Errorf("failed to make the path %s relative to the base path %s . Error: %q", filepath.Dir(dockerfilePath), t.Env.GetEnvironmentOutput(), err)
					continue
				}
				dockerfileImageBuildConfig.ContextUnix = common.GetUnixPath(relDockerContextPath)
				dockerfileImageBuildConfig.ContextWindows = common.GetWindowsPath(relDockerContextPath)
				dockerfilesImageBuildConfig = append(dockerfilesImageBuildConfig, dockerfileImageBuildConfig)
			} else {
				dockerfileImageBuildConfig.ContextUnix = common.GetUnixPath(filepath.Join(common.DefaultSourceDir, dockerContextPath))
				dockerfileImageBuildConfig.ContextWindows = common.GetWindowsPath(filepath.Join(common.DefaultSourceDir, dockerContextPath))
				dockerfilesImageBuildConfig = append(dockerfilesImageBuildConfig, dockerfileImageBuildConfig)
			}
			createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
				Name: t.Env.ProjectName,
				Type: artifacts.NewImagesArtifactType,
				Configs: map[transformertypes.ConfigType]interface{}{
					artifacts.NewImagesConfigType: artifacts.NewImages{
						ImageNames: []string{imageName.ImageName},
					},
				},
			})
		}
	}
	if len(dockerfilesImageBuildConfig) == 0 {
		return nil, nil, nil
	}
	relSourceDir, err := filepath.Rel(t.DockerfileImageBuildScriptConfig.OutputPath, common.DefaultSourceDir)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to make the sources directory %s relative to the scripts directory %s . Error: %q",
			common.DefaultSourceDir,
			t.DockerfileImageBuildScriptConfig.OutputPath,
			err,
		)
	}
	containerImageBuildShScriptPaths := []string{}
	containerImageBuildBatScriptPaths := []string{}
	templateData := DockerfileImageBuildScriptTemplateConfig{
		RelParentOfSourceDir: filepath.Join(relSourceDir, ".."),
		RegistryURL:          commonqa.ImageRegistry(),
		RegistryNamespace:    commonqa.ImageRegistryNamespace(),
		DockerfilesConfig:    dockerfilesImageBuildConfig,
	}
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
		DestPath:       t.DockerfileImageBuildScriptConfig.OutputPath,
		TemplateConfig: templateData,
	})
	containerImageBuildShScriptPaths = append(
		containerImageBuildShScriptPaths,
		filepath.Join(
			t.DockerfileImageBuildScriptConfig.OutputPath,
			buildImagesFileName+common.ShExt,
		),
		filepath.Join(
			t.DockerfileImageBuildScriptConfig.OutputPath,
			buildAndPushImagesFileName+common.ShExt,
		),
	)
	containerImageBuildBatScriptPaths = append(
		containerImageBuildBatScriptPaths,
		filepath.Join(
			t.DockerfileImageBuildScriptConfig.OutputPath,
			buildImagesFileName+common.BatExt,
		),
		filepath.Join(
			t.DockerfileImageBuildScriptConfig.OutputPath,
			buildAndPushImagesFileName+common.BatExt,
		),
	)
	createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
		Name: string(artifacts.ContainerImageBuildScriptArtifactType),
		Type: artifacts.ContainerImageBuildScriptArtifactType,
		Paths: map[transformertypes.PathType][]string{
			artifacts.ContainerImageBuildShScriptPathType:         containerImageBuildShScriptPaths,
			artifacts.ContainerImageBuildShScriptContextPathType:  {"."},
			artifacts.ContainerImageBuildBatScriptPathType:        containerImageBuildBatScriptPaths,
			artifacts.ContainerImageBuildBatScriptContextPathType: {"."},
		},
	})
	return pathMappings, createdArtifacts, nil
}
