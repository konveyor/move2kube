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

package apiresource

import (
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/internal/types/tekton"
	log "github.com/sirupsen/logrus"
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

// GetSupportedKinds returns the kinds that this type supports.
func (*EventListener) GetSupportedKinds() []string {
	return []string{eventListenerKind}
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (el *EventListener) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	if common.IsStringPresent(supportedKinds, eventListenerKind) {
		irresources := ir.TektonResources.EventListeners
		for _, irresource := range irresources {
			objs = append(objs, el.createNewResource(irresource))
		}
	} else {
		log.Errorf("Could not find a valid resource type in cluster to create an event listener.")
	}
	return objs
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (el *EventListener) createNewResource(ireventlistener tekton.EventListener) *triggersv1alpha1.EventListener {
	eventListener := new(triggersv1alpha1.EventListener)
	eventListener.TypeMeta = metav1.TypeMeta{
		Kind:       eventListenerKind,
		APIVersion: triggersv1alpha1.SchemeGroupVersion.String(),
	}
	eventListener.ObjectMeta = metav1.ObjectMeta{Name: ireventlistener.Name}
	eventListener.Spec = triggersv1alpha1.EventListenerSpec{
		ServiceAccountName: ireventlistener.ServiceAccountName,
		Triggers: []triggersv1alpha1.EventListenerTrigger{
			triggersv1alpha1.EventListenerTrigger{
				Bindings: []*triggersv1alpha1.EventListenerBinding{
					&triggersv1alpha1.EventListenerBinding{Ref: ireventlistener.TriggerBindingName},
				},
				Template: &triggersv1alpha1.EventListenerTemplate{Name: ireventlistener.TriggerTemplateName},
			},
		},
	}
	return eventListener
}

// ConvertToClusterSupportedKinds converts the object to supported types if possible.
func (el *EventListener) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) {
	supKinds := el.GetSupportedKinds()
	for _, supKind := range supKinds {
		if common.IsStringPresent(supportedKinds, supKind) {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}
