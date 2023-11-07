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
	// ContainerImageBuildScriptArtifactType represents the image build script artifact type
	ContainerImageBuildScriptArtifactType transformertypes.ArtifactType = "ContainerImageBuildScript"
)

const (
	// ContainerImageBuildShScriptPathType represents the image build script path type
	ContainerImageBuildShScriptPathType transformertypes.PathType = "ContainerImageBuildShScript"
	// ContainerImageBuildBatScriptPathType represents the image build script path type
	ContainerImageBuildBatScriptPathType transformertypes.PathType = "ContainerImageBuildBatScript"
	// ContainerImageBuildShScriptContextPathType represents the image build script path type
	ContainerImageBuildShScriptContextPathType transformertypes.PathType = "ContainerImageBuildShScriptContextScript"
	// ContainerImageBuildBatScriptContextPathType represents the image build script path type
	ContainerImageBuildBatScriptContextPathType transformertypes.PathType = "ContainerImageBuildBatScriptContextScript"
)
