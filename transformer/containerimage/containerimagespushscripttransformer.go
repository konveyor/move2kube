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
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	pushImagesFileName = "pushimages"
)

// ContainerImagesPushScript implements Transformer interface
type ContainerImagesPushScript struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// ImagePushTemplateConfig represents template config used by ImagePush script
type ImagePushTemplateConfig struct {
	RegistryURL            string
	RegistryNamespace      string
	Images                 []string
	DockerContainerRuntime bool
	PodmanContainerRuntime bool
}

// Init Initializes the transformer
func (t *ContainerImagesPushScript) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *ContainerImagesPushScript) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *ContainerImagesPushScript) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *ContainerImagesPushScript) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	ipt := ImagePushTemplateConfig{}
	for _, a := range newArtifacts {
		if a.Type != artifacts.NewImagesArtifactType {
			continue
		}
		images := artifacts.NewImages{}
		err := a.GetConfig(artifacts.NewImagesConfigType, &images)
		if err != nil {
			logrus.Errorf("Unable to read Image config : %s", err)
		}
		ipt.Images = common.MergeSlices(ipt.Images, images.ImageNames)
	}
	if len(ipt.Images) == 0 {
		return nil, nil, nil
	}
	ipt.RegistryURL = commonqa.ImageRegistry()
	ipt.RegistryNamespace = commonqa.ImageRegistryNamespace()
	selectedContainerRuntimes := commonqa.GetContainerRuntimes()
	if len(selectedContainerRuntimes) == 1 && selectedContainerRuntimes[0] == "buildx" {
		return nil, nil, nil
	}
	containerImagePushShScriptPaths := []string{}
	containerImagePushBatScriptPaths := []string{}
	for _, containerRuntime := range selectedContainerRuntimes {
		switch containerRuntime {
		case "docker":
			ipt.DockerContainerRuntime = true
		case "podman":
			ipt.PodmanContainerRuntime = true
		case "buildx":
			continue
		default:
			logrus.Errorf("unsupported container runtime %s", containerRuntime)
			continue
		}
		containerImagePushShScriptPaths = append(containerImagePushShScriptPaths, filepath.Join(common.ScriptsDir, pushImagesFileName+"_"+containerRuntime+common.ShExt))
		containerImagePushBatScriptPaths = append(containerImagePushBatScriptPaths, filepath.Join(common.ScriptsDir, pushImagesFileName+"_"+containerRuntime+common.ShExt))
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir, containerRuntime),
			DestPath:       common.ScriptsDir,
			TemplateConfig: ipt,
		})
	}
	containerImagePushShScriptPaths = append(containerImagePushShScriptPaths, filepath.Join(common.ScriptsDir, pushImagesFileName+common.ShExt))
	containerImagePushBatScriptPaths = append(containerImagePushBatScriptPaths, filepath.Join(common.ScriptsDir, pushImagesFileName+common.ShExt))
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir, "default"),
		DestPath:       common.ScriptsDir,
		TemplateConfig: ipt,
	})
	artifacts := []transformertypes.Artifact{{
		Name: string(artifacts.ContainerImagesPushScriptArtifactType),
		Type: artifacts.ContainerImagesPushScriptArtifactType,
		Paths: map[transformertypes.PathType][]string{artifacts.ContainerImagesPushShScriptPathType: containerImagePushShScriptPaths,
			artifacts.ContainerImagesPushBatScriptPathType: containerImagePushBatScriptPaths},
	}}
	return pathMappings, artifacts, nil
}
