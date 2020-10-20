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

package metadata

import (
	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

// QACacheLoader loads the qa caches
type QACacheLoader struct {
}

// UpdatePlan - output a plan based on the input directory contents
func (i QACacheLoader) UpdatePlan(inputPath string, plan *plantypes.Plan) error {
	files, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		return err
	}
	for _, path := range files {
		cm := new(qatypes.Cache)
		if common.ReadYaml(path, &cm) == nil && cm.Kind == string(qatypes.QACacheKind) {
			relpath, _ := plan.GetRelativePath(path)
			plan.Spec.Inputs.QACaches = append(plan.Spec.Inputs.QACaches, relpath)
		}
	}
	return nil
}

// LoadToIR starts the cache responders
func (i QACacheLoader) LoadToIR(p plantypes.Plan, ir *irtypes.IR) error {
	cachepaths := []string{}
	for i := len(p.Spec.Inputs.QACaches) - 1; i >= 0; i-- {
		cachepaths = append(cachepaths, p.GetFullPath(p.Spec.Inputs.QACaches[i]))
	}
	qaengine.AddCaches(cachepaths)
	return nil
}
