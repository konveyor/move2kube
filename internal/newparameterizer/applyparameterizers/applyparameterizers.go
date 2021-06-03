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

package applyparameterizers

import (
	"github.com/konveyor/move2kube/internal/newparameterizer/types"
	startypes "github.com/konveyor/move2kube/internal/starlark/types"
	log "github.com/sirupsen/logrus" // TODO
)

// ApplyParameterizers applies the given parameterizers to the given k8s resources
func ApplyParameterizers(parameterizers []types.ParameterizerT, k8sResources []startypes.K8sResourceT) ([]startypes.K8sResourceT, map[string]interface{}, error) {
	log.Trace("start ApplyParameterizers")
	defer log.Trace("end ApplyParameterizers")
	var err error
	values := map[string]interface{}{}
	for _, parameterizer := range parameterizers {
		k8sResources, values, err = applyParameterizer(parameterizer, k8sResources, values)
		if err != nil {
			return k8sResources, values, err
		}
	}
	return k8sResources, values, nil
}

func applyParameterizer(parameterizer types.ParameterizerT, k8sResources []startypes.K8sResourceT, values map[string]interface{}) ([]startypes.K8sResourceT, map[string]interface{}, error) {
	log.Trace("start applyParameterizer")
	defer log.Trace("end applyParameterizer")
	filteredIdxs, err := filterK8sResources(parameterizer, k8sResources)
	if err != nil {
		return k8sResources, values, err
	}
	for _, filteredIdx := range filteredIdxs {
		k8sResource := k8sResources[filteredIdx]
		parameterizedK8sResource, err := parameterizer.Parameterize(k8sResource, values, types.DevEnv) // TODO: accept the environment from the caller
		if err != nil {
			return k8sResources, values, err
		}
		k8sResources[filteredIdx] = parameterizedK8sResource
	}
	return k8sResources, values, nil
}

func filterK8sResources(parameterizer types.ParameterizerT, k8sResources []startypes.K8sResourceT) ([]int, error) {
	log.Trace("start filterK8sResources")
	defer log.Trace("end filterK8sResources")
	idxs := []int{}
	for i, k8sResource := range k8sResources {
		ok, err := parameterizer.Filter(k8sResource)
		if err != nil {
			return idxs, err
		}
		if ok {
			idxs = append(idxs, i)
		}
	}
	return idxs, nil
}
