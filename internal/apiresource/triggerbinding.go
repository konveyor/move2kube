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

// TriggerBinding handles all objects like a trigger binding.
type TriggerBinding struct {
}

// GetSupportedKinds returns the kinds that this type supports.
func (*TriggerBinding) GetSupportedKinds() []string {
	return []string{string(triggersv1alpha1.NamespacedTriggerBindingKind)}
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (tb *TriggerBinding) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	if common.IsStringPresent(supportedKinds, string(triggersv1alpha1.NamespacedTriggerBindingKind)) {
		irresources := ir.TektonResources.TriggerBindings
		for _, irresource := range irresources {
			objs = append(objs, tb.createNewResource(irresource))
		}
	} else {
		log.Errorf("Could not find a valid resource type in cluster to create a trigger binding.")
	}
	return objs
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (*TriggerBinding) createNewResource(irtriggerbinding tekton.TriggerBinding) *triggersv1alpha1.TriggerBinding {
	triggerBinding := new(triggersv1alpha1.TriggerBinding)
	triggerBinding.TypeMeta = metav1.TypeMeta{
		Kind:       string(triggersv1alpha1.NamespacedTriggerBindingKind),
		APIVersion: triggersv1alpha1.SchemeGroupVersion.String(),
	}
	triggerBinding.ObjectMeta = metav1.ObjectMeta{Name: irtriggerbinding.Name}
	return triggerBinding
}

// ConvertToClusterSupportedKinds converts the object to supported types if possible.
func (tb *TriggerBinding) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) {
	supKinds := tb.GetSupportedKinds()
	for _, supKind := range supKinds {
		if common.IsStringPresent(supportedKinds, supKind) {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}
