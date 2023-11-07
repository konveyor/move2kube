/*
 *  Copyright IBM Corporation 2020, 2021
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

package operator

import (
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
)

// OperatorsToInitializeArtifactType is the type for an artifact that containing the output of TCA (operator names and URLs).
const OperatorsToInitializeArtifactType transformertypes.ArtifactType = "OperatorsToInitialize"

// OperatorsToInitializeArtifactConfigType is the type for OperatorArtifactConfig config used by OperatorArtifactType artifacts.
const OperatorsToInitializeArtifactConfigType transformertypes.ConfigType = "OperatorsToInitializeConfig"

// OperatorArtifactConfig represents the data inside a OperatorArtifactType artifact.
type OperatorArtifactConfig struct {
	Operators map[string]OperatorArtifactConfigOp `json:"operators,omitempty"`
}

// InstallPlanApprovalType reprents an OLM automatic or manual install plan.
type InstallPlanApprovalType string

const (
	// AutomaticApproval means the operator installation is approved automatically.
	AutomaticApproval InstallPlanApprovalType = "Automatic"
	// ManualApproval means the operator installation has to be approved manually.
	ManualApproval InstallPlanApprovalType = "Manual"
)

// IsValid returns true if the install plan is valid.
func (i InstallPlanApprovalType) IsValid() bool {
	return i == AutomaticApproval || i == ManualApproval
}

// OperatorArtifactConfigOp contains all the info needed to install the operator with OLM.
type OperatorArtifactConfigOp struct {
	Url                 string                  `json:"url,omitempty"`
	OperatorName        string                  `json:"operatorName"`
	CatalogSource       string                  `json:"catalogSource,omitempty"`
	CatalogChannel      string                  `json:"catalogChannel,omitempty"`
	InstallPlanApproval InstallPlanApprovalType `json:"installPlanApproval,omitempty"`
}
