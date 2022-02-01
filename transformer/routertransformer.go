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

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
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
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.RouterConfig, err)
		return err
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
		logrus.Errorf("Unable to get transformer selector : %s", err)
		return nil, nil, err
	}
	transformers := GetInitializedTransformersF(filters)
	transformerNames := []string{}
	for _, tr := range transformers {
		c, _ := tr.GetConfig()
		transformerNames = append(transformerNames, c.Name)
	}
	if len(transformerNames) == 0 {
		err := fmt.Errorf("no transformers to choose for router %s", t.Config.Name)
		logrus.Errorf("%s", err)
		return nil, nil, err
	}
	for _, a := range newArtifacts {
		filledID, err := t.GetStringFromTemplate(t.RouterConfig.RouterQuestion.ID, a)
		if err != nil {
			logrus.Errorf("Unable to get full string for ID %s : %s", t.RouterConfig.RouterQuestion.ID, err)
		}
		filledDesc, err := t.GetStringFromTemplate(t.RouterConfig.RouterQuestion.Desc, a)
		if err != nil {
			logrus.Errorf("Unable to get full string for Desc %s : %s", t.RouterConfig.RouterQuestion.Desc, err)
		}
		filledHints := []string{}
		for _, hint := range t.RouterConfig.RouterQuestion.Hints {
			filledHint, err := t.GetStringFromTemplate(hint, a)
			if err != nil {
				logrus.Errorf("Unable to get full string for Hint %s : %s", hint, err)
			}
			filledHints = append(filledHints, filledHint)
		}
		logrus.Debugf("Using %s router to route %s artifact between %+v", t.Config.Name, a.Type, transformerNames)
		transformerName := qaengine.FetchSelectAnswer(filledID, filledDesc, filledHints, transformerNames[0], transformerNames)
		a.ProcessWith.MatchExpressions = []metav1.LabelSelectorRequirement{{
			Key:      transformertypes.LabelName,
			Operator: metav1.LabelSelectorOpIn,
			Values:   []string{transformerName},
		}}
		artifactsCreated = append(artifactsCreated, a)
	}
	return nil, artifactsCreated, nil
}

// GetStringFromTemplate Translates question properties from templates to string
func (t *Router) GetStringFromTemplate(templateString string, artifact transformertypes.Artifact) (filledString string, err error) {
	// To ensure we use the artifact json struct tags instead of artifact property names
	objJSONBytes, err := json.Marshal(artifact)
	if err != nil {
		logrus.Errorf("Error while marshalling object %+v to json. Error: %q", artifact, err)
		return templateString, err
	}
	var jsonObj interface{}
	if err := yaml.Unmarshal(objJSONBytes, &jsonObj); err != nil {
		logrus.Errorf("Unable to unmarshal the json as yaml:\n%s\nError: %q", objJSONBytes, err)
		return templateString, err
	}
	return common.GetStringFromTemplate(templateString, jsonObj)
}
