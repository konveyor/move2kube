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
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

const (
	// ContainerImagesBuildScriptArtifactType represents the image push script artifact type
	ContainerImagesBuildScriptArtifactType transformertypes.ArtifactType = "ContainerImagesBuildScript"
)

const (
	// ContainerImagesBuildShScriptPathType represents the image push script path type
	ContainerImagesBuildShScriptPathType transformertypes.PathType = "ContainerImagesBuildShScript"
	// ContainerImagesBuildBatScriptPathType represents the image push script path type
	ContainerImagesBuildBatScriptPathType transformertypes.PathType = "ContainerImagesBuildBatScript"
)
