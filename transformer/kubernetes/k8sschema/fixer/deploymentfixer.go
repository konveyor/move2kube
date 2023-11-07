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

package fixer

import (
	"fmt"

	"github.com/konveyor/move2kube-wasm/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apps "k8s.io/kubernetes/pkg/apis/apps"
)

type deploymentFixer struct {
}

func (f deploymentFixer) getGroupVersionKind() schema.GroupVersionKind {
	return apps.SchemeGroupVersion.WithKind(common.DeploymentKind)
}

func (f deploymentFixer) fix(obj runtime.Object) (runtime.Object, error) {
	d, ok := obj.(*apps.Deployment)
	if !ok {
		return obj, fmt.Errorf("non Matching type. Expected Deployment : Got %T", obj)
	}
	if d.Spec.Selector == nil {
		d.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: d.Spec.Template.Labels,
		}
	}
	obj = d
	return obj, nil
}
