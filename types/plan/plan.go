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

// ContainerBuildTypeValue defines the containerization type
type ContainerBuildTypeValue string

// TargetInfoArtifactTypeValue defines the target info type
type TargetInfoArtifactTypeValue string

// SourceArtifactTypeValue defines the source artifact type
type SourceArtifactTypeValue string

// TargetArtifactTypeValue defines the target artifact type
type TargetArtifactTypeValue string

// PlanKind is kind of plan file
const PlanKind types.Kind = "Plan"

const (
	// DockerFileContainerBuildTypeValue defines the containerization type as docker file
	DockerFileContainerBuildTypeValue ContainerBuildTypeValue = "NewDockerfile"
	// ReuseDockerFileContainerBuildTypeValue defines the containerization type as reuse of dockerfile
	ReuseDockerFileContainerBuildTypeValue ContainerBuildTypeValue = "ReuseDockerfile"
	// ReuseContainerBuildTypeValue defines the containerization type as reuse of an existing container
	ReuseContainerBuildTypeValue ContainerBuildTypeValue = "ReuseImage"
	// CNBContainerBuildTypeValue defines the containerization type of cloud native buildpack
	CNBContainerBuildTypeValue ContainerBuildTypeValue = "CNB"
	// ManualContainerBuildTypeValue defines that the tool assumes that the image will be created manually
	ManualContainerBuildTypeValue ContainerBuildTypeValue = "Manual"
	// S2IContainerBuildTypeValue defines the containerization type of S2I
	S2IContainerBuildTypeValue ContainerBuildTypeValue = "S2I"
)

const (
	// ComposeFileArtifactType defines the source artifact type of Docker compose
	ComposeFileArtifactType SourceArtifactTypeValue = "DockerCompose"
	// ImageInfoArtifactType defines the source artifact type of image info
	ImageInfoArtifactType SourceArtifactTypeValue = "ImageInfo"
	// CfManifestArtifactType defines the source artifact type of cf manifest
	CfManifestArtifactType SourceArtifactTypeValue = "CfManifest"
	// CfRunningManifestArtifactType defines the source artifact type of a manifest of a running instance
	CfRunningManifestArtifactType SourceArtifactTypeValue = "CfRunningManifest"
	// DockerfileArtifactType defines the source artifact type of dockerfile
	DockerfileArtifactType SourceArtifactTypeValue = "Dockerfile"
)

const (
	GenerationModeContainer = "Container"
	GenerationModeOperator  = "Operator"
	GenerationModeService   = "Service"
)

const (
	GenerationPathTypeContext    = "Context"
	GenerationPathTypeDockerfile = "Dockerfile"
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
	Translators         map[string]string                        `yaml:"translators" m2kpath:"normal"`
	Services            map[string]Service                       `yaml:"services"` //[servicename]
	K8sFiles            []string                                 `yaml:"kubernetesYamls,omitempty" m2kpath:"normal"`
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
	ArtifactTypess         []string            `yaml:"artifacttypes,omitempty" json:"artifacts,omitempty"`
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
			K8sFiles:            []string{},
			Services:            make(map[string]Service),
			TargetInfoArtifacts: make(map[TargetInfoArtifactTypeValue][]string),
			TargetCluster:       TargetClusterType{Type: common.DefaultClusterType},
		},
	}
	return plan
}
