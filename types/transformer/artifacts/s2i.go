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

// S2IMetadataArtifactType is the name of the S2I artifact type
const S2IMetadataArtifactType transformertypes.ArtifactType = "S2IMetadata"

// S2IMetadataConfigType is the name of the S2I config type
const S2IMetadataConfigType transformertypes.ConfigType = "S2IMetadata"

// ProjectPath will be used as context

// S2IMetadataConfig stores the configurations related to S2I
type S2IMetadataConfig struct {
	S2IBuilder string `json:"S2IBuilder" yaml:"S2IBuilder"`
	ImageName  string `json:"ImageName,omitempty" yaml:"ImageName,omitempty"`
}
