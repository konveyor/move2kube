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

package transformer

import (
	"encoding/json"
	"fmt"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	"github.com/konveyor/move2kube-wasm/qaengine"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Router implements Transformer interface
type Router struct {
	Config       transformertypes.Transformer
	Env          *environment.Environment
	RouterConfig *RouterYamlConfig
}

// RouterQuestion stores the templated question for Router
type RouterQuestion struct {
	ID    string   `yaml:"id" json:"id"`
	Desc  string   `yaml:"description,omitempty" json:"description,omitempty"`
	Hints []string `yaml:"hints,omitempty" json:"hints,omitempty"`
}

// RouterYamlConfig stores the yaml configuration for Router transformer
type RouterYamlConfig struct {
	TransformerSelector metav1.LabelSelector `yaml:"transformerSelector" json:"transformerSelector"`
	RouterQuestion      RouterQuestion       `yaml:"question" json:"question"`
}

// Init Initializes the transformer
func (t *Router) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.RouterConfig = &RouterYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.RouterConfig)
	if err != nil {
		return fmt.Errorf("failed to load config for Transformer %+v into %T . Error: %w", t.Config.Spec.Config, t.RouterConfig, err)
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Router) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detects if necessary
func (t *Router) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return nil, nil
}

// Transform transforms the artifacts
func (t *Router) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	artifactsCreated := []transformertypes.Artifact{}
	filters, err := metav1.LabelSelectorAsSelector(&t.RouterConfig.TransformerSelector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get transformer selector. Error: %w", err)
	}
	transformers := GetInitializedTransformersF(filters)
	transformerNames := []string{}
	for _, tr := range transformers {
		c, _ := tr.GetConfig()
		transformerNames = append(transformerNames, c.Name)
	}
	if len(transformerNames) == 0 {
		return nil, nil, fmt.Errorf("no transformers to choose for router '%s'", t.Config.Name)
	}
	for _, newArtifact := range newArtifacts {
		filledID, err := t.GetStringFromTemplate(t.RouterConfig.RouterQuestion.ID, newArtifact)
		if err != nil {
			logrus.Errorf("failed to get full string for ID '%s' . Error: %q", t.RouterConfig.RouterQuestion.ID, err)
			continue
		}
		filledDesc, err := t.GetStringFromTemplate(t.RouterConfig.RouterQuestion.Desc, newArtifact)
		if err != nil {
			logrus.Errorf("failed to get full string for Desc '%s' . Error: %q", t.RouterConfig.RouterQuestion.Desc, err)
			continue
		}
		filledHints := []string{}
		for _, hint := range t.RouterConfig.RouterQuestion.Hints {
			filledHint, err := t.GetStringFromTemplate(hint, newArtifact)
			if err != nil {
				logrus.Errorf("failed to get full string for Hint '%s' . Error: %q", hint, err)
				continue
			}
			filledHints = append(filledHints, filledHint)
		}
		logrus.Debugf("using the '%s' router to route the '%s' artifact between the following transformers: %+v", t.Config.Name, newArtifact.Type, transformerNames)
		transformerName := qaengine.FetchSelectAnswer(filledID, filledDesc, filledHints, transformerNames[0], transformerNames, nil)
		logrus.Debugf("routing to the transformer named '%s'", transformerName)
		if newArtifact.ProcessWith.MatchLabels == nil {
			newArtifact.ProcessWith.MatchLabels = map[string]string{}
		}
		newArtifact.ProcessWith.MatchLabels[transformertypes.LabelName] = transformerName
		artifactsCreated = append(artifactsCreated, newArtifact)
	}
	return nil, artifactsCreated, nil
}

// GetStringFromTemplate Translates question properties from templates to string
func (t *Router) GetStringFromTemplate(templateString string, artifact transformertypes.Artifact) (filledString string, err error) {
	// To ensure we use the artifact json struct tags instead of artifact property names
	objJSONBytes, err := json.Marshal(artifact)
	if err != nil {
		return templateString, fmt.Errorf("failed to marshal the object %+v to json. Error: %w", artifact, err)
	}
	var jsonObj interface{}
	if err := yaml.Unmarshal(objJSONBytes, &jsonObj); err != nil {
		return templateString, fmt.Errorf("failed to unmarshal the json as yaml. Error: %w Actual: %s", err, objJSONBytes)
	}
	return common.GetStringFromTemplate(templateString, jsonObj)
}
