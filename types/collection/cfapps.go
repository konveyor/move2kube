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

package collection

import (
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/types"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
)

// CfAppsMetadataKind defines kind of cf runtime instance apps file
const CfAppsMetadataKind types.Kind = "CfApps"

// CfApps defines definition of cf runtime instance apps file
type CfApps struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             CfAppsSpec `yaml:"spec,omitempty"`
}

// CfApp defines CfApp information
type CfApp struct {
	Application cfclient.App    `yaml:"application"`
	Environment cfclient.AppEnv `yaml:"environment"`
}

// CfAppsSpec stores the data
type CfAppsSpec struct {
	CfApps []CfApp `yaml:"applications"`
}

// NewCfApps creates a new instance of CfInstanceApps
func NewCfApps() CfApps {
	return CfApps{
		TypeMeta: types.TypeMeta{
			Kind:       string(CfAppsMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}

// FormatMapsWithInterface stringifies interfaces in cloud foundry data
func FormatMapsWithInterface(cfAppInstances CfApps) CfApps {
	for index, app := range cfAppInstances.Spec.CfApps {
		app.Application.DockerCredentialsJSON = common.StringifyMap(app.Application.DockerCredentialsJSON)
		app.Application.Environment = common.StringifyMap(app.Application.Environment)
		app.Environment.Environment = common.StringifyMap(app.Environment.Environment)
		app.Environment.ApplicationEnv = common.StringifyMap(app.Environment.ApplicationEnv)
		app.Environment.RunningEnv = common.StringifyMap(app.Environment.RunningEnv)
		app.Environment.StagingEnv = common.StringifyMap(app.Environment.StagingEnv)
		app.Environment.SystemEnv = common.StringifyMap(app.Environment.SystemEnv)
		cfAppInstances.Spec.CfApps[index] = app
	}
	return cfAppInstances
}
