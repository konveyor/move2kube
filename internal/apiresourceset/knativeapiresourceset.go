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

package apiresourceset

import (
	"github.com/konveyor/move2kube/internal/apiresource"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"k8s.io/apimachinery/pkg/runtime"
	knative "knative.dev/serving/pkg/client/clientset/versioned/scheme"
)

// KnativeAPIResourceSet manages knative related objects
type KnativeAPIResourceSet struct {
}

// GetScheme returns knative scheme object
func (*KnativeAPIResourceSet) GetScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	must(knative.AddToScheme(scheme))
	return scheme
}

// CreateAPIResources converts ir object to runtime objects
func (knativeAPIResourceSet *KnativeAPIResourceSet) CreateAPIResources(oldir irtypes.IR) []runtime.Object {
	ir := irtypes.NewEnhancedIRFromIR(oldir)
	targetObjs := []runtime.Object{}
	ignoredObjs := ir.CachedObjects
	for _, apiResource := range knativeAPIResourceSet.getAPIResources(ir) {
		apiResource.SetClusterContext(ir.TargetClusterSpec)
		resourceIgnoredObjs := apiResource.LoadResources(ir.CachedObjects, ir)
		ignoredObjs = intersection(ignoredObjs, resourceIgnoredObjs)
		resourceObjs := apiResource.GetUpdatedResources(ir)
		targetObjs = append(targetObjs, resourceObjs...)
	}
	targetObjs = append(targetObjs, ignoredObjs...)
	return targetObjs
}

func (knativeAPIResourceSet *KnativeAPIResourceSet) getAPIResources(ir irtypes.EnhancedIR) []apiresource.APIResource {
	apiresources := []apiresource.APIResource{
		{
			Scheme:       knativeAPIResourceSet.GetScheme(),
			IAPIResource: &apiresource.KnativeService{Cluster: ir.TargetClusterSpec},
		},
	}
	return apiresources
}
