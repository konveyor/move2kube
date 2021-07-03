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

package plan

import (
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

// PlanKind is kind of plan file
const PlanKind types.Kind = "Plan"

// Plan defines the format of plan
type Plan struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             Spec `yaml:"spec,omitempty"`
}

// Service is the type that stores a plan service
type Service []Transformer

// Spec stores the data about the plan
type Spec struct {
	RootDir           string `yaml:"rootDir"`
	ConfigurationsDir string `yaml:"configurationsDir,omitempty"`

	Services map[string]Service `yaml:"services"` //[servicename]

	TargetCluster TargetClusterType `yaml:"targetCluster,omitempty"`
	Configuration Configuration     `yaml:"configuration,omitempty"`
}

// Configuration stores all configurations related to the plan
type Configuration struct {
	Transformers   map[string]string `yaml:"transformers,omitempty" m2kpath:"normal"`   //[name]filepath
	TargetClusters map[string]string `yaml:"targetClusters,omitempty" m2kpath:"normal"` //[clustername]filepath
}

// TargetClusterType contains either the type of the target cluster or path to a file containing the target cluster metadata.
// Specify one or the other, not both.
type TargetClusterType struct {
	Type string `yaml:"type,omitempty"`
	Path string `yaml:"path,omitempty" m2kpath:"normal"`
}

// Transformer stores transformer option
type Transformer struct {
	Mode                   transformertypes.Mode                       `yaml:"mode" json:"mode"` // container, customresource, service, generic
	Name                   string                                      `yaml:"transformerName" json:"transformerName"`
	ArtifactTypes          []transformertypes.ArtifactType             `yaml:"generates,omitempty" json:"generates,omitempty"`
	ExclusiveArtifactTypes []transformertypes.ArtifactType             `yaml:"exclusive,omitempty" json:"exclusive,omitempty"`
	Paths                  map[transformertypes.PathType][]string      `yaml:"paths,omitempty" json:"paths,omitempty" m2kpath:"normal"`
	Configs                map[transformertypes.ConfigType]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
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
			Services:      map[string]Service{},
			TargetCluster: TargetClusterType{Type: common.DefaultClusterType},
			Configuration: Configuration{
				Transformers:   map[string]string{},
				TargetClusters: map[string]string{},
			},
		},
	}
	return plan
}

// MergeServices merges two service maps
func MergeServices(s1 map[string]Service, s2 map[string]Service) map[string]Service {
	if s1 == nil {
		return s2
	}
	for s2n, s2t := range s2 {
		s1[s2n] = append(s1[s2n], s2t...)
	}
	return s1
}
