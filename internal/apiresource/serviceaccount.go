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
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ServiceAccount handles all objects like a service account.
type ServiceAccount struct {
}

// GetSupportedKinds returns the kinds that this type supports.
func (*ServiceAccount) GetSupportedKinds() []string {
	return []string{rbacv1.ServiceAccountKind}
}

// CreateNewResources creates the runtime objects from the intermediate representation.
func (sa *ServiceAccount) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
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

func (*ServiceAccount) createNewResource(irserviceaccount irtypes.ServiceAccount) *corev1.ServiceAccount {
	serviceAccount := new(corev1.ServiceAccount)
	serviceAccount.TypeMeta = metav1.TypeMeta{
		Kind:       rbacv1.ServiceAccountKind,
		APIVersion: corev1.SchemeGroupVersion.String(),
	}
	serviceAccount.ObjectMeta = metav1.ObjectMeta{Name: irserviceaccount.Name}
	for _, secretName := range irserviceaccount.SecretNames {
		serviceAccount.Secrets = append(serviceAccount.Secrets, corev1.ObjectReference{Name: secretName})
	}
	return serviceAccount
}

// ConvertToClusterSupportedKinds converts the object to supported types if possible.
func (sa *ServiceAccount) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) {
	supKinds := sa.GetSupportedKinds()
	for _, supKind := range supKinds {
		if common.IsStringPresent(supportedKinds, supKind) {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}
