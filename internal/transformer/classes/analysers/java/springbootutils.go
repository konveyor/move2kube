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

package java

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/source/springboot"
	"github.com/magiconair/properties"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	springbootAppNameConfig  = "spring.application.name"
	springbootProfilesConfig = "spring.profiles"
)

type filetype string

const (
	yamlfile filetype = "yaml"
	propfile filetype = "properties"
)

const (
	yamlSeperator = `---`
)

type SpringBootMetadataFiles struct {
	bootstrapFiles     []string
	bootstrapYamlFiles []string
	appPropFiles       []string
	appYamlFiles       []string
}

func getSpringBootMetadataFiles(dir string) (springbootMetadataFiles SpringBootMetadataFiles) {
	springbootMetadataFiles.bootstrapFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), []string{"bootstrap.properties"}, nil)
	springbootMetadataFiles.bootstrapYamlFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), nil, []string{"bootstrap.ya?ml"})
	springbootMetadataFiles.appPropFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), nil, []string{"application(-.+)?.properties"})
	springbootMetadataFiles.appYamlFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), nil, []string{"application(-.+)?.ya?ml"})
	return
}

func getSpringBootAppNameAndProfiles(springbootMetadataFiles SpringBootMetadataFiles) (appName string, profiles []string) {
	appName = ""
	profiles = []string{}
	if len(springbootMetadataFiles.bootstrapFiles) != 0 {
		props, err := properties.LoadFiles(springbootMetadataFiles.bootstrapFiles, properties.UTF8, false)
		if err != nil {
			logrus.Errorf("Unable to read bootstrap properties : %s", err)
		} else {
			appName = props.GetString(springbootAppNameConfig, "")
		}
	}
	if len(springbootMetadataFiles.bootstrapYamlFiles) != 0 && appName != "" {
		propss := getYamlSegmentsAsProperties(getYamlSegmentsFromFiles(springbootMetadataFiles.bootstrapYamlFiles))
		for _, props := range propss {
			if appName = props.GetString(springbootAppNameConfig, ""); appName != "" {
				break
			}
		}
	}
	if len(springbootMetadataFiles.appPropFiles) != 0 {
		for _, appPropFile := range springbootMetadataFiles.appPropFiles {
			if appPropFile == "application.properties" {
				propss := getYamlSegmentsAsProperties(getYamlSegmentsFromFiles([]string{appPropFile}))
				for _, props := range propss {
					if appName != "" {
						appName = props.GetString(springbootAppNameConfig, "")
					}
					if currProfilesStr := strings.TrimSpace(props.GetString(springbootProfilesConfig, "")); currProfilesStr != "" {
						currProfiles := strings.Split(currProfilesStr, ",")
						for _, currProfile := range currProfiles {
							currProfile = strings.TrimSpace(currProfile)
							if currProfile != "" && !common.IsStringPresent(profiles, currProfile) {
								profiles = append(profiles, currProfile)
							}
						}
					}
				}
			}
		}
	}
	appYamlPropFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), nil, []string{"application(-.+)?.ya?ml"})
	if len(appYamlPropFiles) != 0 {
		if appName != "" {
			for _, appYamlFile := range appYamlPropFiles {
				appYaml := springboot.SpringApplicationYaml{}
				err := common.ReadYaml(appYamlFile, &appYaml)
				if err != nil {
					logrus.Errorf("Unable to read bootstrap yaml properties : %s", err)
				} else {
					if appYaml.Spring.SpringApplication.Name != "" {
						appName = appYaml.Spring.SpringApplication.Name
						break
					}
				}
			}
		}
	}
	return
}

func getYamlAsProperties(yamlStr string) (props *properties.Properties, err error) {
	decoder := yaml.NewDecoder(strings.NewReader(yamlStr))
	var dataBucket yaml.Node
	errorReading := decoder.Decode(&dataBucket)
	if errorReading != io.EOF && errorReading != nil {
		return nil, errorReading
	}
	var output bytes.Buffer
	writer := bufio.NewWriter(&output)
	propsEncoder := yqlib.NewPropertiesEncoder(writer)
	err = propsEncoder.Encode(&dataBucket)
	if err != nil {
		logrus.Errorf("Error while encoding to properties : %s", err)
		return nil, err
	}
	writer.Flush()
	return properties.LoadString(output.String())
}

func getYamlSegmentsFromFile(yamlFilename string) (yamlSegments []string) {
	yamlbytes, err := ioutil.ReadFile(yamlFilename)
	if err != nil {
		logrus.Errorf("Unable to read bootstrap yaml file : %s", err)
		return nil
	}
	return strings.Split(string(yamlbytes), yamlSeperator)
}

func getYamlSegmentsFromFiles(yamlFilenames []string) (yamlSegments []string) {
	yamlSegments = []string{}
	for _, yamlFilename := range yamlFilenames {
		yamlSegments = append(yamlSegments, getYamlSegmentsFromFile(yamlFilename)...)
	}
	return
}

func getYamlSegmentsAsProperties(yamlSegments []string) (props []*properties.Properties) {
	props = []*properties.Properties{}
	for _, yamlSegment := range yamlSegments {
		propsForSegment, err := properties.LoadString(yamlSegment)
		if err != nil {
			logrus.Errorf("Unable to parse properties segment : %s", err)
			continue
		}
		props = append(props, propsForSegment)
	}
	return
}
