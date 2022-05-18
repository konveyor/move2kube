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
	"path/filepath"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/sirupsen/logrus"
)

// getCfClient returns a new cf client
func getCfClient(homeDir string) (*cfclient.Client, error) {
	var client *cfclient.Client
	var err error
	var cfClientConfig *cfclient.Config
	cfHomeDirs := [3]string{"", ".ibmcloud", ".bluemix"}
	for _, cfHomeDir := range cfHomeDirs {
		cfClientConfig, err = cfclient.NewConfigFromCFHome(filepath.Join(homeDir, cfHomeDir))
		if err != nil {
			if cfHomeDir == "" {
				logrus.Debugf("The .cf directory based cf login failed. Unable to get cf config : %s", err)
			} else {
				logrus.Debugf("The %s directory based cf login failed. Unable to get cf config : %s", cfHomeDir, err)
			}
		} else {
			client, err = cfclient.NewClient(cfClientConfig)
			if err != nil {
				if cfHomeDir == "" {
					logrus.Debugf("The .cf directory based cf login failed while creating new client.")
				} else {
					logrus.Debugf("The %s directory based cf login failed while creating new client.", cfHomeDir)
				}
			} else {
				break
			}
		}
	}
	return client, err
}
