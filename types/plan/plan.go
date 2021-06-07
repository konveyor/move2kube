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

package plan

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types"
)

const (
	ProjectPathSourceArtifact = "ProjectPath"
)

const (
	ContainerBuildTargetArtifactType     = "ContainerBuild"
	K8sServiceMetadataTargetArtifactType = "KubernetesServiceMetadata"
)

// TargetInfoArtifactTypeValue defines the target info type
type TargetInfoArtifactTypeValue string

// PlanKind is kind of plan file
const PlanKind types.Kind = "Plan"

const (
	ModeContainer = "Container"
	ModeOperator  = "Operator"
	ModeService   = "Service"
)

const (
	// K8sClusterArtifactType defines target info
	K8sClusterArtifactType TargetInfoArtifactTypeValue = "KubernetesCluster"
)

// Plan defines the format of plan
type Plan struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty"`
	Spec              Spec `yaml:"spec,omitempty"`
}

type Service []Translator

// Spec stores the data about the plan
type Spec struct {
	RootDir             string                                   `yaml:"rootDir"`
	ConfigurationsDir   string                                   `yaml:"configurationsDir"`
	AllTranslators      map[string]string                        `yaml:"allTranslators" m2kpath:"normal"`
	Services            map[string]Service                       `yaml:"services"` //[servicename]
	TopTranslators      []Translator                             `yaml:"topTranslators"`
	TargetInfoArtifacts map[TargetInfoArtifactTypeValue][]string `yaml:"targetInfoArtifacts,omitempty" m2kpath:"normal"` //[targetinfoartifacttype][List of artifacts]
	TargetCluster       TargetClusterType                        `yaml:"targetCluster,omitempty"`
}

// TargetClusterType contains either the type of the target cluster or path to a file containing the target cluster metadata.
// Specify one or the other, not both.
type TargetClusterType struct {
	Type string `yaml:"type,omitempty"`
	Path string `yaml:"path,omitempty" m2kpath:"normal"`
}

// Translator stores translator option
type Translator struct {
	Mode                   string              `yaml:"mode" json:"mode"` // container, customresource, service, generic
	Name                   string              `yaml:"name" json:"name"`
	ArtifactTypes          []string            `yaml:"artifacttypes,omitempty" json:"artifacts,omitempty"`
	ExclusiveArtifactTypes []string            `yaml:"exclusiveArtifactTypes,omitempty" json:"exclusiveArtifacts,omitempty"`
	Config                 interface{}         `yaml:"config,omitempty" json:"config,omitempty"`
	Paths                  map[string][]string `yaml:"paths,omitempty" json:"paths,omitempty" m2kpath:"normal"`
}

func (p *Plan) AddServicesToPlan(services map[string][]Translator) {
	for sn, s := range services {
		if os, ok := p.Spec.Services[sn]; ok {
			p.Spec.Services[sn] = append(os, s...)
		} else {
			p.Spec.Services[sn] = s
		}
	}
}

// NewPlan creates a new plan
// Sets the version and optionally fills in some default values
func NewPlan() Plan {
	plan := Plan{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(PlanKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: common.DefaultProjectName,
		},
		Spec: Spec{
			Services:            make(map[string]Service),
			TargetInfoArtifacts: make(map[TargetInfoArtifactTypeValue][]string),
			TargetCluster:       TargetClusterType{Type: common.DefaultClusterType},
		},
	}
	return plan
}

func MergeServices(s1 map[string]Service, s2 map[string]Service) map[string]Service {
	if s1 == nil {
		return s2
	}
	for s2n, s2t := range s2 {
		s1[s2n] = append(s1[s2n], s2t...)
	}
	return s1
}
