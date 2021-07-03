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

package types

import (
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/qaengine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// K8sResourceT is a k8s resource
type K8sResourceT = map[string]interface{}

// PatchOpT has Patch
type PatchOpT string

// ParamTargetT has Param Target
type ParamTargetT string

// HelmValuesT has Helm Values
type HelmValuesT map[string]interface{}

// PackagingFileT is the file format for the packaging
type PackagingFileT struct {
	metav1.TypeMeta   `yaml:",inline" json:",inline"`
	metav1.ObjectMeta `yaml:"metadata" json:"metadata"`
	Spec              PackagingSpecT `yaml:"spec" json:"spec"`
}

// PackagingSpecT is the spec inside the packaging file
type PackagingSpecT struct {
	Paths             []PackagingSpecPathT `yaml:"paths" json:"paths"`
	ParameterizerRefs []string             `yaml:"parameterizerRefs,omitempty" json:"parameterizerRefs,omitempty"`
	Parameterizers    []ParameterizerT     `yaml:"parameterizers,omitempty" json:"parameterizers,omitempty"`
}

// PackagingSpecPathT is the set of source paths to be parameterized
type PackagingSpecPathT struct {
	Src           string   `yaml:"src" json:"src"`
	Out           string   `yaml:"out,omitempty" json:"out,omitempty"`
	Helm          string   `yaml:"helm,omitempty" json:"helm,omitempty"`
	HelmChartName string   `yaml:"helmChartName,omitempty" json:"helmChartName,omitempty"`
	Kustomize     string   `yaml:"kustomize,omitempty" json:"kustomize,omitempty"`
	OCTemplates   string   `yaml:"ocTemplates,omitempty" json:"ocTemplates,omitempty"`
	Envs          []string `yaml:"envs,omitempty" json:"envs,omitempty"`
}

// ParameterizerFileT is the file format for the parameterizers
type ParameterizerFileT struct {
	metav1.TypeMeta   `yaml:",inline" json:",inline"`
	metav1.ObjectMeta `yaml:"metadata" json:"metadata"`
	Spec              ParameterizerSpecT `yaml:"spec" json:"spec"`
}

// ParameterizerSpecT is the spec inside the parameterizers file
type ParameterizerSpecT struct {
	Parameterizers []ParameterizerT `yaml:"parameterizers" json:"parameterizers"`
}

// ParameterizerT is a paramterizer
type ParameterizerT struct {
	Target     string            `yaml:"target" json:"target"`
	Template   string            `yaml:"template,omitempty" json:"template,omitempty"`
	Regex      string            `yaml:"regex,omitempty" json:"regex,omitempty"`
	Default    interface{}       `yaml:"default,omitempty" json:"default,omitempty"`
	Question   *qaengine.Problem `yaml:"question,omitempty" json:"question,omitempty"`
	Filters    []FilterT         `yaml:"filters,omitempty" json:"filters,omitempty"`
	Parameters []ParameterT      `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// FilterT is used to choose the k8s resources that the parameterizer should be applied on
type FilterT struct {
	Kind       string   `yaml:"kind,omitempty" json:"kind,omitempty"`
	APIVersion string   `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	Name       string   `yaml:"name,omitempty" json:"name,omitempty"`
	Envs       []string `yaml:"envs,omitempty" json:"envs,omitempty"`
}

// ParameterT is used to specify the environment specific defaults for the keys in the template
type ParameterT struct {
	Name              string            `yaml:"name" json:"name"`
	Default           string            `yaml:"default,omitempty" json:"default,omitempty"`
	HelmTemplate      string            `yaml:"helmTemplate,omitempty" json:"helmTemplate,omitempty"`
	OpenshiftTemplate string            `yaml:"openshiftTemplate,omitempty" json:"openshiftTemplate,omitempty"`
	Values            []ParameterValueT `yaml:"values,omitempty" json:"values,omitempty"`
}

// ParameterValueT is used to specify the value for a parameter in different contexts
type ParameterValueT struct {
	Envs         []string          `yaml:"envs,omitempty" json:"envs,omitempty"`
	Kind         string            `yaml:"kind,omitempty" json:"kind,omitempty"`
	APIVersion   string            `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	MetadataName string            `yaml:"metadataName,omitempty" json:"metadataName,omitempty"`
	Custom       map[string]string `yaml:"custom,omitempty" json:"custom,omitempty"`
	Value        string            `yaml:"value" json:"value"`
}

// PatchMetadataT is contains the target k8s resources and the patch filename
type PatchMetadataT struct {
	Path   string               `yaml:"path"`
	Target PatchMetadataTargetT `yaml:"target"`
}

// PatchMetadataTargetT is used to specify the target k8s resource that the json path should be applied on (this is specific to kustomize)
type PatchMetadataTargetT struct {
	Group     string `yaml:"group"`
	Version   string `yaml:"version"`
	Kind      string `yaml:"kind"`
	Namespace string `yaml:"namespace,omitempty"`
	Name      string `yaml:"name"`
}

// PatchT represents a single json patch https://tools.ietf.org/html/rfc6902
type PatchT struct {
	Op    PatchOpT    `yaml:"op"`
	Path  string      `yaml:"path"`
	Value interface{} `yaml:"value"`
}

// OCParamT is the type for a single Openshift Templates parameter
type OCParamT struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// ParamOrStringT is string along with a flag to indicate if it is a parameter
type ParamOrStringT struct {
	IsParam bool
	Data    string
}

const (
	// PackagingKind is the kind for Packaging yamls
	PackagingKind = "Packaging"
	// ParameterizerKind is the kind for Parameterizer yamls
	ParameterizerKind = "Parameterizer"
	// ReplaceOp replaces the value at the key with a different value
	ReplaceOp PatchOpT = "replace"
	// AddOp inserts a value at the key
	AddOp PatchOpT = "add"
	// RemoveOp removes the value at the key
	RemoveOp PatchOpT = "remove"
	// TargetHelm is used when the target is the parameterization of Helm
	TargetHelm ParamTargetT = "helm"
	// TargetKustomize is used when the target is the parameterization of Kustomize
	TargetKustomize ParamTargetT = "kustomize"
	// TargetOCTemplates is used when the target is the parameterization of OCTemplates
	TargetOCTemplates ParamTargetT = "octemplates"
	// ParamQuesIDPrefix is used as a prefix when the key is not specified in the questions in a parameterizer
	ParamQuesIDPrefix = common.BaseKey + common.Delim + "parameterization"
)
