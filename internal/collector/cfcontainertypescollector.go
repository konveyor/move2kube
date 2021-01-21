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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	sourcetypes "github.com/konveyor/move2kube/internal/collector/sourcetypes"
	"github.com/konveyor/move2kube/internal/common"
	containerize "github.com/konveyor/move2kube/internal/containerizer"
	source "github.com/konveyor/move2kube/internal/source"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/konveyor/move2kube/types/plan"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

// CFContainerTypesCollector collects buildpacks supported by the instance
type CFContainerTypesCollector struct {
}

// GetAnnotations returns annotations on which this collector should be invoked
func (c *CFContainerTypesCollector) GetAnnotations() []string {
	annotations := []string{"cloudfoundry", "cf"}
	return annotations
}

//Collect gets the cf containerization types
func (c *CFContainerTypesCollector) Collect(inputDirectory string, outputPath string) error {
	//Creating the output sub-directory if it does not exist
	outputPath = filepath.Join(outputPath, "cf")
	err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create output path %s : %s", outputPath, err)
		return err
	}
	var cfcontainerizers = collecttypes.NewCfContainerizers()
	cfcontainerizers.Spec.BuildpackContainerizers = []collecttypes.BuildpackContainerizer{}
	buildpackNames := getCfbuildpackNames(inputDirectory)
	log.Debugf("buildpackNames : %s", buildpackNames)
	cnbcontainerizer := new(containerize.CNBContainerizer)
	//TODO: How do you also load existing builders that are currently being collected by cnbbuildercollector?
	cnbcontainerizer.Init("")
	buildpacks := cnbcontainerizer.GetAllBuildpacks()
	log.Debugf("buildpacks : %s", buildpacks)
	fileName := "cfcontainertypes_"
	for _, buildpackName := range buildpackNames {
		buildpackcontainerizer := getBuildpackContainerizer(buildpackName, buildpacks)
		cfcontainerizers.Spec.BuildpackContainerizers = append(cfcontainerizers.Spec.BuildpackContainerizers, buildpackcontainerizer)
		fileName = fileName + buildpackName
	}
	if fileName != "" {
		outputPath = filepath.Join(outputPath, common.NormalizeForFilename(fileName)+".yaml")
		err := common.WriteYaml(outputPath, cfcontainerizers)
		if err != nil {
			log.Errorf("Unable to write cf container type output %s : %s", fileName, err)
		}
		return err
	}
	return fmt.Errorf("No buildpacks found")
}

func getBuildpackContainerizer(buildpackName string, options map[string][]string) collecttypes.BuildpackContainerizer { //[containerization taregt option][builder]
	buildpackcontainerizer := collecttypes.BuildpackContainerizer{}
	buildpackcontainerizer.ContainerBuildType = plan.CNBContainerBuildTypeValue
	buildpackcontainerizer.BuildpackName = buildpackName
	bpoptions := map[string]string{}
	bps := []string{}
	for targetoption, buildpacks := range options {
		option := common.GetClosestMatchingString(buildpacks, buildpackName)
		if _, ok := bpoptions[option]; !ok {
			bpoptions[option] = targetoption
			bps = append(bps, option)
		}
	}
	bp := common.GetClosestMatchingString(bps, buildpackName)
	buildpackcontainerizer.ContainerizationTargetOptions = []string{bpoptions[bp]}
	return buildpackcontainerizer
}

func getCfbuildpackNames(inputPath string) []string {
	buildpacks := []string{}
	if inputPath != "" {
		bps, err := getAllUsedBuildpacks(inputPath)
		if err != nil {
			log.Warnf("Unable to find used buildpacks : %s", err)
		} else {
			for _, buildpack := range bps {
				if !common.IsStringPresent(buildpacks, buildpack) {
					buildpacks = append(buildpacks, buildpack)
				}
			}
		}
	} else {
		bps, err := getAllCfInstanceBuildpacks()
		if err != nil {
			log.Warnf("Unable to collect buildpacks from cf instance : %s", err)
		} else {
			for _, buildpack := range bps {
				if !common.IsStringPresent(buildpacks, buildpack) {
					buildpacks = append(buildpacks, buildpack)
				}
			}
		}

		bps, err = getAllCfAppBuildpacks()
		if err != nil {
			log.Warnf("Unable to find used buildpacks : %s", err)
		} else {
			for _, buildpack := range bps {
				if !common.IsStringPresent(buildpacks, buildpack) {
					buildpacks = append(buildpacks, buildpack)
				}
			}
		}
	}
	return buildpacks
}

func getAllCfInstanceBuildpacks() ([]string, error) {
	var buildpacks []string
	cmd := exec.Command("cf", "buildpacks")
	outputStr, err := cmd.Output()
	if err != nil {
		log.Warnf("Error while getting buildpacks : %s", err)
		return nil, err
	}
	lines := strings.Split(string(outputStr), "\n")
	for _, line := range lines {
		if line == "Getting buildpacks..." {
			continue
		}
		buildpackmatches := strings.Fields(string(line))
		if len(buildpackmatches) == 0 || buildpackmatches[0] == "buildpack" {
			continue
		}
		buildpacks = append(buildpacks, buildpackmatches[0])

	}
	return buildpacks, nil
}

func getAllCfAppBuildpacks() ([]string, error) {
	var buildpacks []string
	cmd := exec.Command("cf", "curl", "/v2/apps")
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("%s", err.Error())
		return nil, err
	}
	log.Debugf("Cf Curl output %s", output)
	sourcecfinstanceapps := sourcetypes.CfInstanceApps{}
	err = json.Unmarshal([]byte(output), &sourcecfinstanceapps)
	if err != nil {
		log.Errorf("Error in unmarshalling yaml: %s. Skipping", err)
		return nil, err
	}

	log.Debugf("Detected %d apps", len(sourcecfinstanceapps.CfResources))
	for _, sourcecfapp := range sourcecfinstanceapps.CfResources {
		if sourcecfapp.CfAppEntity.Buildpack != "" {
			buildpacks = append(buildpacks, sourcecfapp.CfAppEntity.Buildpack)
		}
		if sourcecfapp.CfAppEntity.DetectedBuildpack != "" {
			buildpacks = append(buildpacks, sourcecfapp.CfAppEntity.DetectedBuildpack)
		}
	}
	return buildpacks, nil
}

func getAllUsedBuildpacks(directorypath string) ([]string, error) {
	var buildpacks []string
	files, err := common.GetFilesByExt(directorypath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize application manifest yamls : %s", err)
	}
	for _, fullpath := range files {
		applications, _, err := source.ReadApplicationManifest(fullpath, "", plantypes.Yamls)
		if err != nil {
			log.Debugf("Error while trying to parse manifest : %s", err)
			continue
		}
		for _, application := range applications {
			if application.Buildpack.IsSet {
				buildpacks = append(buildpacks, application.Buildpack.Value)
			}
			buildpacks = append(buildpacks, application.Buildpacks...)
		}
	}
	return buildpacks, nil
}
