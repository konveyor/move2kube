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
	"github.com/konveyor/move2kube-wasm/types/collection"
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// imagePullPolicyOptimizer sets the pull policy to be always
type imagePullPolicyPreprocessor struct {
}

func (ep imagePullPolicyPreprocessor) preprocess(ir irtypes.IR, targetCluster collection.ClusterMetadata) (irtypes.IR, error) {
	for k, scObj := range ir.Services {
		for i := range scObj.Containers {
			scObj.Containers[i].ImagePullPolicy = core.PullAlways
		}
		ir.Services[k] = scObj
	}

	return ir, nil
}
