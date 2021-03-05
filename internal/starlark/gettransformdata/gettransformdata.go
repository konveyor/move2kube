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
	"io/ioutil"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/starlark/types"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// GetTransforms returns the transformations
func GetTransforms(transformsPath string, ansFn types.AnswerFnT, statQuesFn types.StaticQuestionFnT, dynQuesFn types.DynamicQuestionFnT) ([]types.TransformT, error) {
	log.Trace("start GetTransforms")
	defer log.Trace("end GetTransforms")
	transformPaths, err := common.GetFilesByExt(transformsPath, []string{"." + types.TransformFileExtension})
	if err != nil {
		return nil, err
	}
	return GetTransformsFromPaths(transformPaths, ansFn, statQuesFn, dynQuesFn)
}

// GetTransformsFromPaths returns the transformations given a list of script file paths
func GetTransformsFromPaths(transformPaths []string, ansFn types.AnswerFnT, statQuesFn types.StaticQuestionFnT, dynQuesFn types.DynamicQuestionFnT) ([]types.TransformT, error) {
	transforms := []types.TransformT{}
	for _, transformPath := range transformPaths {
		transformBytes, err := ioutil.ReadFile(transformPath)
		if err != nil {
			return transforms, err
		}
		currTransforms, err := GetTransformsFromSource(string(transformBytes), ansFn, statQuesFn, dynQuesFn)
		if err != nil {
			return transforms, err
		}
		transforms = append(transforms, currTransforms...)
	}
	return transforms, nil
}

// GetTransformsFromSource gets a list of transforms given a transformation script
func GetTransformsFromSource(transformStr string, ansFn types.AnswerFnT, statQuesFn types.StaticQuestionFnT, dynQuesFn types.DynamicQuestionFnT) ([]types.TransformT, error) {
	log.Trace("start GetTransformsFromSource")
	defer log.Trace("end GetTransformsFromSource")
	return new(SimpleTransformT).GetTransformsFromSource(transformStr, ansFn, statQuesFn, dynQuesFn)
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
	var k8sResource types.MapT
	err := yaml.Unmarshal([]byte(k8sYaml), &k8sResource)
	return []types.K8sResourceT{k8sResource}, err
}
