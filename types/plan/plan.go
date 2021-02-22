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

// Spec stores the data about the plan
type Spec struct {
	RootDir             string                                   `yaml:"rootDir"`
	Generators          []string                                 `yaml:"generators"` // Generator name or image name
	K8sFiles            []string                                 `yaml:"kubernetesYamls,omitempty" m2kpath:"normal"`
	Services            map[string]Service                       `yaml:"services"`                                       //[servicename]
	TargetInfoArtifacts map[TargetInfoArtifactTypeValue][]string `yaml:"targetInfoArtifacts,omitempty" m2kpath:"normal"` //[targetinfoartifacttype][List of artifacts]
	TargetCluster       TargetClusterType                        `yaml:"targetCluster,omitempty"`
}

// TargetClusterType contains either the type of the target cluster or path to a file containing the target cluster metadata.
// Specify one or the other, not both.
type TargetClusterType struct {
	Type string `yaml:"type,omitempty"`
	Path string `yaml:"path,omitempty" m2kpath:"normal"`
}

// Service defines a plan service
type Service struct {
	SourceArtifacts   []SourceArtifact   `yaml:"sourceArtifacts,omitempty"`
	GenerationOptions []GenerationOption `yaml:"generatorOptions,omitempty"`
}

func (p *Plan) AddServicesToPlan(services map[string]Service) {
	for sn, s := range services {
		if os, ok := p.Spec.Services[sn]; ok {
			os.Merge(s)
			p.Spec.Services[sn] = os
		} else {
			p.Spec.Services[sn] = s
		}
	}
}

func (s *Service) Merge(ns Service) {
	for _, sa := range ns.SourceArtifacts {
		s.AddSourceArtifact(sa)
	}
	s.GenerationOptions = append(s.GenerationOptions, ns.GenerationOptions...)
}

func (s *Service) AddSourceArtifact(sa SourceArtifact) {
	found := false
	for osai, osa := range s.SourceArtifacts {
		if osa.Type == sa.Type && osa.ID == sa.ID {
			s.SourceArtifacts[osai].Artifacts = common.MergeStringSlices(s.SourceArtifacts[osai].Artifacts, sa.Artifacts)
			found = true
			break
		}
	}
	if !found {
		s.SourceArtifacts = append(s.SourceArtifacts, sa)
	}
}

func (s *Service) AddGenerationOption(o GenerationOption) {
	s.GenerationOptions = append(s.GenerationOptions, o)
}

// SourceArtifact stores information about a source artifact
type SourceArtifact struct {
	ID        string                  `yaml:"id"`
	Type      SourceArtifactTypeValue `yaml:"type"`
	Artifacts []string                `yaml:"artifacts" m2kpath:"if:Type:in:DockerCompose,CfManifest,CfRunningManifest,Dockerfile"` //[translationartifacttype][List of artifacts]
}

// GenerationOption stores generation target option
type GenerationOption struct {
	Mode   string            `yaml:"mode"` // container, operator, service
	Name   string            `yaml:"name"`
	Config interface{}       `yaml:"config"`
	Paths  map[string]string `yaml:"paths" m2kpath:"normal"`
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
