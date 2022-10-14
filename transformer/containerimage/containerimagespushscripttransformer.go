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
	pushImagesFileName                 = "pushimages"
	defaultDockerPushScriptsOutputPath = common.ScriptsDir
)

// ContainerImagesPushScript implements Transformer interface
type ContainerImagesPushScript struct {
	Config                          transformertypes.Transformer
	Env                             *environment.Environment
	DockerfileImagePushScriptConfig *DockerfileImagePushScriptConfig
}

// DockerfileImagePushScriptConfig stores the transformer specific configuration
type DockerfileImagePushScriptConfig struct {
	OutputPath string `yaml:"outputPath"`
}

// ImagePushTemplateConfig represents template config used by ImagePush script
type ImagePushTemplateConfig struct {
	RegistryURL       string
	RegistryNamespace string
	Images            []string
}

// Init Initializes the transformer
func (t *ContainerImagesPushScript) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.DockerfileImagePushScriptConfig = &DockerfileImagePushScriptConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.DockerfileImagePushScriptConfig); err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.DockerfileImagePushScriptConfig, err)
		return err
	}
	if t.DockerfileImagePushScriptConfig.OutputPath == "" {
		t.DockerfileImagePushScriptConfig.OutputPath = defaultDockerPushScriptsOutputPath
	}
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
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
		DestPath:       t.DockerfileImagePushScriptConfig.OutputPath,
		TemplateConfig: ipt,
	})
	artifacts := []transformertypes.Artifact{{
		Name: string(artifacts.ContainerImagesPushScriptArtifactType),
		Type: artifacts.ContainerImagesPushScriptArtifactType,
		Paths: map[transformertypes.PathType][]string{
			artifacts.ContainerImagesPushShScriptPathType:  {filepath.Join(t.DockerfileImagePushScriptConfig.OutputPath, pushImagesFileName+common.ShExt)},
			artifacts.ContainerImagesPushBatScriptPathType: {filepath.Join(t.DockerfileImagePushScriptConfig.OutputPath, pushImagesFileName+common.BatExt)}},
	}}
	return pathMappings, artifacts, nil
}
