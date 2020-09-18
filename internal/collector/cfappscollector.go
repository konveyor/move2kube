/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package collector

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	sourcetypes "github.com/konveyor/move2kube/internal/collector/sourcetypes"
	common "github.com/konveyor/move2kube/internal/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
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

	//To run: cf curl /v2/apps/
	cmd := exec.Command("cf", "curl", "/v2/apps")
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("%s", err.Error())
		return err
	}
	log.Debugf("Cf Curl output %s", output)
	sourcecfinstanceapps := sourcetypes.CfInstanceApps{}
	err = json.Unmarshal([]byte(output), &sourcecfinstanceapps)
	if err != nil {
		log.Errorf("Error in unmarshalling yaml: %s. Skipping.", err)
		return err
	}
	outputPath = filepath.Join(outputPath, "cf")
	err = os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create outputPath %s : %s", outputPath, err)
	}
	cfinstanceapps := collecttypes.NewCfInstanceApps()
	cfinstanceapps.Spec.CfApplications = []collecttypes.CfApplication{}
	fileName := "instanceapps_"

	log.Debugf("Detected %d apps", len(sourcecfinstanceapps.CfResources))
	for _, sourcecfapp := range sourcecfinstanceapps.CfResources {
		app := collecttypes.CfApplication{}
		app.Name = sourcecfapp.CfAppEntity.Name
		log.Debugf("Reading info about %s", app.Name)

		if sourcecfapp.CfAppEntity.Buildpack != "null" {
			app.Buildpack = sourcecfapp.CfAppEntity.Buildpack
		}
		if sourcecfapp.CfAppEntity.DetectedBuildpack != "null" {
			app.DetectedBuildpack = sourcecfapp.CfAppEntity.DetectedBuildpack
		}
		if sourcecfapp.CfAppEntity.DockerImage != "null" {
			app.DockerImage = sourcecfapp.CfAppEntity.DockerImage
		}
		app.Instances = sourcecfapp.CfAppEntity.Instances
		app.Memory = sourcecfapp.CfAppEntity.Memory
		app.Env = sourcecfapp.CfAppEntity.Env
		app.Ports = sourcecfapp.CfAppEntity.Ports
		cfinstanceapps.Spec.CfApplications = append(cfinstanceapps.Spec.CfApplications, app)

		fileName = fileName + app.Name
	}

	if fileName != "" {
		outputPath = filepath.Join(outputPath, common.NormalizeForFilename(fileName)+".yaml")
		err = common.WriteYaml(outputPath, cfinstanceapps)
		if err != nil {
			log.Errorf("Unable to write collect output : %s", err)
		}
		return err
	}

	return nil
}
