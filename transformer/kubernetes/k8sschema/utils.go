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

package k8sschema

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Intersection finds overlapping objects between the two arrays
func Intersection(objs1 []runtime.Object, objs2 []runtime.Object) []runtime.Object {
	objs := []runtime.Object{}
	for _, obj1 := range objs1 {
		found := false
		for _, obj2 := range objs2 {
			if cmp.Equal(obj1, obj2) {
				found = true
				break
			}
		}
		if found {
			objs = append(objs, obj1)
		}
	}
	return objs
}

// GetInfoFromK8sResource returns some useful information given a k8s resource
func GetInfoFromK8sResource(k8sResource K8sResourceT) (kind, apiVersion, name string, err error) {
	logrus.Trace("start getInfoFromK8sResource")
	defer logrus.Trace("end getInfoFromK8sResource")
	kindI, ok := k8sResource["kind"]
	if !ok {
		return "", "", "", fmt.Errorf("there is no kind specified in the k8s resource %+v", k8sResource)
	}
	kind, ok = kindI.(string)
	if !ok {
		return "", "", "", fmt.Errorf("expected kind to be of type string. Actual value %+v is of type %T", kindI, kindI)
	}
	apiVersionI, ok := k8sResource["apiVersion"]
	if !ok {
		return kind, "", "", fmt.Errorf("there is no apiVersion specified in the k8s resource %+v", k8sResource)
	}
	apiVersion, ok = apiVersionI.(string)
	if !ok {
		return kind, "", "", fmt.Errorf("expected apiVersion to be of type string. Actual value %+v is of type %T", apiVersionI, apiVersionI)
	}
	metadataI, ok := k8sResource["metadata"]
	if !ok {
		return kind, apiVersion, "", fmt.Errorf("there is no metadata specified in the k8s resource %+v", k8sResource)
	}
	name, err = getNameFromMetadata(metadataI)
	if err != nil {
		return kind, apiVersion, "", err
	}
	return kind, apiVersion, name, nil
}

func getNameFromMetadata(metadataI interface{}) (string, error) {
	metadata, ok := metadataI.(map[interface{}]interface{})
	if !ok {
		metadata, ok := metadataI.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("expected metadata to be of map type. Actual value %+v is of type %T", metadataI, metadataI)
		}
		nameI, ok := metadata["name"]
		if !ok {
			return "", fmt.Errorf("there is no name specified in the k8s resource metadata %+v", metadata)
		}
		name, ok := nameI.(string)
		if !ok {
			return "", fmt.Errorf("expected name to be of type string. Actual value %+v is of type %T", nameI, nameI)
		}
		return name, nil
	}
	nameI, ok := metadata["name"]
	if !ok {
		return "", fmt.Errorf("there is no name specified in the k8s resource metadata %+v", metadata)
	}
	name, ok := nameI.(string)
	if !ok {
		return "", fmt.Errorf("expected name to be of type string. Actual value %+v is of type %T", nameI, nameI)
	}
	return name, nil
}

// GetK8sResourcesWithPaths gets the k8s resources from a folder along
// with the relaive paths where they were found.
// Mutiple resources maybe specified in the same yaml file.
func GetK8sResourcesWithPaths(k8sResourcesPath string, currDirOnly bool) (map[string][]K8sResourceT, error) {
	logrus.Trace("GetK8sResourcesWithPaths start")
	defer logrus.Trace("GetK8sResourcesWithPaths end")
	var yamlPaths []string
	var err error
	if currDirOnly {
		yamlPaths, err = common.GetFilesByExtInCurrDir(k8sResourcesPath, []string{".yaml", ".yml"})
		if err != nil {
			return nil, fmt.Errorf("failed to list the files with the given extensions in the directory '%s'. Error: %w", k8sResourcesPath, err)
		}
	} else {
		yamlPaths, err = common.GetFilesByExt(k8sResourcesPath, []string{".yaml", ".yml"})
		if err != nil {
			return nil, fmt.Errorf("failed to list the files with the given extensions in the directory '%s'. Error: %w", k8sResourcesPath, err)
		}
	}
	k8sResources := map[string][]K8sResourceT{}
	for _, yamlPath := range yamlPaths {
		k8sYamlBytes, err := os.ReadFile(yamlPath)
		if err != nil {
			logrus.Errorf("Failed to read the yaml file at path %s . Error: %q", yamlPath, err)
			continue
		}
		currK8sResources, err := GetK8sResourcesFromYaml(string(k8sYamlBytes))
		if err != nil {
			logrus.Debugf("Failed to get k8s resources from the yaml file at path %s . Error: %q", yamlPath, err)
			continue
		}
		relYamlPath, err := filepath.Rel(k8sResourcesPath, yamlPath)
		if err != nil {
			logrus.Errorf("failed to make the k8s yaml path %s relative to the source folder %s . Error: %q", yamlPath, k8sResourcesPath, err)
			continue
		}
		k8sResources[relYamlPath] = append(k8sResources[relYamlPath], currK8sResources...)
	}
	return k8sResources, nil
}

// GetK8sResourcesFromYaml decodes k8s resources from yaml
func GetK8sResourcesFromYaml(k8sYaml string) ([]K8sResourceT, error) {
	// TODO: split yaml file into multiple resources

	// NOTE: This roundabout method is required to avoid yaml.v3 unmarshalling timestamps into time.Time
	var resourceI interface{}
	if err := yaml.Unmarshal([]byte(k8sYaml), &resourceI); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the string '%s' as YAML. Error: %w", k8sYaml, err)
	}
	resourceJSONBytes, err := json.Marshal(resourceI)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the K8s resource as JSON. K8s resource: %+v Error: %w", resourceI, err)
	}
	var k8sResource K8sResourceT
	if err := json.Unmarshal(resourceJSONBytes, &k8sResource); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the string '%s' as a K8s resource JSON. Error: %w", resourceJSONBytes, err)
	}
	return []K8sResourceT{k8sResource}, nil
}

// GetKubernetesObjsInDir returns returns all kubernetes objects in a dir
func GetKubernetesObjsInDir(dir string) []runtime.Object {
	objs := []runtime.Object{}
	codecs := serializer.NewCodecFactory(GetSchema())
	filePaths, err := common.GetFilesByExtInCurrDir(dir, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Errorf("Unable to fetch yaml files at path %q Error: %q", dir, err)
		return nil
	}
	for _, filePath := range filePaths {
		data, err := os.ReadFile(filePath)
		if err != nil {
			logrus.Debugf("Failed to read the yaml file at path %q Error: %q", filePath, err)
			continue
		}
		obj, _, err := codecs.UniversalDeserializer().Decode(data, nil, nil)
		if err != nil {
			logrus.Debugf("Failed to decode the file at path %q as a k8s file. Error: %q", filePath, err)
			continue
		}
		objGroupName := obj.GetObjectKind().GroupVersionKind().Group
		if objGroupName == types.GroupName {
			continue
		}
		objs = append(objs, obj)
	}
	return objs
}
