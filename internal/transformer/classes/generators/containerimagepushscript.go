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
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// ContainerImagePushScript implements Transformer interface
type ContainerImagePushScript struct {
	TConfig transformertypes.Transformer
	Env     *environment.Environment
}

// ImagePushTemplateConfig represents template config used by ImagePush script
type ImagePushTemplateConfig struct {
	RegistryURL       string
	RegistryNamespace string
	Images            []string
}

// Init Initializes the transformer
func (t *ContainerImagePushScript) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *ContainerImagePushScript) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.TConfig, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *ContainerImagePushScript) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *ContainerImagePushScript) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *ContainerImagePushScript) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	ipt := ImagePushTemplateConfig{}
	for _, a := range newArtifacts {
		if a.Artifact == artifacts.NewImageArtifactType {
			image := artifacts.NewImage{}
			err := a.GetConfig(artifacts.NewImageConfigType, &image)
			if err != nil {
				logrus.Errorf("Unable to read Image config : %s", err)
			}
			if !common.IsStringPresent(ipt.Images, image.ImageName) {
				ipt.Images = append(ipt.Images, image.ImageName)
			}
		}
	}
	if len(ipt.Images) == 0 {
		return nil, nil, nil
	}
	ipt.RegistryURL = commonqa.ImageRegistry()
	ipt.RegistryNamespace = commonqa.ImageRegistryNamespace(t.Env.ProjectName)
	pathMappings = append(pathMappings, transformertypes.PathMapping{
		Type:           transformertypes.TemplatePathMappingType,
		SrcPath:        filepath.Join(t.Env.Context, t.TConfig.Spec.TemplatesDir),
		DestPath:       common.ScriptsDir,
		TemplateConfig: ipt,
	})
	artifacts := []transformertypes.Artifact{{
		Name:     artifacts.ImagePushScriptArtifactType,
		Artifact: artifacts.ImagePushScriptArtifactType,
		Paths: map[string][]string{artifacts.ImagePushShScriptPathType: {filepath.Join(common.ScriptsDir, "pushimages.sh")},
			artifacts.ImagePushBatScriptPathType: {filepath.Join(common.ScriptsDir, "pushimages.bat")}},
	}}
	return pathMappings, artifacts, nil
}
