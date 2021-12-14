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

	"github.com/konveyor/move2kube/common"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	networking "k8s.io/kubernetes/pkg/apis/networking"
)

type ingressFixer struct {
}

func (f ingressFixer) getGroupVersionKind() schema.GroupVersionKind {
	return networking.SchemeGroupVersion.WithKind(common.IngressKind)
}

func (f ingressFixer) fix(obj runtime.Object) (runtime.Object, error) {
	ptf := networking.PathTypePrefix
	i, ok := obj.(*networking.Ingress)
	if !ok {
		return obj, fmt.Errorf("non Matching type. Expected Ingress : Got %T", obj)
	}
	for ri, r := range i.Spec.Rules {
		for pi, p := range r.HTTP.Paths {
			if p.PathType == nil {
				i.Spec.Rules[ri].HTTP.Paths[pi].PathType = &ptf
			}
		}
	}
	obj = i
	return obj, nil
}
