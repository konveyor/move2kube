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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/magiconair/properties"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/kubernetes/pkg/apis/core"
)

var (
	defaultSpringBootResourcesPath = filepath.Join("src", "main", "resources")
)

const (
	springbootAppNameConfig  = "spring.application.name"
	springbootProfilesConfig = "spring.profiles"
	springbootGroup          = "org.springframework.boot"
)

const (
	seperator = `---`
)

// SpringBootMetadataFiles defines the lists of configuration files from spring boot applications
type SpringBootMetadataFiles struct {
	bootstrapFiles     []string
	bootstrapYamlFiles []string
	appPropFiles       []string
	appYamlFiles       []string
}

// FlattenedProperty defines the key value pair of the spring boot properties
type FlattenedProperty struct {
	Name  string
	Value string
}

func injectProperties(ir irtypes.IR, serviceName string) irtypes.IR {
	const vcapMountDir = "/vcap"
	const vcapPropertyFile = "vcap-properties.yaml"
	const vcapVolumeName = "vcapsecretvolume"
	const propertyImportEnvKey = "SPRING_CONFIG_IMPORT"

	for serviceKey, service := range ir.Services {
		if service.Name != serviceName {
			continue
		}
		// Flatten the VCAP_* environment JSON values to create spring-boot properties
		var vcapEnvList []FlattenedProperty
		for _, c := range service.Containers {
			for _, env := range c.Env {
				if env.Name == common.VcapServiceEnvName {
					vcapEnvList = append(vcapEnvList, flattenToVcapServicesProperties(env)...)
				} else if env.Name == common.VcapApplicationEnvName {
					vcapEnvList = append(vcapEnvList, flattenToVcapApplicationProperties(env)...)
				}
			}
		}
		if len(vcapEnvList) == 0 {
			continue
		}
		// Dump the entire VCAP_* property key-value pair data as one large chunk of string data
		// which will then be used as value to the VCAP property file name.
		var data []string
		for _, vcapEnv := range vcapEnvList {
			data = append(data, strings.Join([]string{vcapEnv.Name, vcapEnv.Value}, ":"))
		}
		// Create a secret for VCAP_* property key-value pairs
		ir.Storages = append(ir.Storages, irtypes.Storage{Name: serviceName,
			StorageType: irtypes.SecretKind,
			Content:     map[string][]byte{vcapPropertyFile: []byte(strings.Join(data, "\n"))}})
		// Create volume mount path for by assigning a pre-defined directory and property file.
		mountPath := filepath.Join(vcapMountDir, vcapPropertyFile)
		for index, c := range service.Containers {
			// Add an environment variable for SPRING_CONFIG_IMPORT and its value in every container
			c.Env = append(c.Env, core.EnvVar{Name: propertyImportEnvKey, Value: mountPath})
			// Create volume mounts for each container of the service
			c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{Name: vcapVolumeName, MountPath: mountPath})
			service.Containers[index] = c
		}
		// Create a volume for each service which maps to the secret created for VCAP_* property key-value pairs
		service.Volumes = append(service.Volumes,
			core.Volume{Name: vcapVolumeName,
				VolumeSource: core.VolumeSource{Secret: &core.SecretVolumeSource{SecretName: serviceName}}})
		ir.Services[serviceKey] = service
	}
	return ir
}

// interfaceSliceToDelimitedString converts an interface slice to string slice
func interfaceSliceToDelimitedString(intSlice []interface{}) string {
	var stringSlice []string
	for _, value := range intSlice {
		stringSlice = append(stringSlice, fmt.Sprintf("%v", value))
	}
	return strings.Join(stringSlice, ",")
}

// flattenPropertyKey flattens a given variable defined by <name, unflattenedValue>
func flattenPropertyKey(prefix string, unflattenedValue interface{}) []FlattenedProperty {
	var flattenedList []FlattenedProperty
	switch unflattened := unflattenedValue.(type) {
	case []interface{}:
		flattenedList = append(flattenedList,
			FlattenedProperty{Name: prefix, Value: interfaceSliceToDelimitedString(unflattened)})
		for index, value := range unflattened {
			envName := fmt.Sprintf("%s[%v]", prefix, index)
			flattenedList = append(flattenedList, flattenPropertyKey(envName, value)...)
		}
	case map[string]interface{}:
		for name, value := range unflattened {
			envName := fmt.Sprintf("%s.%s", prefix, name)
			flattenedList = append(flattenedList, flattenPropertyKey(envName, value)...)
		}
	case string:
		return []FlattenedProperty{{Name: prefix,
			Value: fmt.Sprintf("%s", unflattened)}}
	case int:
		return []FlattenedProperty{{Name: prefix,
			Value: fmt.Sprintf("%d", unflattened)}}
	case bool:
		return []FlattenedProperty{{Name: prefix,
			Value: fmt.Sprintf("%t", unflattened)}}
	default:
		if unflattened != nil {
			return []FlattenedProperty{{Name: prefix,
				Value: fmt.Sprintf("%#v", unflattened)}}
		} else {
			return []FlattenedProperty{{Name: prefix, Value: ""}}
		}
	}
	return flattenedList
}

// flattenToVcapServicesProperties flattens the variables specified in VCAP_SERVICES
func flattenToVcapServicesProperties(env core.EnvVar) []FlattenedProperty {
	var flattenedEnvList []FlattenedProperty
	var serviceInstanceMap map[string][]interface{}
	err := json.Unmarshal([]byte(env.Value), &serviceInstanceMap)
	if err != nil {
		logrus.Errorf("Could not unmarshal the service map instance (%s) in %s: %s", env.Name, err)
		return nil
	}
	for _, serviceInstances := range serviceInstanceMap {
		for _, serviceInstance := range serviceInstances {
			mapOfInstance := serviceInstance.(map[string]interface{})
			key := ""
			if serviceName, ok := mapOfInstance["name"].(string); ok {
				key = serviceName
			} else {
				if labelName, ok := mapOfInstance["label"].(string); ok {
					key = labelName
				}
			}
			flattenedEnvList = append(flattenedEnvList,
				flattenPropertyKey("vcap.services."+key, serviceInstance)...)
		}
	}
	return flattenedEnvList
}

// flattenToVcapApplicationProperties flattens the variables specified in VCAP_APPLICATION
func flattenToVcapApplicationProperties(env core.EnvVar) []FlattenedProperty {
	var serviceInstanceMap map[string]interface{}
	err := json.Unmarshal([]byte(env.Value), &serviceInstanceMap)
	if err != nil {
		logrus.Errorf("Could not unmarshal the service map instance (%s) in %s: %s", env.Name, err)
		return nil
	}
	return flattenPropertyKey("vcap.application", serviceInstanceMap)
}

func getSpringBootAppNameAndProfilesFromDir(dir string) (appName string, profiles []string) {
	return getSpringBootAppNameAndProfiles(getSpringBootMetadataFiles(dir))
}

func getSpringBootMetadataFiles(dir string) (springbootMetadataFiles SpringBootMetadataFiles) {
	springbootMetadataFiles.bootstrapFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultSpringBootResourcesPath), []string{"bootstrap.properties"}, nil)
	springbootMetadataFiles.bootstrapYamlFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultSpringBootResourcesPath), nil, []string{"bootstrap.ya?ml"})
	springbootMetadataFiles.appPropFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultSpringBootResourcesPath), nil, []string{"application(-.+)?.properties"})
	springbootMetadataFiles.appYamlFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultSpringBootResourcesPath), nil, []string{"application(-.+)?.ya?ml"})
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
		propss := getYamlSegmentsAsProperties(getSegmentsFromFiles(springbootMetadataFiles.bootstrapYamlFiles))
		for _, props := range propss {
			if appName = props.GetString(springbootAppNameConfig, ""); appName != "" {
				break
			}
		}
	}
	if len(springbootMetadataFiles.appPropFiles) != 0 {
		for _, appPropFile := range springbootMetadataFiles.appPropFiles {
			if filepath.Base(appPropFile) == "application.properties" {
				propss := getPropertiesFileSegmentsAsProperties(getSegmentsFromFiles([]string{appPropFile}))
				for _, props := range propss {
					if appName != "" {
						appName = props.GetString(springbootAppNameConfig, "")
					}
					if currProfilesStr := strings.TrimSpace(props.GetString(springbootProfilesConfig, "")); currProfilesStr != "" {
						currProfiles := strings.Split(currProfilesStr, ",")
						for _, currProfile := range currProfiles {
							currProfile = strings.TrimPrefix(strings.TrimSpace(currProfile), `!`)
							if currProfile != "" && !common.IsStringPresent(profiles, currProfile) {
								profiles = append(profiles, currProfile)
							}
						}
					}
				}
			} else {
				currProfile := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(appPropFile), "application-"), ".properties")
				if currProfile != "" && !common.IsStringPresent(profiles, currProfile) {
					profiles = append(profiles, currProfile)
				}
			}
		}
	}
	if len(springbootMetadataFiles.appYamlFiles) != 0 {
		for _, appYamlFile := range springbootMetadataFiles.appYamlFiles {
			if strings.HasPrefix(appYamlFile, "application.") {
				propss := getYamlSegmentsAsProperties(getSegmentsFromFiles([]string{appYamlFile}))
				for _, props := range propss {
					if appName != "" {
						appName = props.GetString(springbootAppNameConfig, "")
					}
					if currProfilesStr := strings.TrimSpace(props.GetString(springbootProfilesConfig, "")); currProfilesStr != "" {
						currProfiles := strings.Split(currProfilesStr, ",")
						for _, currProfile := range currProfiles {
							currProfile = strings.TrimPrefix(strings.TrimSpace(currProfile), `!`)
							if currProfile != "" && !common.IsStringPresent(profiles, currProfile) {
								profiles = append(profiles, currProfile)
							}
						}
					}
				}
			} else {
				currProfile := strings.TrimSuffix(strings.TrimSuffix(strings.TrimPrefix(filepath.Base(appYamlFile), "application-"), ".yaml"), ".yml")
				if currProfile != "" && !common.IsStringPresent(profiles, currProfile) {
					profiles = append(profiles, currProfile)
				}
			}
		}
	}
	return appName, profiles
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

func getSegmentsFromFile(fileName string) (segments []string) {
	filebytes, err := os.ReadFile(fileName)
	if err != nil {
		logrus.Errorf("Unable to read file : %s", err)
		return nil
	}
	return strings.Split(string(filebytes), seperator)
}

func getSegmentsFromFiles(filenames []string) (segments []string) {
	segments = []string{}
	for _, filename := range filenames {
		segments = append(segments, getSegmentsFromFile(filename)...)
	}
	return
}

func getYamlSegmentsAsProperties(yamlSegments []string) (props []*properties.Properties) {
	props = []*properties.Properties{}
	for _, yamlSegment := range yamlSegments {
		propsForSegment, err := getYamlAsProperties(yamlSegment)
		if err != nil {
			logrus.Errorf("Unable to decode yaml file as properties : %s", err)
		}
		props = append(props, propsForSegment)
	}
	return
}

func getPropertiesFileSegmentsAsProperties(segments []string) (props []*properties.Properties) {
	props = []*properties.Properties{}
	for _, segment := range segments {
		propsForSegment, err := properties.LoadString(segment)
		if err != nil {
			logrus.Errorf("Unable to parse properties segment : %s", err)
			continue
		}
		props = append(props, propsForSegment)
	}
	return
}
