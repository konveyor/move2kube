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
	operatortypes "github.com/konveyor/move2kube/types/operator"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

const (
	defaultOperatorPath = common.DeployDir + string(os.PathSeparator) + "operators"
)

// OperatorTransformer implements the Transformer interface
type OperatorTransformer struct {
	Config                    transformertypes.Transformer
	Env                       *environment.Environment
	OperatorTransformerConfig *OperatorTransformerConfig
}

// OperatorTransformerConfig stores the transformer specific configuration
type OperatorTransformerConfig struct {
	OutputPath string `yaml:"outputPath"`
}

// Init Initializes the transformer
func (t *OperatorTransformer) Init(tc transformertypes.Transformer, e *environment.Environment) error {
	t.Config = tc
	t.Env = e
	t.OperatorTransformerConfig = &OperatorTransformerConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.OperatorTransformerConfig); err != nil {
		return fmt.Errorf("failed to load the config for Transformer %+v into %T . Error: %q", t.Config.Spec.Config, t.OperatorTransformerConfig, err)
	}
	if t.OperatorTransformerConfig.OutputPath == "" {
		t.OperatorTransformerConfig.OutputPath = defaultOperatorPath
	}
	return nil
}

// GetConfig returns the transformer config
func (t *OperatorTransformer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each subdirectory
func (t *OperatorTransformer) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	return nil, nil
}

// Transform transforms artifacts
func (t *OperatorTransformer) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	logrus.Trace("OperatorTransformer Transform start")
	defer logrus.Trace("OperatorTransformer Transform end")
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		if newArtifact.Type != operatortypes.OperatorsToInitializeArtifactType {
			continue
		}
		operatorConfig := operatortypes.OperatorArtifactConfig{}
		if err := newArtifact.GetConfig(operatortypes.OperatorsToInitializeArtifactConfigType, &operatorConfig); err != nil {
			logrus.Errorf("failed to load the config from the operator artifact %+v . Error: %q", newArtifact, err)
			continue
		}
		for _, operator := range operatorConfig.Operators {
			operatorFilename := common.NormalizeForFilename(operator.OperatorName)
			if !operator.InstallPlanApproval.IsValid() {
				logrus.Warnf("the install plan for this operator is invalid. Operator: %+v", operator)
				continue
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.TemplatePathMappingType,
				SrcPath:        "subscription.yaml",
				DestPath:       filepath.Join(t.OperatorTransformerConfig.OutputPath, operatorFilename, operatorFilename+".yaml"),
				TemplateConfig: operator,
			})
		}
	}
	if len(pathMappings) > 0 {
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "README.md"),
			DestPath: filepath.Join(t.OperatorTransformerConfig.OutputPath, "README.md"),
		})
	}
	return pathMappings, createdArtifacts, nil
}
