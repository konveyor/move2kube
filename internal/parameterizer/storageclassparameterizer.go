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

package parameterize

import (
	irtypes "github.com/konveyor/move2kube/internal/types"
	log "github.com/sirupsen/logrus"
)

// StorageClassParameterizer parameterizes the storage class
type storageClassParameterizer struct {
}

// Parameterize parameterizes the storage class
func (sc storageClassParameterizer) parameterize(ir *irtypes.IR) error {
	scMap := make(map[string][]int)

	for i, storage := range ir.Storages {
		if storage.StorageType == irtypes.PVCKind {
			scName := *storage.PersistentVolumeClaimSpec.StorageClassName
			if indices, ok := scMap[scName]; ok {
				indices = append(indices, i)
				scMap[scName] = indices
			} else {
				scMap[scName] = []int{i}
			}
		}
	}

	if len(scMap) > 1 {
		log.Warnf("Storage class not common across all PVC. Hence, parameterization is skipped.")
		return nil
	}

	paramSC := "{{ .Values.storageclass }}"

	for scName, indexList := range scMap {
		ir.Values.StorageClass = scName
		for _, i := range indexList {
			ir.Storages[i].PersistentVolumeClaimSpec.StorageClassName = &paramSC
		}
	}

	return nil
}
