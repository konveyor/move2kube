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

package fixer

import (
	"github.com/konveyor/move2kube/internal/k8sschema"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// fixer can be used to fix K8s resources
type fixer interface {
	getGroupVersionKind() schema.GroupVersionKind
	fix(obj runtime.Object) (runtime.Object, error)
}

var (
	fixers = []fixer{deploymentFixer{}, ingressFixer{}}
)

// Fix fixes kubernetes objects
func Fix(obj runtime.Object) runtime.Object {
	objgv := obj.GetObjectKind().GroupVersionKind().GroupVersion()
	for _, fixer := range fixers {
		fgvk := fixer.getGroupVersionKind()
		if fgvk.Kind == obj.GetObjectKind().GroupVersionKind().Kind {
			logrus.Debugf("Running fixer %T", fixer)
			newobj, err := k8sschema.ConvertToVersion(obj, fgvk.GroupVersion())
			if err != nil {
				logrus.Errorf("Unable to convert to %s for fixer %T", fgvk, fixer)
				continue
			}
			newobj, err = fixer.fix(newobj)
			if err != nil {
				logrus.Errorf("Unable to fix %s using fixer %T : %s", fgvk, fixer, err)
				continue
			}
			newobj, err = k8sschema.ConvertToVersion(newobj, objgv)
			if err != nil {
				logrus.Errorf("Unable to convert back %s : %s", fgvk, err)
				continue
			}
			obj = newobj
		}
	}
	return obj
}
