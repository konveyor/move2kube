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
	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	roleKind = "Role"
)

// Role handles all objects like a role.
type Role struct {
}

// GetSupportedKinds returns the kinds that this type supports.
func (*Role) GetSupportedKinds() []string {
	return []string{roleKind}
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (r *Role) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	if common.IsStringPresent(supportedKinds, roleKind) {
		irresources := ir.Roles
		for _, irresource := range irresources {
			objs = append(objs, r.createNewResource(irresource))
		}
	} else {
		log.Errorf("Could not find a valid resource type in cluster to create a role.")
	}
	return objs
}

func (*Role) createNewResource(irrole irtypes.Role) *rbacv1.Role {
	role := new(rbacv1.Role)
	role.TypeMeta = metav1.TypeMeta{
		Kind:       roleKind,
		APIVersion: rbacv1.SchemeGroupVersion.String(),
	}
	role.ObjectMeta = metav1.ObjectMeta{Name: irrole.Name}
	rules := []rbacv1.PolicyRule{}
	for _, policyRule := range irrole.PolicyRules {
		rules = append(rules, rbacv1.PolicyRule{APIGroups: policyRule.APIGroups, Resources: policyRule.Resources, Verbs: policyRule.Verbs})
	}
	role.Rules = rules
	return role
}

// ConvertToClusterSupportedKinds converts the object to supported types if possible.
func (r *Role) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) {
	supKinds := r.GetSupportedKinds()
	for _, supKind := range supKinds {
		if common.IsStringPresent(supportedKinds, supKind) {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}
