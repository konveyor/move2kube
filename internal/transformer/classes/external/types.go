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

package external

import (
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

// DetectOutput structure is the data format for receiving data from starlark detect functions
type DetectOutput struct {
	NamedServices   map[string]transformertypes.ServicePlan `yaml:"namedServices,omitempty" json:"namedServices,omitempty"`
	UnNamedServices []transformertypes.TransformerPlan      `yaml:"unnamedServices,omitempty" json:"unnamedServices,omitempty"`
}

// TransformOutput structure is the data format for receiving data from starlark transform functions
type TransformOutput struct {
	PathMappings     []transformertypes.PathMapping `yaml:"pathMappings,omitempty" json:"pathMappings,omitempty"`
	CreatedArtifacts []transformertypes.Artifact    `yaml:"artifacts,omitempty" json:"artifacts,omitempty"`
}
