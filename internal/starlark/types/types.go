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

package types

// MapT is the most common map type
type MapT = map[string]interface{}

// K8sResourceT is a k8s resource
type K8sResourceT = MapT

// KindsAPIVersionsT is a map from kind to a list apiVersions used for filtering k8s resources
type KindsAPIVersionsT = map[string][]string

const (
	// TransformFileExtension is the file extension for transformation starlark scripts
	TransformFileExtension = "star"
)

// TransformT is a transformation that can be applied to k8s resources
type TransformT interface {
	// Transform applies the transformation on the given k8s resource
	// The k8s resource is changed in place, so the returned resource
	// could be the same object as the input resource.
	Transform(k8sResource K8sResourceT) (K8sResourceT, error)
	// Filter returns true if the transformation can be applied to the given k8s resource
	Filter(k8sResource K8sResourceT) (bool, error)
}

// DynamicQuestionFnT is provided the questions that must be answered while the script is running
type DynamicQuestionFnT = func(question interface{}) (answer interface{}, err error)
