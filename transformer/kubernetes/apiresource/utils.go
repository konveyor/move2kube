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

package apiresource

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/transformer/kubernetes/k8sschema"
	"github.com/konveyor/move2kube-wasm/transformer/kubernetes/k8sschema/fixer"
	collecttypes "github.com/konveyor/move2kube-wasm/types/collection"
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// TransformIRAndPersist transforms IR to yamls and writes to filesystem
func TransformIRAndPersist(
	ir irtypes.EnhancedIR,
	outputPath string,
	apiResources []IAPIResource,
	targetCluster collecttypes.ClusterMetadata,
	setDefaultValuesInYamls bool,
) (files []string, err error) {
	logrus.Trace("TransformIRAndPersist start")
	defer logrus.Trace("TransformIRAndPersist end")
	targetObjs := []runtime.Object{}
	for _, apiResource := range apiResources {
		newObjs := (&APIResource{IAPIResource: apiResource}).convertIRToObjects(ir, targetCluster)
		targetObjs = append(targetObjs, newObjs...)
	}
	if err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission); err != nil {
		return nil, fmt.Errorf("failed to create the deploy directory at path '%s' . Error: %w", outputPath, err)
	}
	logrus.Debugf("number of services to be serialized %d", len(targetObjs))
	convertedObjs, err := convertVersion(targetObjs, targetCluster.Spec, setDefaultValuesInYamls)
	if err != nil {
		return nil, fmt.Errorf("failed to fix, convert and transform the objects. Error: %w", err)
	}
	filesWritten, err := writeObjects(outputPath, convertedObjs)
	if err != nil {
		return nil, fmt.Errorf("failed to write the transformed objects to the directory at path '%s' . Error: %w", outputPath, err)
	}
	return filesWritten, nil
}

// TransformObjsAndPersist transforms versions of yamls in current directory and writes to filesystem
func TransformObjsAndPersist(inputPath, outputPath string, apis []IAPIResource, targetCluster collecttypes.ClusterMetadata, setDefaultValuesInYamls bool) (files []string, err error) {
	targetObjs := []runtime.Object{}
	if pendingObjs := k8sschema.GetKubernetesObjsInDir(inputPath); len(pendingObjs) != 0 {
		for _, apiResource := range apis {
			var newObjs []runtime.Object
			newObjs, pendingObjs = (&APIResource{IAPIResource: apiResource}).convertObjectsToSupportedVersion(pendingObjs, targetCluster)
			targetObjs = append(targetObjs, newObjs...)
		}
		targetObjs = append(targetObjs, pendingObjs...)
	}
	if err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission); err != nil {
		logrus.Errorf("failed to create deploy directory at path '%s' . Error: %q", outputPath, err)
	}
	logrus.Debugf("Total %d services to be serialized.", len(targetObjs))
	convertedObjs, err := convertVersion(targetObjs, targetCluster.Spec, setDefaultValuesInYamls)
	if err != nil {
		logrus.Errorf("Failed to fix, convert and transform the objects. Error: %q", err)
	}
	filesWritten, err := writeObjects(outputPath, convertedObjs)
	if err != nil {
		return nil, fmt.Errorf("failed to write the transformed objects to the directory at path '%s' . Error: %w", outputPath, err)
	}
	return filesWritten, nil
}

// writeObjects writes the runtime objects to yaml files
func writeObjects(outputPath string, objs []runtime.Object) ([]string, error) {
	if err := os.MkdirAll(outputPath, common.DefaultDirectoryPermission); err != nil {
		return nil, fmt.Errorf("failed to create the output directory at path '%s' . Error: %w", outputPath, err)
	}
	filesWritten := []string{}
	for _, obj := range objs {
		objYamlBytes, err := common.MarshalObjToYaml(obj)
		if err != nil {
			logrus.Errorf("failed to marshal the runtime. Object to yaml. Object: %+v Error: %q", obj, err)
			continue
		}
		yamlPath := filepath.Join(outputPath, getFilename(obj))
		if err := os.WriteFile(yamlPath, objYamlBytes, common.DefaultFilePermission); err != nil {
			logrus.Errorf("failed to write the yaml to file at path '%s' . Error: %q", yamlPath, err)
			continue
		}
		filesWritten = append(filesWritten, yamlPath)
	}
	return filesWritten, nil
}

func convertVersion(objs []runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, setDefaultValuesInYamls bool) ([]runtime.Object, error) {
	newobjs := []runtime.Object{}
	for _, obj := range objs {
		fixedobj := fixer.Fix(obj)
		newobj, err := k8sschema.ConvertToSupportedVersion(fixedobj, clusterSpec, setDefaultValuesInYamls)
		if err != nil {
			logrus.Errorf("failed to convert to supported version. Writing as is. Error: %q", err)
			newobj = obj
		}
		newobjs = append(newobjs, newobj)
	}
	return newobjs, nil
}

func getFilename(obj runtime.Object) string {
	val := reflect.ValueOf(obj).Elem()
	typeMeta := val.FieldByName("TypeMeta").Interface().(metav1.TypeMeta)
	objectMeta := val.FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	return fmt.Sprintf("%s-%s.yaml", objectMeta.Name, strings.ToLower(typeMeta.Kind))
}
