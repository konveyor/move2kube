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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	cfclient "github.com/cloudfoundry-community/go-cfclient/v2"
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/types"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

const (
	inlineDepthRelations = "inline-relations-depth"
	depth                = "2"
	listAppsPath         = "/v2/apps"
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
	Filters       CfCollectFilters `yaml:"filters,omitempty"`
	CfCollectApps []CfCollectApp   `yaml:"applications,omitempty"`
}

// CfCollectFilters stores the spaceguid and querydepth to be used to filter while collecting metadata
type CfCollectFilters struct {
	SpaceGuid  string `yaml:"spaceguid,omitempty"`
	QueryDepth string `yaml:"query_depth,omitempty"`
}

// CfCollectApp defines CfCollectApp information
type CfCollectApp struct {
	AppSpec CfAppSpec `yaml:"application"`
}

// CfAppSpec defines CF app spec
type CfAppSpec struct {
	Name string `json:"name"`
	Guid string `json:"guid"`
}

// AppResponse defines app response
type AppResponse struct {
	Count     int           `json:"total_results"`
	Pages     int           `json:"total_pages"`
	NextUrl   string        `json:"next_url"`
	Resources []AppResource `json:"resources"`
}

// AppResource defines app resource
type AppResource struct {
	Meta   cfclient.Meta `json:"metadata"`
	Entity App           `json:"entity"`
}

// CfAppsCollector collects cf runtime applications
type CfAppsCollector struct {
}

// GetAnnotations returns annotations on which this collector should be invoked
func (c *CfAppsCollector) GetAnnotations() []string {
	annotations := []string{"cf", "cloudfoundry"}
	return annotations
}

// setQueryDepth sets the depth to be used in
func setQueryDepth(queryDepth string) url.Values {
	query := url.Values{}
	if queryDepth == "" {
		queryDepth = depth
	}
	query.Set(inlineDepthRelations, queryDepth)
	logrus.Debugf("CF collect query depth = %s", queryDepth)
	return query
}

// listAppsBySpaceGuid collects all CF apps data for given spaceGuid, if no appName/appGuid is present. If appName/appGuid is present, then it updates the path to be used in the listAppsByNameOrGuid func
func listAppsBySpaceGuid(client *cfclient.Client, spaceGuid string, queryDepth string, collectApps []App, numCfCollectApps int) (string, []App) {
	query := setQueryDepth(queryDepth)
	path := ""
	if spaceGuid == "" {
		return path, collectApps
	}
	logrus.Debugf("Detected CF Space guid: %s", spaceGuid)
	path = fmt.Sprintf("/v2/spaces/%s/apps", spaceGuid)
	if numCfCollectApps == 0 {
		apps, err := listApps(client, path, query, -1) // If no CF app is specified in yaml, collect all apps in the provided spaceguid
		if err != nil {
			logrus.Errorf("Unable to collect the cf apps from the Space guid %s : %s", spaceGuid, err)
		} else {
			collectApps = append(collectApps, apps...)
		}
	}
	return path, collectApps
}

func getAppByGuid(client *cfclient.Client, guid string, queryDepth string) (App, error) {
	var appResource AppResource
	query := setQueryDepth(queryDepth)
	requestUrl := getRequestUrl(listAppsPath+"/"+guid, query) // /v2/apps/:appGuid fetches the app with given guid
	logrus.Debugf("CF collect queryDepth val = %v", queryDepth)
	logrus.Debugf("CF collect request URL = %v", requestUrl)
	r := client.NewRequest("GET", requestUrl)
	resp, err := client.DoRequest(r)
	if err != nil {
		return App{}, errors.Wrap(err, "Error requesting apps")
	}
	defer resp.Body.Close()
	resBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return App{}, errors.Wrap(err, "Error reading app response body")
	}
	err = json.Unmarshal(resBody, &appResource)
	if err != nil {
		return App{}, errors.Wrap(err, "Error unmarshalling app")
	}
	return mergeAppResource(client, appResource), nil
}

func getAppsByName(client *cfclient.Client, path string, query url.Values, collectApps []App, appName string) []App {
	apps, err := listApps(client, path, query, -1)
	if err != nil {
		logrus.Errorf("Unable to collect the selected cf app %s : %s", appName, err)
		return collectApps
	}
	if len(apps) != 0 {
		collectApps = append(collectApps, apps...)
		return collectApps
	}
	cfErr := cfclient.NewAppNotFoundError()
	logrus.Errorf(fmt.Sprintf(cfErr.Description, appName))
	return collectApps
}

func listAppsByNameOrGuid(client *cfclient.Client, path string, cfCollectApps []CfCollectApp, queryDepth string) []App {
	var collectApps []App
	for _, cfCollectApp := range cfCollectApps {
		if path == "" {
			path = listAppsPath
		}
		appSpec := cfCollectApp.AppSpec
		if appSpec.Guid != "" {
			app, err := getAppByGuid(client, appSpec.Guid, queryDepth)
			if err != nil {
				logrus.Errorf("Unable to collect the app with guid %s %q", appSpec.Guid, err)
			} else {
				collectApps = append(collectApps, app)
				continue
			}
		}
		if appSpec.Name != "" {
			query := setQueryDepth(queryDepth)
			query.Add("q", fmt.Sprintf("name:%s", appSpec.Name)) // /v2/apps/ and /v2/spaces/:spaceGuid/apps/ support querying a particular app by AppName
			collectApps = getAppsByName(client, path, query, collectApps, appSpec.Name)
		}
	}
	return collectApps
}

// collectSelectiveCfApps collects the selected cf apps
func collectSelectiveCfApps(inputPath string, client *cfclient.Client) []App {
	var collectApps []App
	filePaths, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Errorf("failed to look for yaml files in the directory %s . Error: %q", inputPath, err)
		return collectApps
	}
	depth := ""
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
		path := ""
		if cfInstanceCollectApps.Spec.Filters.QueryDepth != "" {
			// This depth var to be used if no apps get collected by name/guid/spaceguid
			depth = cfInstanceCollectApps.Spec.Filters.QueryDepth
		}
		// If the yaml has spaceGuid in the filters and no appName/appGuid is present, collect all CF apps data for given spaceGuid. If appName/appGuid is present, then listAppsBySpaceGuid() updates the path to be used in the listAppsByNameOrGuid()
		path, collectAppsBySpaceGuid := listAppsBySpaceGuid(client, cfInstanceCollectApps.Spec.Filters.SpaceGuid, cfInstanceCollectApps.Spec.Filters.QueryDepth, collectApps, len(cfInstanceCollectApps.Spec.CfCollectApps))
		collectApps = append(collectApps, collectAppsBySpaceGuid...)
		if len(collectApps) == 0 {
			collectApps = append(collectApps, listAppsByNameOrGuid(client, path, cfInstanceCollectApps.Spec.CfCollectApps, cfInstanceCollectApps.Spec.Filters.QueryDepth)...)
		}
	}
	// If the yaml only has query_depth field, and no appGuid, appName, or spaceGuid, then collect all CF apps info for the given query depth
	if len(collectApps) == 0 && depth != "" {
		collectApps = collectAllCfApps(client, depth)
	}
	return collectApps
}

// collectAllCfApps collects all the cf apps
func collectAllCfApps(client *cfclient.Client, depth string) []App {
	var collectApps []App
	query := setQueryDepth(depth)
	apps, err := listApps(client, listAppsPath, query, -1)
	if err != nil {
		logrus.Errorf("Unable to get list of cf apps : %s", err)
		return collectApps
	}
	collectApps = append(collectApps, apps...)
	return collectApps
}

// Collect gets the cf app metadata by querying the cf app. Assumes that the authentication with cluster is already done.
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
	var collectApps []App
	if inputPath != "" {
		collectApps = collectSelectiveCfApps(inputPath, client)
	} else {
		collectApps = collectAllCfApps(client, depth)
	}
	outputPath = filepath.Join(outputPath, "cf")
	err = os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		logrus.Errorf("Unable to create outputPath %s : %s", outputPath, err)
	}
	cfinstanceapps := NewCfApps()
	cfinstanceapps.Name = common.NormalizeForMetadataName(strings.TrimSpace(cfInfo.Name))
	for _, app := range collectApps {
		cfapp := CfApp{
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
	cfinstanceapps = FormatMapsWithInterface(cfinstanceapps)
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

func getRequestUrl(path string, query url.Values) string {
	encodedQuery := ""
	if query.Encode() != "" {
		encodedQuery = "?" + query.Encode()
	}
	return path + encodedQuery
}

func listApps(client *cfclient.Client, path string, query url.Values, totalPages int) ([]App, error) {
	requestUrl := getRequestUrl(path, query)
	logrus.Debugf("CF collect requestUrl = %s", requestUrl)
	pages := 0
	apps := []App{}
	for {
		var appResp AppResponse
		r := client.NewRequest("GET", requestUrl)
		resp, err := client.DoRequest(r)

		if err != nil {
			return nil, errors.Wrap(err, "Error requesting apps")
		}
		defer resp.Body.Close()
		resBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Error reading app request")
		}

		err = json.Unmarshal(resBody, &appResp)
		if err != nil {
			return nil, errors.Wrap(err, "Error unmarshalling app")
		}

		for _, app := range appResp.Resources {
			apps = append(apps, mergeAppResource(client, app))
		}

		requestUrl = appResp.NextUrl
		if requestUrl == "" || query.Get("page") != "" {
			break
		}

		pages++
		if totalPages > 0 && pages >= totalPages {
			break
		}
	}
	return apps, nil
}

func mergeAppResource(client *cfclient.Client, app AppResource) App {
	app.Entity.Guid = app.Meta.Guid
	app.Entity.CreatedAt = app.Meta.CreatedAt
	app.Entity.UpdatedAt = app.Meta.UpdatedAt
	app.Entity.SpaceData.Entity.Guid = app.Entity.SpaceData.Meta.Guid
	app.Entity.SpaceData.Entity.OrgData.Entity.Guid = app.Entity.SpaceData.Entity.OrgData.Meta.Guid
	app.Entity.c = client
	return app.Entity
}
