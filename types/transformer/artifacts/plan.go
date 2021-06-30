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
	"github.com/konveyor/move2kube/types/collection"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

// PlanConfigType stores the config type name for plan info
const PlanConfigType transformertypes.ConfigType = "Plan"

// PlanConfig stores the Plan configuration
type PlanConfig struct {
	PlanName      string                     `yaml:"planName"`
	TargetCluster collection.ClusterMetadata `yaml:"cluster"`
}
