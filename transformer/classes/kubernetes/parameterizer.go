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
	"io/ioutil"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/parameterizer"
	parameterizertypes "github.com/konveyor/move2kube/types/parameterizer"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	helmPathTemplateName       = "HelmPath"
	kustomizePathTemplateName  = "KustomizePath"
	ocTemplatePathTemplateName = "OCTemplatePath"
	randUpLimit                = 10000000
)

// Parameterizer implements Transformer interface
type Parameterizer struct {
	Config     transformertypes.Transformer
	Env        *environment.Environment
	PathConfig ParameterizerPathConfig
}

// ParameterizerPathConfig implements Parameterizer path config interface
type ParameterizerPathConfig struct {
	HelmPath       string `yaml:"helmPath"`
	OCTemplatePath string `yaml:"ocTemplatePath"`
	KustomizePath  string `yaml:"kustomizePath"`
}

// ParameterizerPathTemplateConfig implements Parameterizer template config interface
type ParameterizerPathTemplateConfig struct {
	YamlsPath        string
	PrjPath          string
	PathTemplateName string
}

// Init Initializes the transformer
func (t *Parameterizer) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	t.PathConfig = ParameterizerPathConfig{}
	err := common.GetObjFromInterface(t.Config.Spec.Config, &t.PathConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.PathConfig, err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Parameterizer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each subdirectory
func (t *Parameterizer) DirectoryDetect(dir string) (namedServices map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms artifacts
func (t *Parameterizer) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	psmap, err := parameterizer.CollectParamsFromPath(t.Env.Context)
	if err != nil {
		logrus.Errorf("Error while parsing for params : %s", err)
		return nil, nil, err
	}
	ps := []parameterizertypes.ParameterizerT{}
	for _, p := range psmap {
		ps = append(ps, p...)
	}
	for _, a := range newArtifacts {
		yamlsPath := a.Paths[artifacts.KubernetesYamlsPathType][0]
		tempPath, err := ioutil.TempDir(t.Env.TempPath, "*")
		if err != nil {
			logrus.Errorf("Unable to create temp dir : %s", err)
		}
		baseDirName := filepath.Base(yamlsPath) + "-parameterized"
		destPath := filepath.Join(tempPath, baseDirName)

		pt := parameterizertypes.PackagingSpecPathT{Helm: "helm",
			Kustomize:   "kustomize",
			OCTemplates: "octemplates"}
		_, err = parameterizer.Parameterize(yamlsPath, destPath, pt, ps)
		if err != nil {
			logrus.Errorf("Unable to parameterize : %s", err)
		}

		helmKey := helmPathTemplateName + common.GetRandomString(randUpLimit)
		kustomizeKey := kustomizePathTemplateName + common.GetRandomString(randUpLimit)
		octKey := ocTemplatePathTemplateName + common.GetRandomString(randUpLimit)

		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.PathTemplatePathMappingType,
			SrcPath:        t.PathConfig.HelmPath,
			TemplateConfig: ParameterizerPathTemplateConfig{YamlsPath: yamlsPath, PathTemplateName: helmKey},
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.PathTemplatePathMappingType,
			SrcPath:        t.PathConfig.KustomizePath,
			TemplateConfig: ParameterizerPathTemplateConfig{YamlsPath: yamlsPath, PathTemplateName: kustomizeKey},
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.PathTemplatePathMappingType,
			SrcPath:        t.PathConfig.OCTemplatePath,
			TemplateConfig: ParameterizerPathTemplateConfig{YamlsPath: yamlsPath, PathTemplateName: octKey},
		})

		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  filepath.Join(destPath, pt.Helm),
			DestPath: fmt.Sprintf("{{ .%s }}", helmKey),
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  filepath.Join(destPath, pt.Kustomize),
			DestPath: fmt.Sprintf("{{ .%s }}", kustomizeKey),
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  filepath.Join(destPath, pt.OCTemplates),
			DestPath: fmt.Sprintf("{{ .%s }}", octKey),
		})
	}
	return pathMappings, nil, nil
}
