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

package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	"github.com/konveyor/move2kube-wasm/transformer/kubernetes/apiresource"
	"github.com/konveyor/move2kube-wasm/transformer/kubernetes/k8sschema"
	collecttypes "github.com/konveyor/move2kube-wasm/types/collection"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/konveyor/move2kube-wasm/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	defaultKVCOutputPath = "{{ $rel := Rel .YamlsPath }}source/{{ $rel }}{{ if ne $rel \".\" }}/..{{end}}/{{ FilePathBase .YamlsPath }}-versionchanged/"
)

// KubernetesVersionChanger implements Transformer interface
type KubernetesVersionChanger struct {
	Config    transformertypes.Transformer
	Env       *environment.Environment
	KVCConfig *KubernetesVersionChangerYamlConfig
}

// KubernetesVersionChangerYamlConfig stores the config
type KubernetesVersionChangerYamlConfig struct {
	OutputPath              string `yaml:"outputPath"`
	SetDefaultValuesInYamls bool   `yaml:"setDefaultValuesInYamls"`
}

// OutputPathParams stores the possible path params
type OutputPathParams struct {
	PathTemplateName string
	YamlsPath        string
}

// Init Initializes the transformer
func (t *KubernetesVersionChanger) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	t.KVCConfig = &KubernetesVersionChangerYamlConfig{}
	err := common.GetObjFromInterface(t.Config.Spec.Config, t.KVCConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.KVCConfig, err)
		return err
	}
	if t.KVCConfig.OutputPath == "" {
		t.KVCConfig.OutputPath = defaultKVCOutputPath
	}
	if !t.KVCConfig.SetDefaultValuesInYamls {
		t.KVCConfig.SetDefaultValuesInYamls = setDefaultValuesInYamls
	}
	return nil
}

// GetConfig returns the transformer config
func (t *KubernetesVersionChanger) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each subdirectory
func (t *KubernetesVersionChanger) DirectoryDetect(dir string) (namedServices map[string][]transformertypes.Artifact, err error) {
	if len(k8sschema.GetKubernetesObjsInDir(dir)) != 0 {
		na := transformertypes.Artifact{
			Type: artifacts.KubernetesOrgYamlsInSourceArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {dir},
				artifacts.ServiceDirPathType:      {dir},
			},
		}
		return map[string][]transformertypes.Artifact{"": {na}}, nil
	}
	return nil, nil
}

// Transform transforms artifacts
func (t *KubernetesVersionChanger) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	apis := []apiresource.IAPIResource{new(apiresource.Deployment), new(apiresource.Service)}
	for _, a := range newArtifacts {
		yamlsPath := a.Paths[artifacts.KubernetesYamlsPathType][0]
		var clusterConfig collecttypes.ClusterMetadata
		if err := a.GetConfig(ClusterMetadata, &clusterConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", clusterConfig, err)
			continue
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("Unable to load config for Transformer into %T : %s", sConfig, err)
		}
		tempDest := filepath.Join(t.Env.TempPath, "k8s-yamls-versionchanged-"+common.GetRandomString())
		err := filepath.WalkDir(yamlsPath, func(path string, info os.DirEntry, err error) error {
			if err != nil && path == yamlsPath {
				// if walk for root search path return gets error
				// then stop walking and return this error
				return err
			}
			if err != nil {
				logrus.Warnf("Skipping path %q due to error: %q", path, err)
				return nil
			}
			if info.IsDir() {
				relInputPath, err := filepath.Rel(yamlsPath, path)
				if err != nil {
					logrus.Errorf("Unable to convert %s as rel path of %s for yamls conversion : %s", path, yamlsPath, err)
					return nil
				}
				if objs := k8sschema.GetKubernetesObjsInDir(path); len(objs) != 0 {
					_, err := apiresource.TransformObjsAndPersist(path, filepath.Join(tempDest, relInputPath), apis, clusterConfig, t.KVCConfig.SetDefaultValuesInYamls)
					if err != nil {
						logrus.Errorf("Unable to transform objs at %s : %s", path, err)
						return nil
					}
				}
			}
			return nil
		})
		if err != nil {
			logrus.Warnf("Error in walking through files due to : %q", err)
		}
		outputPathKey := outputPathTemplateName + common.GetRandomString()
		outputPath := fmt.Sprintf("{{ .%s }}", outputPathKey)
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.PathTemplatePathMappingType,
			SrcPath:        t.KVCConfig.OutputPath,
			TemplateConfig: OutputPathParams{PathTemplateName: outputPathKey, YamlsPath: yamlsPath},
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  tempDest,
			DestPath: outputPath,
		})
		na := transformertypes.Artifact{
			Name: sConfig.ServiceName,
			Type: artifacts.KubernetesYamlsInSourceArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {outputPath},
			},
		}
		createdArtifacts = append(createdArtifacts, na)
	}
	return pathMappings, createdArtifacts, nil
}
