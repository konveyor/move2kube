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
 */package irpreprocessor

import (
	"github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/sirupsen/logrus"
)

const statefulSetKind = "StatefulSet"

type statefulsetPreprocessor struct {
}

func (sp statefulsetPreprocessor) preprocess(ir irtypes.IR, targetCluster collection.ClusterMetadata) (irtypes.IR, error) {
	if targetCluster.Spec.GetSupportedVersions(statefulSetKind) == nil {
		logrus.Debug("StatefulSets not supported by target cluster.\n")
		return ir, nil
	}

	for k, scObj := range ir.Services {
		isStateful := commonqa.Stateful(scObj.Name)
		scObj.StatefulSet = isStateful
		ir.Services[k] = scObj
	}

	return ir, nil
}