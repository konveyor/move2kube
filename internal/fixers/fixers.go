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

package fixers

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	scheme *runtime.Scheme = runtime.NewScheme()
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
	for _, fixer := range fixers {
		fgvk := fixer.getGroupVersionKind()
		if fgvk.Kind == obj.GetObjectKind().GroupVersionKind().Kind {
			log.Debugf("Running fixer %T", fixer)
			newobj, err := convertToVersion(obj, fgvk.GroupVersion())
			if err != nil {
				log.Errorf("Unable to convert to %s for fixer %T", fgvk, fixer)
				continue
			}
			obj = newobj
			newobj, err = fixer.fix(obj)
			if err != nil {
				log.Errorf("Unable to fix %s using fixer %T", fgvk, fixer)
				continue
			}
			obj = newobj
		}
	}
	return obj
}

func convertToVersion(obj runtime.Object, dgv schema.GroupVersion) (newobj runtime.Object, err error) {
	if obj.GetObjectKind().GroupVersionKind().GroupVersion() == dgv {
		return obj, nil
	}
	return scheme.ConvertToVersion(obj, dgv)
}
