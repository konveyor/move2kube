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

package runtransforms

import (
	"github.com/konveyor/move2kube/internal/starlark/types"
	log "github.com/sirupsen/logrus" // TODO
)

// ApplyTransforms applies the given transformations to the given k8s resources
func ApplyTransforms(transforms []types.TransformT, k8sResources []types.K8sResourceT) ([]types.K8sResourceT, error) {
	log.Trace("start applyTransforms")
	defer log.Trace("end applyTransforms")
	var err error
	for _, transform := range transforms {
		k8sResources, err = applyTransform(transform, k8sResources)
		if err != nil {
			return k8sResources, err
		}
	}
	return k8sResources, nil
}

func applyTransform(transform types.TransformT, k8sResources []types.K8sResourceT) ([]types.K8sResourceT, error) {
	log.Trace("start applyTransform")
	defer log.Trace("end applyTransform")
	filteredIdxs, err := filterK8sResources(transform, k8sResources)
	if err != nil {
		return k8sResources, err
	}
	for _, filteredIdx := range filteredIdxs {
		k8sResource := k8sResources[filteredIdx]
		transformedK8sResource, err := transform.Transform(k8sResource)
		if err != nil {
			return k8sResources, err
		}
		k8sResources[filteredIdx] = transformedK8sResource
	}
	return k8sResources, nil
}

func filterK8sResources(transform types.TransformT, k8sResources []types.K8sResourceT) ([]int, error) {
	log.Trace("start filterK8sResources")
	defer log.Trace("end filterK8sResources")
	idxs := []int{}
	for i, k8sResource := range k8sResources {
		ok, err := transform.Filter(k8sResource)
		if err != nil {
			return idxs, err
		}
		if ok {
			idxs = append(idxs, i)
		}
	}
	return idxs, nil
}
