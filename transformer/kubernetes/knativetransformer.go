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
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/transformer/kubernetes/apiresource"
	"github.com/konveyor/move2kube/transformer/kubernetes/irpreprocessor"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	defaultKnativeYamlsOutputPath = common.DeployDir + string(os.PathSeparator) + "knative"
)

// Knative implements Transformer interface
type Knative struct {
	Config        transformertypes.Transformer
	Env           *environment.Environment
	KnativeConfig *KnativeYamlConfig
}

// KnativeYamlConfig stores the knative related information
type KnativeYamlConfig struct {
	OutputPath              string `yaml:"outputPath"`
	SetDefaultValuesInYamls bool   `yaml:"setDefaultValuesInYamls"`
}

// Init Initializes the transformer
func (t *Knative) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env
	t.KnativeConfig = &KnativeYamlConfig{}
	err := common.GetObjFromInterface(t.Config.Spec.Config, t.KnativeConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.KnativeConfig, err)
		return err
	}
	if t.KnativeConfig.OutputPath == "" {
		t.KnativeConfig.OutputPath = defaultKnativeYamlsOutputPath
	}
	if !t.KnativeConfig.SetDefaultValuesInYamls {
		t.KnativeConfig.SetDefaultValuesInYamls = setDefaultValuesInYamls
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Knative) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *Knative) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *Knative) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	logrus.Debugf("Translating IR using Kubernetes transformer")
	pathMappings = []transformertypes.PathMapping{}
	createdArtifacts = []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Type != irtypes.IRArtifactType {
			continue
		}
		var ir irtypes.IR
		err := a.GetConfig(irtypes.IRConfigType, &ir)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", ir, err)
			continue
		}
		var clusterConfig collecttypes.ClusterMetadata
		if err := a.GetConfig(ClusterMetadata, &clusterConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", clusterConfig, err)
			continue
		}
		ir.Name = a.Name
		preprocessedIR, err := irpreprocessor.Preprocess(ir)
		if err != nil {
			logrus.Errorf("Unable to prepreocess IR : %s", err)
		} else {
			ir = preprocessedIR
		}
		deployKnativeDir := t.KnativeConfig.OutputPath
		tempDest := filepath.Join(t.Env.TempPath, deployKnativeDir)
		logrus.Debugf("Starting Kubernetes transform")
		logrus.Debugf("Total services to be transformed : %d", len(ir.Services))
		apis := []apiresource.IAPIResource{&apiresource.KnativeService{}}
		files, err := apiresource.TransformIRAndPersist(irtypes.NewEnhancedIRFromIR(ir), tempDest, apis, clusterConfig, t.KnativeConfig.SetDefaultValuesInYamls)
		if err != nil {
			logrus.Errorf("Unable to transform and persist IR : %s", err)
			return nil, nil, err
		}
		for _, f := range files {
			destPath, err := filepath.Rel(t.Env.TempPath, f)
			if err != nil {
				logrus.Errorf("Invalid yaml path : %s", destPath)
				continue
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.DefaultPathMappingType,
				SrcPath:  f,
				DestPath: destPath,
			})
		}
		na := transformertypes.Artifact{
			Name: t.Config.Name,
			Type: artifacts.KubernetesYamlsArtifactType,
			Paths: map[transformertypes.PathType][]string{
				artifacts.KubernetesYamlsPathType: {deployKnativeDir},
			},
		}
		createdArtifacts = append(createdArtifacts, na)
		logrus.Debugf("Total transformed objects : %d", len(files))
	}
	return pathMappings, createdArtifacts, nil
}
