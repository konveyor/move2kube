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

// DockerfileImageBuildScript implements Transformer interface
type DockerfileImageBuildScript struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// DockerfileImageBuildScriptTemplateConfig represents template config used by ImageBuild script
type DockerfileImageBuildScriptTemplateConfig struct {
	DockerfilesConfig      []DockerfileImageBuildConfig
	RegistryURL            string
	RegistryNamespace      string
	TargetPlatforms        string
	DockerContainerRuntime bool
	PodmanContainerRuntime bool
	BuildxContainerRuntime bool
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
	selectedContainerRuntimes := commonqa.GetContainerRuntimes()
	containerImageBuildShScriptPaths := []string{}
	containerImageBuildBatScriptPaths := []string{}
	dockerfileImageBuildScriptConfig := DockerfileImageBuildScriptTemplateConfig{
		RegistryURL:       commonqa.ImageRegistry(),
		RegistryNamespace: commonqa.ImageRegistryNamespace(),
		DockerfilesConfig: dockerfilesImageBuildConfig,
	}
	for _, containerRuntime := range selectedContainerRuntimes {
		switch containerRuntime {
		case "docker":
			dockerfileImageBuildScriptConfig.DockerContainerRuntime = true
			containerImageBuildShScriptPaths = append(containerImageBuildShScriptPaths, filepath.Join(common.ScriptsDir, "buildandpushdockerimages_"+containerRuntime+common.ShExt))
			containerImageBuildBatScriptPaths = append(containerImageBuildBatScriptPaths, filepath.Join(common.ScriptsDir, "buildandpushdockerimages_"+containerRuntime+common.BatExt))
		case "buildx":
			dockerfileImageBuildScriptConfig.BuildxContainerRuntime = true
			containerImageBuildShScriptPaths = append(containerImageBuildShScriptPaths, filepath.Join(common.ScriptsDir, "builddockerimages_"+containerRuntime+common.ShExt))
			containerImageBuildBatScriptPaths = append(containerImageBuildBatScriptPaths, filepath.Join(common.ScriptsDir, "builddockerimages_"+containerRuntime+common.BatExt))
			dockerfileImageBuildScriptConfig.TargetPlatforms = strings.Join(commonqa.GetTargetPlatforms(), ",")
		case "podman":
			dockerfileImageBuildScriptConfig.PodmanContainerRuntime = true
			containerImageBuildShScriptPaths = append(containerImageBuildShScriptPaths, filepath.Join(common.ScriptsDir, "builddockerimages_"+containerRuntime+common.ShExt))
			containerImageBuildBatScriptPaths = append(containerImageBuildBatScriptPaths, filepath.Join(common.ScriptsDir, "builddockerimages_"+containerRuntime+common.BatExt))
		default:
			logrus.Errorf("unsupported container runtime %s", containerRuntime)
			continue
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir, containerRuntime),
			DestPath:       common.ScriptsDir,
			TemplateConfig: dockerfileImageBuildScriptConfig,
		})
	}
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir, "default"),
		DestPath:       common.ScriptsDir,
		TemplateConfig: dockerfileImageBuildScriptConfig,
	})
	containerImageBuildShScriptPaths = append(containerImageBuildShScriptPaths, filepath.Join(common.ScriptsDir, "builddockerimages"+common.ShExt))
	containerImageBuildBatScriptPaths = append(containerImageBuildBatScriptPaths, filepath.Join(common.ScriptsDir, "builddockerimages"+common.BatExt))
	createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
		Name: string(artifacts.ContainerImageBuildScriptArtifactType),
		Type: artifacts.ContainerImageBuildScriptArtifactType,
		Paths: map[transformertypes.PathType][]string{artifacts.ContainerImageBuildShScriptPathType: containerImageBuildShScriptPaths,
			artifacts.ContainerImageBuildShScriptContextPathType:  {"."},
			artifacts.ContainerImageBuildBatScriptPathType:        containerImageBuildBatScriptPaths,
			artifacts.ContainerImageBuildBatScriptContextPathType: {"."},
		},
	})
	return pathMappings, createdArtifacts, nil
}
