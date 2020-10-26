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
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

// QACacheLoader loads the qa caches
type QACacheLoader struct {
}

// UpdatePlan - output a plan based on the input directory contents
func (*QACacheLoader) UpdatePlan(inputPath string, plan *plantypes.Plan) error {
	filePaths, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Errorf("Unable to fetch yaml files at path %q Error: %q", inputPath, err)
		return err
	}
	for _, filePath := range filePaths {
		cm := qatypes.Cache{}
		if err := common.ReadYaml(filePath, &cm); err != nil {
			log.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		if cm.Kind != string(qatypes.QACacheKind) { // TODO: should we remove this check?
			continue
		}
		plan.Spec.Inputs.QACaches = append(plan.Spec.Inputs.QACaches, filePath)
	}
	return nil
}

// LoadToIR starts the cache responders
func (*QACacheLoader) LoadToIR(plan plantypes.Plan, ir *irtypes.IR) error {
	cachePaths := []string{}
	for i := len(plan.Spec.Inputs.QACaches) - 1; i >= 0; i-- {
		cachePaths = append(cachePaths, plan.Spec.Inputs.QACaches[i])
	}
	qaengine.AddCaches(cachePaths)
	return nil
}
