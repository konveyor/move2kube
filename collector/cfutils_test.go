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

package collector_test

import (
	"testing"

	collector "github.com/konveyor/move2kube/collector"

	"github.com/konveyor/move2kube/types"
)

func TestNewCfApps(t *testing.T) {
	cfapps := collector.NewCfApps()
	if cfapps.Kind != string(collector.CfAppsMetadataKind) || cfapps.APIVersion != types.SchemeGroupVersion.String() {
		t.Fatal("Failed to initialize CfApps properly.")
	}
}
