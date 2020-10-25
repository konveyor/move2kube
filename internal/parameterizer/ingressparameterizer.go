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
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// ingressParameterizer parameterizes the image names
type ingressParameterizer struct {
}

func (ip ingressParameterizer) parameterize(ir *irtypes.IR) error {
	// Parameterize the host with Release-Name prefix only if helm is enabled.
	if ir.Kubernetes.ArtifactType == plantypes.Helm {
		ir.TargetClusterSpec.Host = "{{ .Release.Name }}-" + ir.TargetClusterSpec.Host
	}

	return nil
}
