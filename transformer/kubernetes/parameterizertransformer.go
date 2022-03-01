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

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/transformer/kubernetes/k8sschema"
	"github.com/konveyor/move2kube/transformer/kubernetes/parameterizer"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	helmPathTemplateName       = "HelmPath"
	kustomizePathTemplateName  = "KustomizePath"
	ocTemplatePathTemplateName = "OCTemplatePath"
)

// Parameterizer implements Transformer interface
type Parameterizer struct {
	Config              transformertypes.Transformer
	Env                 *environment.Environment
	ParameterizerConfig *ParameterizerYamlConfig
	parameterizers      []parameterizer.ParameterizerT
}

// ParameterizerYamlConfig implements Parameterizer path config interface
type ParameterizerYamlConfig struct {
	HelmPath       string `yaml:"helmPath"`
	OCTemplatePath string `yaml:"ocTemplatePath"`
	KustomizePath  string `yaml:"kustomizePath"`
	ProjectName    string `yaml:"projectName"`
}

// ParameterizerPathTemplateConfig stores the template config
type ParameterizerPathTemplateConfig struct {
	YamlsPath        string
	ServiceFsPath    string
	PathTemplateName string
}

// Init Initializes the transformer
func (t *Parameterizer) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	t.ParameterizerConfig = &ParameterizerYamlConfig{}
	err := common.GetObjFromInterface(t.Config.Spec.Config, t.ParameterizerConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.ParameterizerConfig, err)
		return err
	}
	if t.ParameterizerConfig.ProjectName == "" {
		t.ParameterizerConfig.ProjectName = e.ProjectName
	}
	psmap, err := parameterizer.CollectParamsFromPath(t.Env.Context)
	if err != nil {
		logrus.Errorf("Error while parsing for params : %s", err)
		return err
	}
	for _, p := range psmap {
		t.parameterizers = append(t.parameterizers, p...)
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Parameterizer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each subdirectory
func (t *Parameterizer) DirectoryDetect(dir string) (namedServices map[string][]transformertypes.Artifact, err error) {
	if len(k8sschema.GetKubernetesObjsInDir(dir)) != 0 {
		na := transformertypes.Artifact{
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
func (t *Parameterizer) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		yamlsPath := a.Paths[artifacts.KubernetesYamlsPathType][0]
		tempPath, err := os.MkdirTemp(t.Env.TempPath, "*")
		if err != nil {
			logrus.Errorf("Unable to create temp dir : %s", err)
		}
		baseDirName := filepath.Base(yamlsPath) + "-parameterized"
		destPath := filepath.Join(tempPath, baseDirName)
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Debugf("Unable to load config for Transformer into %T : %s", sConfig, err)
		}
		projectName, err := common.GetStringFromTemplate(t.ParameterizerConfig.ProjectName,
			map[string]string{common.ProjectNameTemplatizedStringKey: t.Env.ProjectName,
				common.ArtifactNameTemplatizedStringKey: a.Name,
				common.ServiceNameTemplatizedStringKey:  sConfig.ServiceName,
				common.ArtifactTypeTemplatizedStringKey: string(a.Type)})
		if err != nil {
			logrus.Errorf("Unable to evaluate helm chart name : %s", err)
			continue
		}
		pt := parameterizer.ParameterizerConfigT{
			Helm:        "helm",
			Kustomize:   "kustomize",
			OCTemplates: "octemplates",
			ProjectName: projectName,
		}
		if len(t.ParameterizerConfig.HelmPath) == 0 {
			pt.Helm = ""
		}
		if len(t.ParameterizerConfig.KustomizePath) == 0 {
			pt.Kustomize = ""
		}
		if len(t.ParameterizerConfig.OCTemplatePath) == 0 {
			pt.OCTemplates = ""
		}
		filesWritten, err := parameterizer.Parameterize(yamlsPath, destPath, pt, t.parameterizers)
		if err != nil {
			logrus.Errorf("failed to parameterize the YAML files in the source directory %s and write to output directory %s . Error: %q", yamlsPath, destPath, err)
			continue
		}
		logrus.Debugf("Number of files written by parameterizer: %d", len(filesWritten))

		helmKey := helmPathTemplateName + common.GetRandomString()
		kustomizeKey := kustomizePathTemplateName + common.GetRandomString()
		octKey := ocTemplatePathTemplateName + common.GetRandomString()

		serviceFsPath := ""
		if serviceFsPaths, ok := a.Paths[artifacts.ServiceDirPathType]; ok && len(serviceFsPaths) > 0 {
			serviceFsPath = serviceFsPaths[0]
		}
		if len(t.ParameterizerConfig.HelmPath) != 0 {
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.PathTemplatePathMappingType,
				SrcPath:        t.ParameterizerConfig.HelmPath,
				TemplateConfig: ParameterizerPathTemplateConfig{YamlsPath: yamlsPath, PathTemplateName: helmKey, ServiceFsPath: serviceFsPath},
			})
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  filepath.Join(destPath, pt.Helm),
				DestPath: fmt.Sprintf("{{ .%s }}", helmKey),
			})
		}
		if len(t.ParameterizerConfig.KustomizePath) != 0 {
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.PathTemplatePathMappingType,
				SrcPath:        t.ParameterizerConfig.KustomizePath,
				TemplateConfig: ParameterizerPathTemplateConfig{YamlsPath: yamlsPath, PathTemplateName: kustomizeKey, ServiceFsPath: serviceFsPath},
			})
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  filepath.Join(destPath, pt.Kustomize),
				DestPath: fmt.Sprintf("{{ .%s }}", kustomizeKey),
			})
		}
		if len(t.ParameterizerConfig.OCTemplatePath) != 0 {
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.PathTemplatePathMappingType,
				SrcPath:        t.ParameterizerConfig.OCTemplatePath,
				TemplateConfig: ParameterizerPathTemplateConfig{YamlsPath: yamlsPath, PathTemplateName: octKey, ServiceFsPath: serviceFsPath},
			})
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  filepath.Join(destPath, pt.OCTemplates),
				DestPath: fmt.Sprintf("{{ .%s }}", octKey),
			})
		}
	}
	return pathMappings, nil, nil
}
