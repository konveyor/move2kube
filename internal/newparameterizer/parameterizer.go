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

package newparameterizer

import (
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/newparameterizer/applyparameterizers"
	"github.com/konveyor/move2kube/internal/newparameterizer/getparameterizers"
	"github.com/konveyor/move2kube/internal/newparameterizer/types"
	"github.com/konveyor/move2kube/internal/starlark"
	"github.com/konveyor/move2kube/internal/starlark/gettransformdata"
)

// ParameterizeAll parameterizes the k8s yamls in the source path using the
// parameterizers found in the parameterizers path and outputs a helm chart
func ParameterizeAll(parameterizersPath, k8sResourcesPath, outputPath string) ([]string, error) {
	parameterizers, err := getparameterizers.GetParameterizers(parameterizersPath)
	if err != nil {
		return nil, err
	}
	return ParameterizeAllHelper(parameterizers, k8sResourcesPath, outputPath)
}

// ParameterizeAllPaths is same as ParameterizeAll but takes a list of absolute paths to parameterizers instead
func ParameterizeAllPaths(parameterizersPath []string, k8sResourcesPath, outputPath string) ([]string, error) {
	parameterizers, err := getparameterizers.GetParameterizersFromPaths(parameterizersPath)
	if err != nil {
		return nil, err
	}
	return ParameterizeAllHelper(parameterizers, k8sResourcesPath, outputPath)
}

// ParameterizeAllHelper is same as ParameterizeAll but takes a list of parameterizers instead
func ParameterizeAllHelper(parameterizers []types.ParameterizerT, k8sResourcesPath, outputPath string) ([]string, error) {
	k8sResources, err := gettransformdata.GetK8sResources(k8sResourcesPath)
	if err != nil {
		return nil, err
	}
	parameterizedK8sResources, values, err := applyparameterizers.ApplyParameterizers(parameterizers, k8sResources)
	if err != nil {
		return nil, err
	}
	filesWritten, err := starlark.WriteResources(parameterizedK8sResources, outputPath)
	if err != nil {
		return filesWritten, err
	}
	valuesPath := filepath.Join(outputPath, "values.yaml")
	if err := common.WriteYaml(valuesPath, values); err != nil {
		return nil, err
	}
	filesWritten = append(filesWritten, valuesPath)
	return filesWritten, nil
}
