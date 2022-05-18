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
	"strings"

	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/sirupsen/logrus"
)

// CfServicesCollector collects cf runtime applications
type CfServicesCollector struct {
}

// GetAnnotations returns annotations on which this collector should be invoked
func (c *CfServicesCollector) GetAnnotations() []string {
	annotations := []string{"cf", "cloudfoundry"}
	return annotations
}

// getNewIbmcloudCfClient returns a new ibmcloud cf client
func (c *CfServicesCollector) getNewIbmcloudCfClient(homeDir string) (*cfclient.Client, error) {
	var client *cfclient.Client
	cfclientConfig, err := cfclient.NewConfigFromCFHome(filepath.Join(homeDir, ".ibmcloud/"))
	if err == nil {
		client, err = cfclient.NewClient(cfclientConfig)
		if err != nil {
			logrus.Debugf("The .ibmcloud directory based cf login failed while creating new client. Falling back to the .bluemix directory based login")
			cfclientConfig, err = cfclient.NewConfigFromCFHome(filepath.Join(homeDir, ".bluemix/"))
			if err == nil {
				client, err = cfclient.NewClient(cfclientConfig)
				if err != nil {
					logrus.Debugf("The .bluemix directory based cf login failed while creating new client.")
				}
			} else {
				logrus.Debugf("Unable to get cf config from .bluemix directory : %s", err)
			}
		}
	} else {
		logrus.Debugf("Unable to get cf config from .ibmcloud directory : %s", err)
		logrus.Debugf("The .ibmcloud directory based cf login failed. Falling back to the .bluemix directory based login")
		cfclientConfig, err = cfclient.NewConfigFromCFHome(filepath.Join(homeDir, ".bluemix/"))
		if err == nil {
			client, err = cfclient.NewClient(cfclientConfig)
			if err != nil {
				logrus.Debugf("The .bluemix directory based cf login failed while creating new client.")
			}
		} else {
			logrus.Debugf("Unable to get cf config from .bluemix directory : %s", err)
		}
	}
	return client, err
}

// getNewClient returns a new client
func (c *CfServicesCollector) getNewClient(homeDir string) (*cfclient.Client, error) {
	var client *cfclient.Client
	cfclientConfig, err := cfclient.NewConfigFromCF()
	if err == nil {
		client, err = cfclient.NewClient(cfclientConfig)
		if err != nil {
			logrus.Debugf("The .cf directory based cf login failed while creating new client. Falling back to .ibmcloud directory based login")
			client, err = c.getNewIbmcloudCfClient(homeDir)
		}
	} else {
		logrus.Debugf("Error while getting cf config : %s", err)
		logrus.Debugf("The .cf directory based cf login failed. Falling back to the .ibmcloud directory based login")
		client, err = c.getNewIbmcloudCfClient(homeDir)
	}
	return client, err
}

//Collect gets the cf service metadata by querying the cf app. Assumes that the authentication with cluster is already done.
func (c *CfServicesCollector) Collect(inputPath string, outputPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logrus.Errorf("Error while getting current user's home directory: %s", err)
		return err
	}
	client, err := c.getNewClient(homeDir)
	if err != nil {
		logrus.Errorf("Unable to connect to cf client : %s", err)
		return err
	}
	cfInfo, err := client.GetInfo()
	if err != nil {
		logrus.Errorf("Unable to get info of cf instance : %s", err)
	}
	services, err := client.ListServices()
	if err != nil {
		logrus.Errorf("Unable to get list of cf apps : %s", err)
	}
	outputPath = filepath.Join(outputPath, "cf")
	err = os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		logrus.Errorf("Unable to create outputPath %s : %s", outputPath, err)
	}
	cfservices := collecttypes.NewCfServices()
	cfservices.Name = common.NormalizeForMetadataName(strings.TrimSpace(cfInfo.Name))
	cfservices.Spec.CfServices = services
	fileName := "cfservices-" + cfservices.Name
	if fileName != "" {
		outputPath = filepath.Join(outputPath, common.NormalizeForFilename(fileName)+".yaml")
		err = common.WriteYaml(outputPath, cfservices)
		if err != nil {
			logrus.Errorf("Unable to write collect output : %s", err)
		}
		return err
	}

	return nil
}
