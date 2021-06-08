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
	irtypes "github.com/konveyor/move2kube/types/ir"
	log "github.com/sirupsen/logrus"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// ServiceAccount handles all objects like a service account.
type ServiceAccount struct {
}

// getSupportedKinds returns the kinds that this type supports.
func (*ServiceAccount) getSupportedKinds() []string {
	return []string{rbacv1.ServiceAccountKind}
}

// createNewResources creates the runtime objects from the intermediate representation.
func (sa *ServiceAccount) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}
	if common.IsStringPresent(supportedKinds, rbacv1.ServiceAccountKind) {
		irresources := ir.ServiceAccounts
		for _, irresource := range irresources {
			objs = append(objs, sa.createNewResource(irresource))
		}
	} else {
		log.Errorf("Could not find a valid resource type in cluster to create a service account.")
	}
	return objs
}

func (*ServiceAccount) createNewResource(irserviceaccount irtypes.ServiceAccount) *core.ServiceAccount {
	serviceAccount := new(core.ServiceAccount)
	serviceAccount.TypeMeta = metav1.TypeMeta{
		Kind:       rbacv1.ServiceAccountKind,
		APIVersion: core.SchemeGroupVersion.String(),
	}
	serviceAccount.ObjectMeta = metav1.ObjectMeta{Name: irserviceaccount.Name}
	for _, secretName := range irserviceaccount.SecretNames {
		serviceAccount.Secrets = append(serviceAccount.Secrets, core.ObjectReference{Name: secretName})
	}
	return serviceAccount
}

// convertToClusterSupportedKinds converts the object to supported types if possible.
func (sa *ServiceAccount) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR) ([]runtime.Object, bool) {
	if common.IsStringPresent(sa.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
