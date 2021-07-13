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

// Service is the type that stores a plan service
type ServicePlan []TransformerPlan

// Transformer stores transformer option
type TransformerPlan struct {
	Mode              Mode                       `yaml:"mode" json:"mode"` // container, customresource, service, generic
	Name              string                     `yaml:"transformerName" json:"transformerName"`
	ArtifactTypes     []ArtifactType             `yaml:"generates,omitempty" json:"generates,omitempty"`
	BaseArtifactTypes []ArtifactType             `yaml:"generatedBases,omitempty" json:"generatedBases,omitempty"`
	Paths             map[PathType][]string      `yaml:"paths,omitempty" json:"paths,omitempty" m2kpath:"normal"`
	Configs           map[ConfigType]interface{} `yaml:"configs,omitempty" json:"configs,omitempty"`
}
