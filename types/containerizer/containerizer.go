/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package containerizer

import (
	"github.com/konveyor/move2kube/types"
)

// ContainerizerType defines containerizer types
type ContainerizerType string

const (
	// ContainerizerMetadataKind defines kind of cf runtime instance apps file
	ContainerizerMetadataKind types.Kind = "Containerizer"

	// DockerfileContainerizerType defines DockerfileContainerizer Type
	DockerfileContainerizerType ContainerizerType = "DockerfileContainerizer"
	// S2IContainerizerType defines S2IContainerizer Type
	S2IContainerizerType ContainerizerType = "S2IContainerizer"
	// CNBContainerizerType defines CNBContainerizer Type
	CNBContainerizerType ContainerizerType = "CNBContainerizer"
)

// DockerfileContainerizer defines definition of cf runtime instance apps file
type DockerfileContainerizer struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             DockerfileContainerizerSpec `yaml:"spec,omitempty"`
}

// DockerfileContainerizerSpec stores the data
type DockerfileContainerizerSpec struct {
	Type          ContainerizerType `yaml:"type"`
	DetectorImage Image             `yaml:"detectorImage"`
}

// Image defines the structure of a image
type Image struct {
	ImageName  string `yaml:"imageName"`
	Dockerfile string `yaml:"dockerfile"`
}

// NewDockerfileContainerizer creates a new instance of DockerfileContainerizer
func NewDockerfileContainerizer() DockerfileContainerizer {
	return DockerfileContainerizer{
		TypeMeta: types.TypeMeta{
			Kind:       string(ContainerizerMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}
