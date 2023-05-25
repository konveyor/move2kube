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

package collector

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/konveyor/move2kube/types"
	"github.com/sirupsen/logrus"
)

// CfAppsMetadataKind defines kind of cf runtime instance apps file
const CfAppsMetadataKind types.Kind = "CfApps"

// CfApps defines definition of cf runtime instance apps file
type CfApps struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             CfAppsSpec `yaml:"spec,omitempty"`
}

// CfAppsSpec stores the data
type CfAppsSpec struct {
	CfApps []CfApp `yaml:"applications"`
}

// CfApp defines CfApp and environment information
type CfApp struct {
	Application App             `yaml:"application"`
	Environment cfclient.AppEnv `yaml:"environment"`
}

// App defines CF application information
type App struct {
	Guid                     string                     `json:"guid"`
	CreatedAt                string                     `json:"created_at"`
	UpdatedAt                string                     `json:"updated_at"`
	Name                     string                     `json:"name"`
	Memory                   int                        `json:"memory"`
	Instances                int                        `json:"instances"`
	DiskQuota                int                        `json:"disk_quota"`
	SpaceGuid                string                     `json:"space_guid"`
	StackGuid                string                     `json:"stack_guid"`
	State                    string                     `json:"state"`
	PackageState             string                     `json:"package_state"`
	Command                  string                     `json:"command"`
	Buildpack                string                     `json:"buildpack"`
	DetectedBuildpack        string                     `json:"detected_buildpack"`
	DetectedBuildpackGuid    string                     `json:"detected_buildpack_guid"`
	HealthCheckHttpEndpoint  string                     `json:"health_check_http_endpoint"`
	HealthCheckType          string                     `json:"health_check_type"`
	HealthCheckTimeout       int                        `json:"health_check_timeout"`
	Diego                    bool                       `json:"diego"`
	EnableSSH                bool                       `json:"enable_ssh"`
	DetectedStartCommand     string                     `json:"detected_start_command"`
	DockerImage              string                     `json:"docker_image"`
	DockerCredentialsJSON    map[string]interface{}     `json:"docker_credentials_json"`
	DockerCredentials        cfclient.DockerCredentials `json:"docker_credentials"`
	Environment              map[string]interface{}     `json:"environment_json"`
	StagingFailedReason      string                     `json:"staging_failed_reason"`
	StagingFailedDescription string                     `json:"staging_failed_description"`
	Ports                    []int                      `json:"ports"`
	SpaceURL                 string                     `json:"space_url"`
	SpaceData                cfclient.SpaceResource     `json:"space"`
	PackageUpdatedAt         string                     `json:"package_updated_at"`
	c                        *cfclient.Client
}

// getClient returns a new client for the given cf home directory
func getClient(cfHomeDir string) (*cfclient.Client, error) {
	var cfClientConfig *cfclient.Config
	var err error
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logrus.Errorf("Error while getting current user's home directory: %s", err)
		return nil, err
	}
	if cfHomeDir == "" {
		cfClientConfig, err = cfclient.NewConfigFromCF()
	} else {
		cfClientConfig, err = cfclient.NewConfigFromCFHome(filepath.Join(homeDir, cfHomeDir))
	}
	if err != nil {
		logrus.Debugf("Unable to get the cf config: %s", err)
		return nil, err
	}
	client, err := cfclient.NewClient(cfClientConfig)
	if err != nil {
		if cfHomeDir == "" {
			logrus.Debugf("Failed to create a new client using the config.json in .cf directory.")
		} else {
			logrus.Debugf("Failed to create a new client using the config.json in %s directory.", filepath.Join(homeDir, cfHomeDir, ".cf"))
		}
	}
	return client, err
}

// getCfClient returns a new cf client
func getCfClient() (*cfclient.Client, error) {
	var client *cfclient.Client
	var err error
	cfHomeDirs := [3]string{"", ".ibmcloud", ".bluemix"}
	for _, cfHomeDir := range cfHomeDirs {
		client, err = getClient(cfHomeDir)
		if err == nil {
			break
		}
	}
	return client, err
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

// NewCfApps creates a new instance of CfInstanceApps
func NewCfApps() CfApps {
	return CfApps{
		TypeMeta: types.TypeMeta{
			Kind:       string(CfAppsMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
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
