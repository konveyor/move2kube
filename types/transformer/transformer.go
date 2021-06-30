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
	"github.com/konveyor/move2kube/types"
)

// TransformerKind represents the Transformer kind
const TransformerKind = "Transformer"

// Mode represents the mode of deployment artifacts
type Mode = string

const (
	// ModeContainer represents artifacts for container mode of deployment
	ModeContainer Mode = "Container"
	// ModeCR represents artifacts for custom resource mode of deployment
	ModeCR Mode = "CustomResource"
	// ModeService represents artifacts for service mode of deployment
	ModeService Mode = "Service" // Possibly Terraform
	// ModeCustom represents artifacts for custom mode of deployment
	ModeCustom Mode = "Custom"
)

// Transformer defines definition of cf runtime instance apps file
type Transformer struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             TransformerSpec `yaml:"spec,omitempty"`
}

// TransformerSpec stores the data
type TransformerSpec struct {
	FilePath           string      `yaml:"-"`
	Mode               Mode        `yaml:"mode"`
	Class              string      `yaml:"class"`
	ArtifactsToProcess []string    `yaml:"consumes"`  //plantypes.ArtifactType
	Artifacts          []string    `yaml:"generates"` //plantypes.ArtifactType
	ExclusiveArtifacts []string    `yaml:"exclusive"` //plantypes.ArtifactType
	TemplatesDir       string      `yaml:"templates"` //Relative to yaml directory or working directory in image
	Config             interface{} `yaml:"config"`
}

// NewTransformer creates a new instance of tansformer
func NewTransformer() Transformer {
	return Transformer{
		TypeMeta: types.TypeMeta{
			Kind:       TransformerKind,
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}
