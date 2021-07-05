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
	// DockerImageBuildScriptArtifactType represents the image push script artifact type
	DockerImageBuildScriptArtifactType transformertypes.ArtifactType = "DockerImageBuildScript"
)

const (
	// DockerImageBuildShScriptPathType represents the image push script path type
	DockerImageBuildShScriptPathType transformertypes.PathType = "DockerImageBuildShScript"
	// DockerImageBuildBatScriptPathType represents the image push script path type
	DockerImageBuildBatScriptPathType transformertypes.PathType = "DockerImageBuildBatScript"
)
