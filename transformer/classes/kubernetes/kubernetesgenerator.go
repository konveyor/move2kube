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
	"path/filepath"

	"github.com/konveyor/move2kube/apiresource"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/irpreprocessor"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	outputPathTemplateName = "OutputPath"
)

// Kubernetes implements Transformer interface
type Kubernetes struct {
	Config           transformertypes.Transformer
	Env              *environment.Environment
	KubernetesConfig *KubernetesGenYamlConfig
}

// KubernetesGenYamlConfig stores the k8s related information
type KubernetesGenYamlConfig struct {
	OutputPath string `yaml:"outputPath"`
}

// KubernetesPathTemplateConfig implements Kubernetes template config interface
type KubernetesPathTemplateConfig struct {
	PathTemplateName string
	PrjPath          string
}

// Init Initializes the transformer
func (t *Kubernetes) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	t.KubernetesConfig = &KubernetesGenYamlConfig{}
	err := common.GetObjFromInterface(t.Config.Spec.Config, &t.KubernetesConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.KubernetesConfig, err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Kubernetes) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each subdirectory
func (t *Kubernetes) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms artifacts
func (t *Kubernetes) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Debugf("Translating IR using Kubernetes transformer")
	pathMappings = []transformertypes.PathMapping{}
	createdArtifacts = []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != irtypes.IRArtifactType {
			continue
		}
		var ir irtypes.IR
		if err := a.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", ir, err)
			continue
		}
		ir.Name = a.Name
		preprocessedIR, err := irpreprocessor.Preprocess(ir)
		if err != nil {
			logrus.Errorf("Unable to pre-preocess IR : %s", err)
		} else {
			ir = preprocessedIR
		}
		tempDest := filepath.Join(t.Env.TempPath, "k8s-yamls")
		logrus.Debugf("Starting Kubernetes transform")
		logrus.Debugf("Total services to be transformed : %d", len(ir.Services))
		apis := []apiresource.IAPIResource{new(apiresource.Deployment), new(apiresource.Storage), new(apiresource.Service), new(apiresource.ImageStream), new(apiresource.NetworkPolicy)}
		files, err := apiresource.TransformAndPersist(irtypes.NewEnhancedIRFromIR(ir), tempDest, apis, t.Env.TargetCluster)
		if err != nil {
			logrus.Errorf("Unable to transform and persist IR : %s", err)
			return nil, nil, err
		}
		prjPath := ""
		if prjPaths, ok := a.Paths[artifacts.ProjectPathPathType]; ok && len(prjPaths) > 0 {
			prjPath = prjPaths[0]
		}
		outputPathKey := outputPathTemplateName + common.GetRandomString(randUpLimit)
		outputPath := fmt.Sprintf("{{ .%s }}", outputPathKey)
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.PathTemplatePathMappingType,
			SrcPath:        t.KubernetesConfig.OutputPath,
			TemplateConfig: KubernetesPathTemplateConfig{PathTemplateName: outputPathKey, PrjPath: prjPath},
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  tempDest,
			DestPath: outputPath,
		})
		na := transformertypes.Artifact{
			Name:     t.Config.Name,
			Artifact: artifacts.KubernetesYamlsArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {outputPath},
			},
		}
		// Append the project path only if there is one-one mapping between services and artifacts
		if len(ir.Services) == 1 {
			if prjPaths, ok := a.Paths[artifacts.ProjectPathPathType]; ok && len(prjPaths) > 0 {
				na.Paths[artifacts.ProjectPathPathType] = append(na.Paths[artifacts.ProjectPathPathType], prjPaths[0])
				// Loop to get the single service name
				for k := range ir.Services {
					na.Name = k
				}
			}
		}
		createdArtifacts = append(createdArtifacts, na)
		logrus.Infof("Total transformed objects : %d", len(files))
	}
	return pathMappings, createdArtifacts, nil
}
