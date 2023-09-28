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
	"github.com/konveyor/move2kube/transformer/kubernetes/apiresource"
	"github.com/konveyor/move2kube/transformer/kubernetes/irpreprocessor"
	"github.com/konveyor/move2kube/transformer/kubernetes/parameterizer"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	outputPathTemplateName    = "OutputPath"
	defaultK8sYamlsOutputPath = common.DeployDir + string(os.PathSeparator) + "yamls"
	setDefaultValuesInYamls   = false
)

// Kubernetes implements Transformer interface
type Kubernetes struct {
	Config           transformertypes.Transformer
	Env              *environment.Environment
	KubernetesConfig *KubernetesYamlConfig
}

// KubernetesYamlConfig stores the k8s related information
type KubernetesYamlConfig struct {
	IngressName             string `yaml:"ingressName"`
	OutputPath              string `yaml:"outputPath"`
	SetDefaultValuesInYamls bool   `yaml:"setDefaultValuesInYamls"`
}

// KubernetesPathTemplateConfig implements Kubernetes template config interface
type KubernetesPathTemplateConfig struct {
	PathTemplateName string
	ServiceFsPath    string
}

// Init Initializes the transformer
func (t *Kubernetes) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	t.KubernetesConfig = &KubernetesYamlConfig{}
	err := common.GetObjFromInterface(t.Config.Spec.Config, t.KubernetesConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T . Error: %q", t.Config.Spec.Config, t.KubernetesConfig, err)
		return err
	}
	if t.KubernetesConfig.IngressName == "" {
		t.KubernetesConfig.IngressName = e.ProjectName
	}
	if t.KubernetesConfig.OutputPath == "" {
		t.KubernetesConfig.OutputPath = defaultK8sYamlsOutputPath
	}
	if !t.KubernetesConfig.SetDefaultValuesInYamls {
		t.KubernetesConfig.SetDefaultValuesInYamls = setDefaultValuesInYamls
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
func (t *Kubernetes) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Trace("Kubernetes.Transform start")
	defer logrus.Trace("Kubernetes.Transform end")
	pathMappings = []transformertypes.PathMapping{}
	createdArtifacts = []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if newArtifact.Type != irtypes.IRArtifactType {
			continue
		}
		var ir irtypes.IR
		if err := newArtifact.GetConfig(irtypes.IRConfigType, &ir); err != nil {
			logrus.Errorf("failed to load config for Transformer into %T . Error: %q", ir, err)
			continue
		}

		var clusterConfig collecttypes.ClusterMetadata
		if err := newArtifact.GetConfig(ClusterMetadata, &clusterConfig); err != nil {
			logrus.Errorf("failed to load config for Transformer into %T . Error: %q", clusterConfig, err)
			continue
		}
		templatizedKeyMap := map[string]string{common.ProjectNameTemplatizedStringKey: t.Env.ProjectName,
			common.ArtifactNameTemplatizedStringKey: newArtifact.Name,
		}
		if len(ir.Services) == 1 {
			for sn := range ir.Services {
				templatizedKeyMap[common.ServiceNameTemplatizedStringKey] = sn
			}
		}
		ir.Name, err = common.GetStringFromTemplate(t.KubernetesConfig.IngressName, templatizedKeyMap)
		if err != nil {
			logrus.Errorf("failed to compute the Ingress name. Error: %q", err)
			ir.Name = newArtifact.Name
		}
		if ir.Name == "" {
			logrus.Errorf("Evaluating IngressName in Kubernetes transformer resulting in empty string. Defaulting to Artifact Name.")
			ir.Name = newArtifact.Name
		}
		preprocessedIR, err := irpreprocessor.Preprocess(ir, clusterConfig)
		if err != nil {
			logrus.Errorf("failed to pre-preocess the IR. Error: %q", err)
		} else {
			ir = preprocessedIR
		}
		tempDest := filepath.Join(t.Env.TempPath, "k8s-yamls-"+common.GetRandomString())
		logrus.Debugf("Starting Kubernetes transform")
		logrus.Debugf("Total services to be transformed: %d", len(ir.Services))
		apis := []apiresource.IAPIResource{
			new(apiresource.Deployment),
			new(apiresource.Storage),
			new(apiresource.Service),
			new(apiresource.ImageStream),
			new(apiresource.NetworkPolicy),
		}
		files, err := apiresource.TransformIRAndPersist(irtypes.NewEnhancedIRFromIR(ir), tempDest, apis, clusterConfig, t.KubernetesConfig.SetDefaultValuesInYamls)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to transform and persist the IR. Error: %w", err)
		}
		serviceFsPath := ""
		if serviceFsPaths, ok := newArtifact.Paths[artifacts.ServiceDirPathType]; ok && len(serviceFsPaths) > 0 {
			serviceFsPath = serviceFsPaths[0]
		}
		outputPathKey := outputPathTemplateName + common.GetRandomString()
		outputPath := fmt.Sprintf("{{ .%s }}", outputPathKey)
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:           transformertypes.PathTemplatePathMappingType,
			SrcPath:        t.KubernetesConfig.OutputPath,
			TemplateConfig: KubernetesPathTemplateConfig{PathTemplateName: outputPathKey, ServiceFsPath: serviceFsPath},
		})
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  tempDest,
			DestPath: outputPath,
		})
		createdArtifact := transformertypes.Artifact{
			Name: t.Config.Name,
			Type: artifacts.KubernetesYamlsArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {outputPath},
			},
		}
		{
			moreParams := []parameterizer.ParameterizerT{}
			if err := newArtifact.GetConfig(ExtraParameterizersConfigType, &moreParams); err != nil {
				logrus.Debugf("failed to load config of type '%s' into struct of type %T . Error: %q", ExtraParameterizersConfigType, moreParams, err)
			} else {
				if createdArtifact.Configs == nil {
					createdArtifact.Configs = map[string]interface{}{}
				}
				createdArtifact.Configs[ExtraParameterizersConfigType] = moreParams
			}
		}
		// Append the project path only if there is one-one mapping between services and artifacts
		if len(ir.Services) == 1 {
			if serviceFsPaths, ok := newArtifact.Paths[artifacts.ServiceDirPathType]; ok && len(serviceFsPaths) > 0 {
				createdArtifact.Paths[artifacts.ServiceDirPathType] = append(createdArtifact.Paths[artifacts.ServiceDirPathType], serviceFsPaths[0])
				// Loop to get the single service name
				for k := range ir.Services {
					createdArtifact.Name = k
				}
			}
		}
		createdArtifacts = append(createdArtifacts, createdArtifact)
		logrus.Debugf("Total transformed objects : %d", len(files))
	}
	return pathMappings, createdArtifacts, nil
}
