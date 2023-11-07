/*
 *  Copyright IBM Corporation 2022
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

package plan

import (
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/types"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// transformertypes "github.com/konveyor/move2kube/types/transformer"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PlanKind is kind of plan file
const PlanKind types.Kind = "Plan"

// Plan defines the format of plan
type Plan struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             Spec `yaml:"spec,omitempty"`
}

// Spec stores the data about the plan
type Spec struct {
	SourceDir         string `yaml:"sourceDir"`
	CustomizationsDir string `yaml:"customizationsDir,omitempty"`

	Services map[string][]PlanArtifact `yaml:"services"` //[servicename]

	TransformerSelector          metav1.LabelSelector `yaml:"transformerSelector,omitempty"`
	Transformers                 map[string]string    `yaml:"transformers,omitempty" m2kpath:"normal"` //[name]filepath
	InvokedByDefaultTransformers []string             `yaml:"invokedByDefaultTransformers,omitempty"`
	DisabledTransformers         map[string]string    `yaml:"disabledTransformers,omitempty" m2kpath:"normal"` //[name]filepath
}

// PlanArtifact stores the artifact with the transformerName
type PlanArtifact struct {
	ServiceName               string `yaml:"-"`
	TransformerName           string `yaml:"transformerName"`
	transformertypes.Artifact `yaml:",inline"`
}

// NewPlan creates a new plan
// Sets the version and optionally fills in some default values
func NewPlan() Plan {
	plan := Plan{
		TypeMeta: types.TypeMeta{
			Kind:       string(PlanKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
		ObjectMeta: types.ObjectMeta{
			Name: common.DefaultProjectName,
		},
		Spec: Spec{
			Services:                     map[string][]PlanArtifact{},
			Transformers:                 map[string]string{},
			InvokedByDefaultTransformers: []string{},
		},
	}
	return plan
}

// MergeServices merges two service maps
func MergeServices(s1 map[string][]PlanArtifact, s2 map[string][]PlanArtifact) map[string][]PlanArtifact {
	if s1 == nil {
		return s2
	}
	for s2n, s2t := range s2 {
		s1[s2n] = append(s1[s2n], s2t...)
	}
	return s1
}

// MergeServicesT merges two service maps
func MergeServicesT(s1 map[string][]transformertypes.Artifact, s2 map[string][]transformertypes.Artifact) map[string][]transformertypes.Artifact {
	if s1 == nil {
		return s2
	}
	for s2n, s2t := range s2 {
		s1[s2n] = append(s1[s2n], s2t...)
	}
	return s1
}
