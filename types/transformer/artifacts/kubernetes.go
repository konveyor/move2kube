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

package artifacts

import (
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
)

const (
	// KubernetesYamlsArtifactType is the name of the Kubernetes artifact type
	KubernetesYamlsArtifactType transformertypes.ArtifactType = "KubernetesYamls"

	// KubernetesYamlsInSourceArtifactType is the name of the Kubernetes artifact type
	KubernetesYamlsInSourceArtifactType transformertypes.ArtifactType = "KubernetesYamlsInSource"

	// KubernetesOrgYamlsInSourceArtifactType is the name of the Kubernetes original yamls artifact type
	KubernetesOrgYamlsInSourceArtifactType transformertypes.ArtifactType = "KubernetesOrgYamlsInSource"

	// KubernetesYamlsPathType is points to the kubernetes Yamls
	KubernetesYamlsPathType transformertypes.PathType = "KubernetesYamls"
)
