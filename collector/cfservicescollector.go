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

	"github.com/konveyor/move2kube-wasm/common"
	collecttypes "github.com/konveyor/move2kube-wasm/types/collection"

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

// Collect gets the cf service metadata by querying the cf app. Assumes that the authentication with cluster is already done.
func (c *CfServicesCollector) Collect(inputPath string, outputPath string) error {
	client, err := getCfClient()
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
