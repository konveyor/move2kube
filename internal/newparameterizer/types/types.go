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

import (
	startypes "github.com/konveyor/move2kube/internal/starlark/types"
)

type EnvironmentT string

// ParameterizerT is a transformation that can be applied to k8s resources
type ParameterizerT interface {
	// Parameterize applies the parameterization on the given k8s resource
	// The k8s resource is changed in place, so the returned resource
	// could be the same object as the input resource.
	Parameterize(k8sResource startypes.K8sResourceT, values map[string]interface{}, env EnvironmentT) (startypes.K8sResourceT, error)
	// Filter returns true if the parameterization can be applied to the given k8s resource
	Filter(k8sResource startypes.K8sResourceT) (bool, error)
}

const (
	DevEnv     EnvironmentT = "dev"
	StagingEnv EnvironmentT = "staging"
	ProdEnv    EnvironmentT = "prod"
)

const (
	// ParameterizerKind is the k8s kind for parameterizers
	ParameterizerKind = "Parameterizer"
)
