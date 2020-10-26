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

package collection

import (
	"github.com/konveyor/move2kube/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// CfContainerizersMetadataKind defines kind of cfcontainerizers
const CfContainerizersMetadataKind types.Kind = "CfContainerizers"

// CfContainerizers is the file structure of cfcontainerizers
type CfContainerizers struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             CfContainerizersSpec `yaml:"spec,omitempty"`
}

// CfContainerizersSpec stores the data
type CfContainerizersSpec struct {
	BuildpackContainerizers []BuildpackContainerizer `yaml:"buildpackContainerizers"`
}

// BuildpackContainerizer defines the structure of Buildpack and the containerization strategy to be used for it
type BuildpackContainerizer struct {
	BuildpackName                 string                            `yaml:"buildpackName"`
	ContainerBuildType            plantypes.ContainerBuildTypeValue `yaml:"containerBuildType"`
	ContainerizationTargetOptions []string                          `yaml:"targetOptions,omitempty"`
}

// NewCfContainerizers creates new CfContainerizers instance
func NewCfContainerizers() CfContainerizers {
	return CfContainerizers{
		TypeMeta: types.TypeMeta{
			Kind:       string(CfContainerizersMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}
