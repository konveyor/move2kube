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

// CNBDetectedServiceArtifactType is the name of the CNB detected service artifact
const CNBDetectedServiceArtifactType transformertypes.ArtifactType = "CNBDetectedService"

// CNBMetadataArtifactType is the name of the CNB artifact type
const CNBMetadataArtifactType transformertypes.ArtifactType = "CNBMetadata"

// CNBMetadataConfigType is the name of the CNB config type
const CNBMetadataConfigType transformertypes.ConfigType = "CNBMetadata"

// ProjectPath will be used as context

// CNBMetadataConfig stores the configurations related to CNB
type CNBMetadataConfig struct {
	CNBBuilder string `json:"CNBBuilder" yaml:"CNBBuilder"`
	ImageName  string `json:"ImageName,omitempty" yaml:"ImageName,omitempty"`
}
