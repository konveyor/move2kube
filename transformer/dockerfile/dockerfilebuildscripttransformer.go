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

// DockerfileImageBuildScriptTemplateConfig represents template config used by ImagePush script
type DockerfileImageBuildScriptTemplateConfig struct {
	DockerfileName   string
	ImageName        string
	ContextUnix      string
	ContextWindows   string
	ContainerRuntime string
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
	dockerfiles := []DockerfileImageBuildScriptTemplateConfig{}
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
			if common.IsParent(dockerfilePath, t.Env.GetEnvironmentSource()) {
				relDockerContextPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), filepath.Dir(dockerfilePath))
				if err != nil {
					logrus.Errorf("failed to make the path %s relative to the base path %s . Error: %q", filepath.Dir(dockerfilePath), t.Env.GetEnvironmentSource(), err)
					continue
				}
				t1 := DockerfileImageBuildScriptTemplateConfig{
					ImageName:        imageName.ImageName,
					ContextUnix:      common.GetUnixPath(filepath.Join(common.DefaultSourceDir, relDockerContextPath)),
					ContextWindows:   common.GetWindowsPath(filepath.Join(common.DefaultSourceDir, relDockerContextPath)),
					DockerfileName:   relDockerfilePath,
					ContainerRuntime: commonqa.GetContainerRuntime(),
				}
				dockerfiles = append(dockerfiles, t1)
			} else if common.IsParent(dockerfilePath, t.Env.GetEnvironmentOutput()) {
				relDockerContextPath, err := filepath.Rel(t.Env.GetEnvironmentOutput(), filepath.Dir(dockerfilePath))
				if err != nil {
					logrus.Errorf("failed to make the path %s relative to the base path %s . Error: %q", filepath.Dir(dockerfilePath), t.Env.GetEnvironmentOutput(), err)
					continue
				}
				t2 := DockerfileImageBuildScriptTemplateConfig{
					ImageName:        imageName.ImageName,
					ContextUnix:      common.GetUnixPath(relDockerContextPath),
					ContextWindows:   common.GetWindowsPath(relDockerContextPath),
					DockerfileName:   relDockerfilePath,
					ContainerRuntime: commonqa.GetContainerRuntime(),
				}
				dockerfiles = append(dockerfiles, t2)
			} else {
				t3 := DockerfileImageBuildScriptTemplateConfig{
					ImageName:        imageName.ImageName,
					ContextUnix:      common.GetUnixPath(filepath.Join(common.DefaultSourceDir, dockerContextPath)),
					ContextWindows:   common.GetWindowsPath(filepath.Join(common.DefaultSourceDir, dockerContextPath)),
					DockerfileName:   relDockerfilePath,
					ContainerRuntime: commonqa.GetContainerRuntime(),
				}
				dockerfiles = append(dockerfiles, t3)
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
	if len(dockerfiles) == 0 {
		return nil, nil, nil
	}
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
		DestPath:       common.ScriptsDir,
		TemplateConfig: dockerfiles,
	})
	createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
		Name: string(artifacts.ContainerImageBuildScriptArtifactType),
		Type: artifacts.ContainerImageBuildScriptArtifactType,
		Paths: map[transformertypes.PathType][]string{artifacts.ContainerImageBuildShScriptPathType: {filepath.Join(common.ScriptsDir, "builddockerimages.sh")},
			artifacts.ContainerImageBuildShScriptContextPathType:  {"."},
			artifacts.ContainerImageBuildBatScriptPathType:        {filepath.Join(common.ScriptsDir, "builddockerimages.bat")},
			artifacts.ContainerImageBuildBatScriptContextPathType: {"."},
		},
	})
	return pathMappings, createdArtifacts, nil
}
