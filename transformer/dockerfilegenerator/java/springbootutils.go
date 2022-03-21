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

const (
	springBootAppNameKey    = "spring.application.name"
	springBootServerPortKey = "server.port"
	springBootGroup         = "org.springframework.boot"
	// If no profile is active, a default profile is enabled.
	// The name of the default profile is default and it can be tuned using the spring.profiles.default Environment property,
	// as shown in the following example: spring.profiles.default=none
	defaultSpringProfile              = "default"                // https://docs.spring.io/spring-boot/docs/current/reference/html/features.html#features.profiles
	springBootSpringProfilesActiveKey = "spring.profiles.active" // https://docs.spring.io/spring-boot/docs/current/reference/html/application-properties.html#application-properties.core.spring.profiles.active
	springBootSpringProfilesKey       = "spring.profiles"        // Probably an alias for "spring.profiles.active"? Can't find it in the documentation
)

var (
	defaultSpringBootResourcesPath = filepath.Join("src", "main", "resources")
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
	// Flatten the VCAP_* environment JSON values to create spring-boot properties
	var vcapEnvList []FlattenedProperty
	for _, s := range ir.Storages {
		if s.StorageType != irtypes.SecretKind {
			continue
		}
		if serviceName+common.VcapCfSecretSuffix != s.Name {
			continue
		}
		for key, value := range s.Content {
			env := core.EnvVar{Name: key, Value: string(value)}
			if key == common.VcapServiceEnvName {
				vcapEnvList = append(vcapEnvList, flattenToVcapServicesProperties(env)...)
			} else if key == common.VcapApplicationEnvName {
				vcapEnvList = append(vcapEnvList, flattenToVcapApplicationProperties(env)...)
			}
		}
	}
	if len(vcapEnvList) == 0 {
		return ir
	}
	if service, ok := ir.Services[serviceName]; ok {
		// Dump the entire VCAP_* property key-value pair data as one large chunk of string data
		// which will then be used as value to the VCAP property file name.
		var data []string
		for _, vcapEnv := range vcapEnvList {
			data = append(data, strings.Join([]string{vcapEnv.Name, vcapEnv.Value}, ":"))
		}
		// Create a secret for VCAP_* property key-value pairs
		secretName := serviceName + common.VcapSpringBootSecretSuffix
		ir.Storages = append(ir.Storages, irtypes.Storage{Name: secretName,
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
				VolumeSource: core.VolumeSource{Secret: &core.SecretVolumeSource{SecretName: secretName}}})
		ir.Services[serviceName] = service
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
			Value: unflattened}}
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
		}
		return []FlattenedProperty{{Name: prefix, Value: ""}}
	}
	return flattenedList
}

// flattenToVcapServicesProperties flattens the variables specified in VCAP_SERVICES
func flattenToVcapServicesProperties(env core.EnvVar) []FlattenedProperty {
	var flattenedEnvList []FlattenedProperty
	var serviceInstanceMap map[string][]interface{}
	err := json.Unmarshal([]byte(env.Value), &serviceInstanceMap)
	if err != nil {
		logrus.Errorf("Could not unmarshal the service map instance (%s): %s", env.Name, err)
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
		logrus.Errorf("Could not unmarshal the service map instance (%s): %s", env.Name, err)
		return nil
	}
	return flattenPropertyKey("vcap.application", serviceInstanceMap)
}

func getSpringBootAppNameAndProfilesFromDir(dir string) (string, []string) {
	appName, profiles, _ := getSpringBootAppNameProfilesAndPorts(getSpringBootMetadataFiles(dir))
	return appName, profiles
}

func getSpringBootMetadataFiles(dir string) SpringBootMetadataFiles {
	springbootMetadataFiles := SpringBootMetadataFiles{}
	resourcesPath := filepath.Join(dir, defaultSpringBootResourcesPath)
	var err error
	springbootMetadataFiles.bootstrapFiles, err = common.GetFilesByName(resourcesPath, []string{"bootstrap.properties"}, nil)
	if err != nil {
		logrus.Debugf("failed to get files by name for path %s for bootstrap.properties. Error: %q", resourcesPath, err)
	}
	springbootMetadataFiles.bootstrapYamlFiles, err = common.GetFilesByName(resourcesPath, nil, []string{`bootstrap\.ya?ml`})
	if err != nil {
		logrus.Debugf("failed to get files by name for path %s bootstrap.yaml. Error: %q", resourcesPath, err)
	}
	springbootMetadataFiles.appPropFiles, err = common.GetFilesByName(resourcesPath, nil, []string{`application(-.+)?\.properties`})
	if err != nil {
		logrus.Debugf("failed to get files by name for path %s application.properties. Error: %q", resourcesPath, err)
	}
	springbootMetadataFiles.appYamlFiles, err = common.GetFilesByName(resourcesPath, nil, []string{`application(-.+)?\.ya?ml`})
	if err != nil {
		logrus.Debugf("failed to get files by name for path %s application.yaml. Error: %q", resourcesPath, err)
	}
	return springbootMetadataFiles
}

func getSpringBootAppNameProfilesAndPorts(springbootMetadataFiles SpringBootMetadataFiles) (appName string, profiles []string, profilePorts map[string][]int32) {
	appName = ""
	profiles = []string{}
	profilePorts = map[string][]int32{}
	// find sping boot app name from bootstrap.properties or bootstrap.yaml
	if len(springbootMetadataFiles.bootstrapFiles) != 0 {
		props, err := properties.LoadFiles(springbootMetadataFiles.bootstrapFiles, properties.UTF8, false)
		if err != nil {
			logrus.Errorf("failed to load the bootstrap properties files from paths %+v . Error: %q", springbootMetadataFiles.bootstrapFiles, err)
		} else {
			appName = props.GetString(springBootAppNameKey, "")
		}
	} else if len(springbootMetadataFiles.bootstrapYamlFiles) != 0 {
		propss := convertYamlDocumentsToProperties(getYamlDocumentsFromFiles(springbootMetadataFiles.bootstrapYamlFiles))
		for _, props := range propss {
			if appName = props.GetString(springBootAppNameKey, ""); appName != "" {
				break
			}
		}
	}
	// find spring boot app name from application.properties
	for _, appPropFilePath := range springbootMetadataFiles.appPropFiles {
		// TODO: handle multi-document properties files https://spring.io/blog/2020/08/14/config-file-processing-in-spring-boot-2-4#multi-document-properties-files
		props, err := properties.LoadFile(appPropFilePath, properties.UTF8)
		if err != nil {
			logrus.Errorf("failed to load the file at path %s as a properties file. Error: %q", appPropFilePath, err)
			continue
		}
		appPropFilename := filepath.Base(appPropFilePath)
		if appPropFilename == "application.properties" {
			// get app name
			appName = props.GetString(springBootAppNameKey, appName)
			// get active profiles
			// https://docs.spring.io/spring-boot/docs/current/reference/html/features.html#features.profiles
			activeProfilesStr := props.GetString(springBootSpringProfilesActiveKey, "")
			if activeProfilesStr == "" {
				activeProfilesStr = props.GetString(springBootSpringProfilesKey, "")
			}
			activeProfiles := getSpringProfiles(activeProfilesStr)
			// add to list of known spring profiles
			for _, activeProfile := range activeProfiles {
				if !common.IsStringPresent(profiles, activeProfile) {
					profiles = append(profiles, activeProfile)
				}
			}
			// get ports
			if appPort := props.GetInt(springBootServerPortKey, -1); appPort != -1 {
				if len(activeProfiles) > 0 {
					for _, activeProfile := range activeProfiles {
						profilePorts[activeProfile] = append(profilePorts[activeProfile], int32(appPort))
					}
				} else {
					profilePorts[defaultSpringProfile] = append(profilePorts[defaultSpringProfile], int32(appPort))
				}
			}
		} else {
			activeProfile := strings.TrimSuffix(strings.TrimPrefix(appPropFilename, "application-"), ".properties")
			if activeProfile == "" {
				logrus.Warnf("invalid/empty spring profile name for the properties file at path %s", appPropFilePath)
				continue
			}
			// add to list of known spring profiles
			if !common.IsStringPresent(profiles, activeProfile) {
				profiles = append(profiles, activeProfile)
			}
			// get ports
			if appPort := props.GetInt(springBootServerPortKey, -1); appPort != -1 {
				profilePorts[activeProfile] = append(profilePorts[activeProfile], int32(appPort))
			}
			// TODO: should we try to get app name for each profile as well?
		}
	}
	// find spring boot app name from application.yaml
	for _, appYamlFilePath := range springbootMetadataFiles.appYamlFiles {
		// TODO: handle multi document yamls
		propss := convertYamlDocumentsToProperties(getYamlDocumentsFromFiles([]string{appYamlFilePath}))
		if len(propss) == 0 {
			logrus.Warnf("parsed out an empty properties struct from the file at path %s", appYamlFilePath)
			continue
		}
		props := propss[0]
		for _, p := range propss[1:] {
			props.Merge(p)
		}
		// get app name
		appName = props.GetString(springBootAppNameKey, appName)
		// get ports and profiles
		appYamlFilename := filepath.Base(appYamlFilePath)
		if appYamlFilename == "application.yml" || appYamlFilename == "application.yaml" {
			activeProfilesStr := props.GetString(springBootSpringProfilesActiveKey, "")
			if activeProfilesStr == "" {
				activeProfilesStr = props.GetString(springBootSpringProfilesKey, "")
			}
			activeProfiles := getSpringProfiles(activeProfilesStr)
			// add to list of known spring profiles
			for _, activeProfile := range activeProfiles {
				if !common.IsStringPresent(profiles, activeProfile) {
					profiles = append(profiles, activeProfile)
				}
			}
			// get ports
			if appPort := props.GetInt(springBootServerPortKey, -1); appPort != -1 {
				if len(activeProfiles) > 0 {
					for _, activeProfile := range activeProfiles {
						profilePorts[activeProfile] = append(profilePorts[activeProfile], int32(appPort))
					}
				} else {
					profilePorts[defaultSpringProfile] = append(profilePorts[defaultSpringProfile], int32(appPort))
				}
			}
		} else {
			activeProfile := strings.TrimSuffix(strings.TrimPrefix(appYamlFilename, "application-"), filepath.Ext(appYamlFilename))
			if activeProfile == "" {
				logrus.Warnf("invalid/empty spring profile name for the properties file at path %s", appYamlFilePath)
				continue
			}
			// add to list of known spring profiles
			if !common.IsStringPresent(profiles, activeProfile) {
				profiles = append(profiles, activeProfile)
			}
			// get ports
			if appPort := props.GetInt(springBootServerPortKey, -1); appPort != -1 {
				profilePorts[activeProfile] = append(profilePorts[activeProfile], int32(appPort))
			}
			// TODO: should we try to get app name for each profile as well?
		}
	}
	return appName, profiles, profilePorts
}

func getYamlDocumentsFromFile(filePath string) ([][]byte, error) {
	fileBytes, err := os.ReadFile(filePath)
	if err != nil {
		logrus.Errorf("failed to read file at path %s . Error: %q", filePath, err)
		return nil, nil
	}
	return common.SplitYAML(fileBytes)
}

func getYamlDocumentsFromFiles(filePaths []string) []string {
	segments := []string{}
	for _, filePath := range filePaths {
		docs, err := getYamlDocumentsFromFile(filePath)
		if err != nil {
			logrus.Errorf("failed to get YAML documents for the file at path %s , skipping. Error: %q", filePath, err)
			continue
		}
		for _, doc := range docs {
			segments = append(segments, string(doc))
		}
	}
	return segments
}

func getSpringProfiles(springProfilesStr string) []string {
	rawSpringProfiles := strings.Split(springProfilesStr, ",")
	springProfiles := []string{}
	for _, rawSpringProfile := range rawSpringProfiles {
		springProfile := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rawSpringProfile), "!"))
		if springProfile != "" {
			springProfiles = append(springProfiles, rawSpringProfile)
		}
	}
	return springProfiles
}

func convertYamlDocumentToProperties(doc string) (props *properties.Properties, err error) {
	decoder := yaml.NewDecoder(strings.NewReader(doc))
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

func convertYamlDocumentsToProperties(docs []string) []*properties.Properties {
	props := []*properties.Properties{}
	for _, doc := range docs {
		prop, err := convertYamlDocumentToProperties(doc)
		if err != nil {
			logrus.Errorf("failed to decode the YAML document as properties. Document: %s . Error: %q", doc, err)
		}
		props = append(props, prop)
	}
	return props
}
