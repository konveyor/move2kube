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
	"bytes"
	"encoding/json"
	"strings"

	"github.com/konveyor/move2kube/types"
	"github.com/sirupsen/logrus"

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
		app.Application.DockerCredentialsJSON = stringifyMap(app.Application.DockerCredentialsJSON)
		app.Application.Environment = stringifyMap(app.Application.Environment)
		app.Environment.Environment = stringifyMap(app.Environment.Environment)
		app.Environment.ApplicationEnv = stringifyMap(app.Environment.ApplicationEnv)
		app.Environment.RunningEnv = stringifyMap(app.Environment.RunningEnv)
		app.Environment.StagingEnv = stringifyMap(app.Environment.StagingEnv)
		app.Environment.SystemEnv = stringifyMap(app.Environment.SystemEnv)
		cfAppInstances.Spec.CfApps[index] = app
	}
	return cfAppInstances
}

// stringifyMap stringifies the map values
func stringifyMap(inputMap map[string]interface{}) map[string]interface{} {
	for key, value := range inputMap {
		if value == nil {
			inputMap[key] = ""
			continue
		}
		if val, ok := value.(string); ok {
			inputMap[key] = val
			continue
		}
		var b bytes.Buffer
		encoder := json.NewEncoder(&b)
		if err := encoder.Encode(value); err != nil {
			logrus.Error("Unable to unmarshal data to json while converting map interfaces to string")
			continue
		}
		strValue := b.String()
		strValue = strings.TrimSpace(strValue)
		inputMap[key] = strValue
	}
	return inputMap
}
