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
	FilePath           string                              `yaml:"-"`
	Class              string                              `yaml:"class"`
	DirectoryDetect    DirectoryDetect                     `yaml:"directoryDetect"`
	ExternalFiles      map[string]string                   `yaml:"externalFiles"` // [source]destination
	ArtifactsToProcess map[string]ArtifactPreprocessConfig `yaml:"consumes"`      // plantypes.ArtifactType
	TemplatesDir       string                              `yaml:"templates"`     // Relative to yaml directory or working directory in image
	Config             interface{}                         `yaml:"config"`
}

// DirectoryDetect stores the config on how to iterate over the directories
type DirectoryDetect struct {
	Levels                      int  `yaml:"levels"`                      // Supports only 0,1 and -1 currently
	HonorM2KIgnore              bool `yaml:"honorM2KIgnore"`              // TODO: Add support
	IgnoreServiceSubdirectories bool `yaml:"ignoreServiceSubdirectories"` // TODO: Add support
}

// ArtifactPreprocessConfig stores config for how to preprocess artifacts
type ArtifactPreprocessConfig struct {
	Merge bool `yaml:"merge"`
}

// NewTransformer creates a new instance of tansformer
func NewTransformer() Transformer {
	return Transformer{
		TypeMeta: types.TypeMeta{
			Kind:       TransformerKind,
			APIVersion: types.SchemeGroupVersion.String(),
		},
		Spec: TransformerSpec{
			TemplatesDir: "templates/",
			DirectoryDetect: DirectoryDetect{
				Levels:                      -1,
				HonorM2KIgnore:              true,
				IgnoreServiceSubdirectories: true,
			},
		},
	}
}
