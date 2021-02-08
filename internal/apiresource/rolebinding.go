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
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	rbac "k8s.io/kubernetes/pkg/apis/rbac"
)

const (
	roleBindingKind = "RoleBinding"
)

// RoleBinding handles all objects like a role binding.
type RoleBinding struct {
}

// getSupportedKinds returns the kinds that this type supports.
func (*RoleBinding) getSupportedKinds() []string {
	return []string{roleBindingKind}
}

// createNewResources creates the runtime objects from the intermediate representation.
func (rb *RoleBinding) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	if common.IsStringPresent(supportedKinds, roleBindingKind) {
		irresources := ir.RoleBindings
		for _, irresource := range irresources {
			objs = append(objs, rb.createNewResource(irresource))
		}
	} else {
		log.Errorf("Could not find a valid resource type in cluster to create a role binding.")
	}
	return objs
}

func (*RoleBinding) createNewResource(irrolebinding irtypes.RoleBinding) *rbac.RoleBinding {
	roleBinding := new(rbac.RoleBinding)
	roleBinding.TypeMeta = metav1.TypeMeta{
		Kind:       roleBindingKind,
		APIVersion: rbac.SchemeGroupVersion.String(),
	}
	roleBinding.ObjectMeta = metav1.ObjectMeta{Name: irrolebinding.Name}
	roleBinding.Subjects = []rbac.Subject{
		{Kind: rbac.ServiceAccountKind, Name: irrolebinding.ServiceAccountName},
	}
	roleBinding.RoleRef = rbac.RoleRef{APIGroup: rbac.SchemeGroupVersion.Group, Kind: roleKind, Name: irrolebinding.RoleName}

	return roleBinding
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (rb *RoleBinding) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR) ([]runtime.Object, bool) {
	if common.IsStringPresent(rb.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
