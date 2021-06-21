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
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	rbac "k8s.io/kubernetes/pkg/apis/rbac"
)

const (
	roleKind = "Role"
)

// Role handles all objects like a role.
type Role struct {
}

// getSupportedKinds returns the kinds that this type supports.
func (*Role) getSupportedKinds() []string {
	return []string{roleKind}
}

// createNewResources creates the runtime objects from the intermediate representation.
func (r *Role) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	if common.IsStringPresent(supportedKinds, roleKind) {
		irresources := ir.Roles
		for _, irresource := range irresources {
			objs = append(objs, r.createNewResource(irresource))
		}
	} else {
		logrus.Errorf("Could not find a valid resource type in cluster to create a role.")
	}
	return objs
}

func (*Role) createNewResource(irrole irtypes.Role) *rbac.Role {
	role := new(rbac.Role)
	role.TypeMeta = metav1.TypeMeta{
		Kind:       roleKind,
		APIVersion: rbac.SchemeGroupVersion.String(),
	}
	role.ObjectMeta = metav1.ObjectMeta{Name: irrole.Name}
	rules := []rbac.PolicyRule{}
	for _, policyRule := range irrole.PolicyRules {
		rules = append(rules, rbac.PolicyRule{APIGroups: policyRule.APIGroups, Resources: policyRule.Resources, Verbs: policyRule.Verbs})
	}
	role.Rules = rules
	return role
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (r *Role) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsStringPresent(r.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
