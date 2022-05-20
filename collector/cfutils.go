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
	"os"
	"path/filepath"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/sirupsen/logrus"
)

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
