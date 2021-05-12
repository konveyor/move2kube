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

package gettransformdata

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/starlark/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
)

// GetTransforms returns the transformations
func GetTransforms(transformsPath string, dynQuesFn types.DynamicQuestionFnT) ([]types.TransformT, error) {
	log.Trace("start GetTransforms")
	defer log.Trace("end GetTransforms")
	transformPaths, err := common.GetFilesByExt(transformsPath, []string{"." + types.TransformFileExtension})
	if err != nil {
		return nil, err
	}
	return GetTransformsFromPaths(transformPaths, dynQuesFn)
}

// GetTransformsFromPaths returns the transformations given a list of script file paths
func GetTransformsFromPaths(transformPaths []string, dynQuesFn types.DynamicQuestionFnT) ([]types.TransformT, error) {
	transforms := []types.TransformT{}
	for _, transformPath := range transformPaths {
		transformBytes, err := ioutil.ReadFile(transformPath)
		if err != nil {
			return transforms, fmt.Errorf("failed to read the file at path %s Error: %q", transformPath, err)
		}
		currTransforms, err := GetTransformsFromSource(string(transformBytes), dynQuesFn)
		if err != nil {
			return transforms, fmt.Errorf("failed to get the starlark transform from the file at path %s Error: %q", transformPath, err)
		}
		transforms = append(transforms, currTransforms...)
	}
	return transforms, nil
}

// GetTransformsFromSource gets a list of transforms given a transformation script
func GetTransformsFromSource(transformStr string, dynQuesFn types.DynamicQuestionFnT) ([]types.TransformT, error) {
	log.Trace("start GetTransformsFromSource")
	defer log.Trace("end GetTransformsFromSource")
	return new(SimpleTransformT).GetTransformsFromSource(transformStr, dynQuesFn)
}

// GetK8sResources gets the k8s resources
func GetK8sResources(k8sResourcesPath string) ([]types.K8sResourceT, error) {
	log.Trace("start GetK8sResources")
	defer log.Trace("end GetK8sResources")
	yamlPaths, err := common.GetFilesByExt(k8sResourcesPath, []string{".yaml"})
	if err != nil {
		return nil, err
	}
	k8sResources := []types.K8sResourceT{}
	for _, yamlPath := range yamlPaths {
		k8sYamlBytes, err := ioutil.ReadFile(yamlPath)
		if err != nil {
			log.Errorf("Failed to read the yaml file at path %s . Error: %q", yamlPath, err)
			continue
		}
		currK8sResources, err := GetK8sResourcesFromYaml(string(k8sYamlBytes))
		if err != nil {
			log.Debugf("Failed to get k8s resources from the yaml file at path %s . Error: %q", yamlPath, err)
			continue
		}
		k8sResources = append(k8sResources, currK8sResources...)
	}
	return k8sResources, nil
}

// GetK8sResourcesFromYaml decodes k8s resources from yaml
func GetK8sResourcesFromYaml(k8sYaml string) ([]types.K8sResourceT, error) {
	// TODO: split yaml file into multiple resources

	// NOTE: This roundabout method is required to avoid yaml.v3 unmarshalling timestamps into time.Time
	var resourceI interface{}
	if err := yaml.Unmarshal([]byte(k8sYaml), &resourceI); err != nil {
		log.Errorf("Failed to unmarshal k8s yaml. Error: %q", err)
		return nil, err
	}
	resourceJSONBytes, err := json.Marshal(resourceI)
	if err != nil {
		log.Errorf("Failed to marshal the k8s resource into json. K8s resource:\n+%v\nError: %q", resourceI, err)
		return nil, err
	}
	var k8sResource types.K8sResourceT
	err = json.Unmarshal(resourceJSONBytes, &k8sResource)
	return []types.K8sResourceT{k8sResource}, err
}

// GetK8sResourceFromObject converts a runtime.Object into a K8sResourceT
func GetK8sResourceFromObject(obj runtime.Object) (types.K8sResourceT, error) {
	objJSONBytes, err := json.Marshal(obj)
	if err != nil {
		log.Debugf("Failed to marshal the runtime.Object %+v into json. Error: %q", obj, err)
		return nil, err
	}
	var k8sResource types.K8sResourceT
	err = json.Unmarshal(objJSONBytes, &k8sResource)
	return k8sResource, err
}

// GetObjectFromK8sResource converts a K8sResourceT into a runtime.Object
// resource: The resource to convert
// obj: The target struct type (Deployment, Pod, Service, etc.). This can be just an empty struct of the correct type.
func GetObjectFromK8sResource(resource types.K8sResourceT, obj runtime.Object) (runtime.Object, error) {
	if obj == nil {
		return nil, fmt.Errorf("failed to convert the K8sResourceT into a runtime.Object. The target object type is nil")
	}
	// Since K8s structs only have JSON struct tags, we marshal into JSON and back into runtime.Object
	resourceBytes, err := json.Marshal(resource)
	if err != nil {
		log.Errorf("failed to marshal the K8sResourceT to json. Error: %q", err)
		return nil, err
	}
	newObj := reflect.New(reflect.ValueOf(obj).Elem().Type()).Interface().(runtime.Object)
	if err = json.Unmarshal(resourceBytes, newObj); err != nil {
		log.Errorf("failed to unmarshal the json into runtime.Object. Error: %q", err)
		return nil, err
	}
	return newObj, nil
}
