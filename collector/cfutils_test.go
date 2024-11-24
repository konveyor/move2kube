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

	"github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/konveyor/move2kube/types"
	"github.com/stretchr/testify/assert"

	collector "github.com/konveyor/move2kube/collector"
)

func TestNewCfApps(t *testing.T) {
	cfapps := collector.NewCfApps()
	assert.Equal(t, string(collector.CfAppsMetadataKind), cfapps.Kind, "Failed to initialize CfApps kind properly.")
	assert.Equal(t, types.SchemeGroupVersion.String(), cfapps.APIVersion, "Failed to initialize CfApps APIVersion properly.")
}

func TestFormatMapsWithInterface(t *testing.T) {
	app := collector.CfApp{
		Application: collector.App{
			DockerCredentialsJSON: map[string]interface{}{"key1": "value1"},
			Environment:           map[string]interface{}{"env1": "value1"},
		},
		Environment: cfclient.AppEnv{
			Environment:    map[string]interface{}{"env2": "value2"},
			ApplicationEnv: map[string]interface{}{"appenv1": "value1"},
			RunningEnv:     map[string]interface{}{"runenv1": "value1"},
			StagingEnv:     map[string]interface{}{"stageenv1": "value1"},
			SystemEnv:      map[string]interface{}{"sysenv1": "value1"},
		},
	}
	cfApps := collector.CfApps{
		Spec: collector.CfAppsSpec{
			CfApps: []collector.CfApp{app},
		},
	}

	cfApps = collector.FormatMapsWithInterface(cfApps)
	expectedApp := collector.CfApp{
		Application: collector.App{
			DockerCredentialsJSON: map[string]interface{}{"key1": "value1"},
			Environment:           map[string]interface{}{"env1": "value1"},
		},
		Environment: cfclient.AppEnv{
			Environment:    map[string]interface{}{"env2": "value2"},
			ApplicationEnv: map[string]interface{}{"appenv1": "value1"},
			RunningEnv:     map[string]interface{}{"runenv1": "value1"},
			StagingEnv:     map[string]interface{}{"stageenv1": "value1"},
			SystemEnv:      map[string]interface{}{"sysenv1": "value1"},
		},
	}
	expectedCfApps := collector.CfApps{
		Spec: collector.CfAppsSpec{
			CfApps: []collector.CfApp{expectedApp},
		},
	}
	assert.Equal(t, expectedCfApps, cfApps, "Failed to format maps with interface correctly.")
}
