/*
Copyright IBM Corporation 2021

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

package irtranslators

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/k8sschema"
	"github.com/konveyor/move2kube/internal/k8sschema/fixer"
	"github.com/konveyor/move2kube/transformer/gettransformdata"
	"github.com/konveyor/move2kube/transformer/qa"
	"github.com/konveyor/move2kube/transformer/runtransforms"
	transformertypes "github.com/konveyor/move2kube/transformer/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// fixConvertAndTransformObjs runs fixers, converts to a supported version and runs transformations on the objects
func fixConvertAndTransformObjs(objs []runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, transformPaths []string) ([]runtime.Object, error) {
	// Fix and convert
	fixedAndConvertedObjs := []runtime.Object{}
	for _, obj := range objs {
		fixedAndConvertedObj, err := fixAndConvert(obj, clusterSpec)
		if err != nil {
			logrus.Errorf("Failed to fix and convert the runtime.Object. Object:\n%+v\nError: %q", obj, err)
			fixedAndConvertedObj = obj
		}
		fixedAndConvertedObjs = append(fixedAndConvertedObjs, fixedAndConvertedObj)
	}
	// Transform
	// Transform - get the k8s resources
	k8sResources := []transformertypes.K8sResourceT{}
	for _, fixedAndConvertedObj := range fixedAndConvertedObjs {
		k8sResource, err := gettransformdata.GetK8sResourceFromObject(fixedAndConvertedObj)
		if err != nil {
			logrus.Errorf("Failed to convert the object into a K8sResourceT. Object:\n%+v\nError: %q", fixedAndConvertedObj, err)
			return nil, err
		}
		k8sResources = append(k8sResources, k8sResource)
	}
	// Transform - get the transforms
	transforms, err := qa.GetTransformsFromPathsUsingDefaults(transformPaths)
	if err != nil {
		logrus.Errorf("Failed to get the transformations. Error: %q", err)
		return nil, err
	}
	// Transform - run the transformations on the k8s resources
	transformedK8sResources, err := runtransforms.ApplyTransforms(transforms, k8sResources)
	if err != nil {
		logrus.Errorf("Failed to apply the transformations. Error: %q", err)
		return nil, err
	}

	fixedConvertedAndTransformedObjs := []runtime.Object{}
	for i, transformedK8sResource := range transformedK8sResources {
		fixedConvertedAndTransformedObj, err := gettransformdata.GetObjectFromK8sResource(transformedK8sResource, fixedAndConvertedObjs[i])
		if err != nil {
			logrus.Errorf("Failed to convert the K8sResourceT back into a runtime.Object. K8s resource:\n%+v\nObject:\n%+v\nError: %q", transformedK8sResource, fixedAndConvertedObjs[i], err)
			fixedConvertedAndTransformedObj = fixedAndConvertedObjs[i]
		}
		fixedConvertedAndTransformedObjs = append(fixedConvertedAndTransformedObjs, fixedConvertedAndTransformedObj)
	}
	return fixedConvertedAndTransformedObjs, nil
}

// writeObjects writes the runtime objects to yaml files
func writeObjects(outputPath string, objs []runtime.Object) ([]string, error) {
	if err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission); err != nil {
		return nil, err
	}
	filesWritten := []string{}
	for _, obj := range objs {
		objYamlBytes, err := common.MarshalObjToYaml(obj)
		if err != nil {
			logrus.Errorf("failed to marshal the runtime.Object to yaml. Object:\n%+v\nError: %q", obj, err)
			continue
		}
		yamlPath := filepath.Join(outputPath, getFilename(obj))
		if err := ioutil.WriteFile(yamlPath, objYamlBytes, common.DefaultFilePermission); err != nil {
			logrus.Errorf("failed to write the yaml to file at path %s . Error: %q", yamlPath, err)
			continue
		}
		filesWritten = append(filesWritten, yamlPath)
	}
	return filesWritten, nil
}

func fixAndConvert(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec) (runtime.Object, error) {
	fixedobj := fixer.Fix(obj)
	return k8sschema.ConvertToSupportedVersion(fixedobj, clusterSpec)
}

func getFilename(obj runtime.Object) string {
	val := reflect.ValueOf(obj).Elem()
	typeMeta := val.FieldByName("TypeMeta").Interface().(metav1.TypeMeta)
	objectMeta := val.FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	return fmt.Sprintf("%s-%s.yaml", objectMeta.Name, strings.ToLower(typeMeta.Kind))
}
