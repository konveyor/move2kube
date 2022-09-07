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
	"k8s.io/apimachinery/pkg/labels"
)

// TransformerKind represents the Transformer kind
const TransformerKind = "Transformer"

// Transformer defines definition of cf runtime instance apps file
type Transformer struct {
	types.TypeMeta   `yaml:",inline" json:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Spec             TransformerSpec `yaml:"spec,omitempty" json:"spec,omitempty"`
}

// TransformerSpec stores the data
type TransformerSpec struct {
	FilePath           string                                 `yaml:"-" json:"-"`
	Class              string                                 `yaml:"class" json:"class"`
	Isolated           bool                                   `yaml:"isolated" json:"isolated"`
	DirectoryDetect    DirectoryDetect                        `yaml:"directoryDetect" json:"directoryDetect"`
	ExternalFiles      map[string]string                      `yaml:"externalFiles" json:"externalFiles"` // [source]destination
	ConsumedArtifacts  map[ArtifactType]ArtifactProcessConfig `yaml:"consumes" json:"consumes"`
	ProducedArtifacts  map[ArtifactType]ProducedArtifact      `yaml:"produces" json:"produces"`
	Dependency         interface{}                            `yaml:"dependency" json:"dependency"` // metav1.LabelSelector
	Override           interface{}                            `yaml:"override" json:"override"`     // metav1.LabelSelector
	DependencySelector labels.Selector                        `yaml:"-" json:"-"`
	OverrideSelector   labels.Selector                        `yaml:"-" json:"-"`
	TemplatesDir       string                                 `yaml:"templates" json:"templates"` // Relative to yaml directory or working directory in image
	Config             interface{}                            `yaml:"config" json:"config"`
	InvokesByDefault   InvokesByDefault                       `yaml:"invokesByDefault" json:"invokesByDefault"`
}

// InvokesByDefault stores config to toggle default transformers
type InvokesByDefault struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

// DirectoryDetect stores the config on how to iterate over the directories
type DirectoryDetect struct {
	Levels int `yaml:"levels"` // Supports only 0,1 and -1 currently - default behaviour is -1, when directory detect section is missing
}

// ArtifactProcessingMode denotes the artifact processing mode
type ArtifactProcessingMode string

const (
	// Normal denotes normal consumes
	Normal ArtifactProcessingMode = "Normal"
	// MandatoryPassThrough denotes pass through
	MandatoryPassThrough ArtifactProcessingMode = "MandatoryPassThrough"
	// OnDemandPassThrough is generally used for dependencies
	OnDemandPassThrough ArtifactProcessingMode = "OnDemandPassThrough"
)

// ArtifactProcessConfig stores config for preprocessing artifact
type ArtifactProcessConfig struct {
	Merge    bool                   `yaml:"merge" json:"merge"`
	Mode     ArtifactProcessingMode `yaml:"mode" json:"mode"`
	Disabled bool                   `yaml:"disabled" json:"disabled"` // default is false
}

// ProducedArtifact stores config for postprocessing produced artifact
type ProducedArtifact struct {
	ChangeTypeTo ArtifactType `yaml:"changeTypeTo" json:"changeTypeTo"`
	Disabled     bool         `yaml:"disabled" json:"disabled"`
}

// NewTransformer creates a new instance of tansformer
func NewTransformer() Transformer {
	return Transformer{
		TypeMeta: types.TypeMeta{
			Kind:       TransformerKind,
			APIVersion: types.SchemeGroupVersion.String(),
		},
		ObjectMeta: types.ObjectMeta{
			Labels: map[string]string{},
		},
		Spec: TransformerSpec{
			TemplatesDir: "templates/",
			DirectoryDetect: DirectoryDetect{
				Levels: 0,
			},
		},
	}
}
