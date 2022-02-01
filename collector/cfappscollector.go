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

	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"

	cfclient "github.com/cloudfoundry-community/go-cfclient"
	"github.com/sirupsen/logrus"
)

// CfAppsCollector collects cf runtime applications
type CfAppsCollector struct {
}

// GetAnnotations returns annotations on which this collector should be invoked
func (c *CfAppsCollector) GetAnnotations() []string {
	annotations := []string{"cf", "cloudfoundry"}
	return annotations
}

//Collect gets the cf app metadata by querying the cf app. Assumes that the authentication with cluster is already done.
func (c *CfAppsCollector) Collect(inputPath string, outputPath string) error {
	cli, err := cfclient.NewConfigFromCF()
	if err != nil {
		logrus.Errorf("Error while getting cf config : %s", err)
		return err
	}
	client, err := cfclient.NewClient(cli)
	if err != nil {
		logrus.Errorf("Unable to connect to cf client : %s", err)
		return err
	}
	cfInfo, err := client.GetInfo()
	if err != nil {
		logrus.Errorf("Unable to get info of cf instance : %s", err)
	}
	apps, err := client.ListApps()
	if err != nil {
		logrus.Errorf("Unable to get list of cf apps : %s", err)
	}
	outputPath = filepath.Join(outputPath, "cf")
	err = os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		logrus.Errorf("Unable to create outputPath %s : %s", outputPath, err)
	}
	cfinstanceapps := collecttypes.NewCfApps()
	cfinstanceapps.Name = common.NormalizeForMetadataName(cfInfo.Name)
	for _, app := range apps {
		cfapp := collecttypes.CfApp{
			Application: app,
		}
		appEnv, err := client.GetAppEnv(app.Guid)
		if err != nil {
			logrus.Errorf("Unable to get app environment data : %s", err)
		} else {
			cfapp.Environment = appEnv
		}
		cfinstanceapps.Spec.CfApps = append(cfinstanceapps.Spec.CfApps, cfapp)
	}
	cfinstanceapps = collecttypes.FormatMapsWithInterface(cfinstanceapps)
	fileName := "cfapps-" + cfinstanceapps.Name + "-" + cli.ClientID
	if fileName != "" {
		outputPath = filepath.Join(outputPath, common.NormalizeForFilename(fileName)+".yaml")
		err = common.WriteYaml(outputPath, cfinstanceapps)
		if err != nil {
			logrus.Errorf("Unable to write collect output : %s", err)
		}
		return err
	}

	return nil
}
