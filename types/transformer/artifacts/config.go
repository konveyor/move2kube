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
	"github.com/konveyor/move2kube-wasm/types/ir"
	"reflect"

	"github.com/konveyor/move2kube-wasm/common"
	collecttypes "github.com/konveyor/move2kube-wasm/types/collection"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
)

var (
	// ConfigTypes stores the various config types
	ConfigTypes map[string]reflect.Type
)

func init() {
	configObjs := []transformertypes.Config{
		new(ir.IR),
		new(NewImages),
		new(MavenConfig),
		new(GradleConfig),
		new(SpringBootConfig),
		new(ContainerizationOptionsConfig),
		new(collecttypes.ClusterMetadata),
	}
	ConfigTypes = common.GetTypesMap(configObjs)
}
