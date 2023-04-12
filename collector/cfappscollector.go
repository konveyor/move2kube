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
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"

	"github.com/sirupsen/logrus"
)

const (
	inlineDepthRelations = "inline-relations-depth"
	depth                = "2"
)

// CfCollectAppsMetadataKind defines kind of cf collect apps file
const CfCollectAppsMetadataKind types.Kind = "CfCollectApps"

// CfCollectApps defines definition of cf collect apps file
type CfCollectApps struct {
	types.TypeMeta `yaml:",inline"`
	Spec           CfCollectAppsSpec `yaml:"spec,omitempty"`
}

// CfCollectAppsSpec stores the app information
type CfCollectAppsSpec struct {
	CfCollectApps []CfCollectApp `yaml:"applications"`
}

// CfCollectApp defines CfCollectApp information
type CfCollectApp struct {
	Name             string `yaml:"name"`
	Guid             string `yaml:"guid,omitempty"`
	OrganizationGuid string `yaml:"organizationguid,omitempty"`
	SpaceGuid        string `yaml:"spaceguid,omitempty"`
}

// CfAppsCollector collects cf runtime applications
type CfAppsCollector struct {
}

// GetAnnotations returns annotations on which this collector should be invoked
func (c *CfAppsCollector) GetAnnotations() []string {
	annotations := []string{"cf", "cloudfoundry"}
	return annotations
}

// collectSelectiveCfApps collects the selected cf apps
func collectSelectiveCfApps(inputPath string, client *cfclient.Client, depth string) []cfclient.App {
	cfCollectApps := []CfCollectApp{}
	var collectApps []cfclient.App
	filePaths, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Errorf("failed to look for yaml files in the directory %s . Error: %q", inputPath, err)
		return collectApps
	}
	for _, filePath := range filePaths {
		cfInstanceCollectApps := CfCollectApps{}
		if err := common.ReadMove2KubeYaml(filePath, &cfInstanceCollectApps); err != nil {
			logrus.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		if cfInstanceCollectApps.Kind != string(CfCollectAppsMetadataKind) {
			logrus.Debugf("%q is not a valid cf collect apps file. Expected kind: %s Actual Kind: %s", filePath, string(CfCollectAppsMetadataKind), cfInstanceCollectApps.Kind)
			continue
		}
		cfCollectApps = append(cfCollectApps, cfInstanceCollectApps.Spec.CfCollectApps...)
	}
	for _, cfCollectApp := range cfCollectApps {
		query := url.Values{}
		query.Add("q", fmt.Sprintf("name:%s", cfCollectApp.Name))
		// query.Add("q", fmt.Sprintf("organization_guid:%s", orgGuid))
		// query.Add("q", fmt.Sprintf("space_guid:%s", spaceGuid))
		query.Set(inlineDepthRelations, depth)
		apps, err := client.ListAppsByQuery(query)
		if err != nil {
			logrus.Errorf("Unable to collect the selected cf app %s : %s", cfCollectApp.Name, err)
		} else {
			if len(apps) == 0 {
				cfErr := cfclient.NewAppNotFoundError()
				logrus.Errorf(fmt.Sprintf(cfErr.Description, cfCollectApp.Name))
			} else {
				collectApps = append(collectApps, apps...)
			}
		}
	}
	return collectApps
}

func collectAllCfApps(client *cfclient.Client, depth string) []cfclient.App {
	var collectApps []cfclient.App
	query := url.Values{}
	query.Set(inlineDepthRelations, depth)
	apps, err := client.ListAppsByQuery(query)
	if err != nil {
		logrus.Errorf("Unable to get list of cf apps : %s", err)
	} else {
		collectApps = append(collectApps, apps...)
	}
	return collectApps
}

//Collect gets the cf app metadata by querying the cf app. Assumes that the authentication with cluster is already done.
func (c *CfAppsCollector) Collect(inputPath string, outputPath string) error {
	client, err := getCfClient()
	if err != nil {
		logrus.Errorf("Unable to connect to cf client : %s", err)
		return err
	}
	cfInfo, err := client.GetInfo()
	if err != nil {
		logrus.Errorf("Unable to get info of cf instance : %s", err)
	}
	var collectApps []cfclient.App
	if inputPath != "" {
		collectApps = collectSelectiveCfApps(inputPath, client, depth)
	} else {
		collectApps = collectAllCfApps(client, depth)
	}
	outputPath = filepath.Join(outputPath, "cf")
	err = os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		logrus.Errorf("Unable to create outputPath %s : %s", outputPath, err)
	}
	cfinstanceapps := collecttypes.NewCfApps()
	cfinstanceapps.Name = common.NormalizeForMetadataName(strings.TrimSpace(cfInfo.Name))
	for _, app := range collectApps {
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
	fileName := "cfapps-" + cfinstanceapps.Name
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
