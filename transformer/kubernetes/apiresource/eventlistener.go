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
	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	triggersv1alpha1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	eventListenerKind = "EventListener"
)

// EventListener handles all objects like an event listener.
type EventListener struct {
}

// getSupportedKinds returns the kinds that this type supports.
func (*EventListener) getSupportedKinds() []string {
	return []string{eventListenerKind}
}

// createNewResources creates the runtime objects from the intermediate representation.
func (el *EventListener) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	// Since tekton is an extension, the tekton resources are put in a separate folder from the main application.
	// We ignore supported kinds because these resources are optional and it's upto the user to install the extension if they need it.
	irresources := ir.TektonResources.EventListeners
	for _, irresource := range irresources {
		objs = append(objs, el.createNewResource(irresource, targetCluster))
	}
	return objs
}

// createNewResources creates the runtime objects from the intermediate representation.
func (el *EventListener) createNewResource(ireventlistener irtypes.EventListener, targetCluster collecttypes.ClusterMetadata) *triggersv1alpha1.EventListener {
	eventListener := new(triggersv1alpha1.EventListener)
	eventListener.TypeMeta = metav1.TypeMeta{
		Kind:       eventListenerKind,
		APIVersion: triggersv1alpha1.SchemeGroupVersion.String(),
	}
	eventListener.ObjectMeta = metav1.ObjectMeta{Name: ireventlistener.Name}
	eventListener.Spec = triggersv1alpha1.EventListenerSpec{
		ServiceAccountName: ireventlistener.ServiceAccountName,
		Triggers: []triggersv1alpha1.EventListenerTrigger{
			{
				Bindings: []*triggersv1alpha1.EventListenerBinding{
					{Ref: ireventlistener.TriggerBindingName},
				},
				Template: &triggersv1alpha1.EventListenerTemplate{
					Ref: &ireventlistener.TriggerTemplateName,
				},
			},
		},
	}
	return eventListener
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (el *EventListener) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsPresent(el.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
