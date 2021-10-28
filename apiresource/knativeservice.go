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

package apiresource

import (
	"github.com/konveyor/move2kube/k8sschema"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/kubernetes/pkg/apis/core"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"
)

const (
	// knativeServiceKind defines the KNative service kind
	knativeServiceKind string = "Service"
)

// KnativeService handles the Knative service object
type KnativeService struct {
}

// createNewResources creates new knative services for IR
func (d *KnativeService) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}

	for _, service := range ir.Services {
		podSpec := service.PodSpec
		podSpec.RestartPolicy = core.RestartPolicyAlways
		knativeservice := &knativev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       knativeServiceKind,
				APIVersion: knativev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:        service.Name,
				Labels:      getServiceLabels(service.Name),
				Annotations: getAnnotations(service),
			},
			Spec: knativev1.ServiceSpec{
				ConfigurationSpec: knativev1.ConfigurationSpec{
					Template: knativev1.RevisionTemplateSpec{
						Spec: knativev1.RevisionSpec{
							PodSpec: k8sschema.ConvertToV1PodSpec(&podSpec),
						},
					},
				},
			},
		}
		objs = append(objs, knativeservice)
	}
	return objs
}

// convertToClusterSupportedKinds converts kinds to cluster supported kinds
func (d *KnativeService) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if d1, ok := obj.(*knativev1.Service); ok {
		return []runtime.Object{d1}, true
	}
	return nil, false
}

// getSupportedKinds returns kinds supported by Knative service
func (d *KnativeService) getSupportedKinds() []string {
	return []string{knativeServiceKind}
}
