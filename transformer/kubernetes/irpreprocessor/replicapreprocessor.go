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

package irpreprocessor

import (
	"github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/qaengine/commonqa"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// replicaOptimizer sets the minimum number of replicas
type replicaPreprocessor struct {
}

const (
	minReplicas int = 2
)

func (ep replicaPreprocessor) preprocess(ir irtypes.IR, targetCluster collection.ClusterMetadata) (irtypes.IR, error) {
	replicaCountStr := commonqa.MinimumReplicaCount(cast.ToString(minReplicas))
	replicaCount, err := cast.ToIntE(replicaCountStr)
	if err != nil {
		logrus.Errorf("Replica count %s is not a number. Reverting to default %d.", replicaCountStr, minReplicas)
		replicaCount = minReplicas
	}
	for k, scObj := range ir.Services {
		if scObj.Replicas < replicaCount {
			scObj.Replicas = replicaCount
		}
		ir.Services[k] = scObj
	}

	return ir, nil
}
