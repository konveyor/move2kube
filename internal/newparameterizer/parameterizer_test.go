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

package newparameterizer_test

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/newparameterizer/applyparameterizers"
	"github.com/konveyor/move2kube/internal/newparameterizer/getparameterizers"
	"github.com/konveyor/move2kube/internal/starlark"
	"github.com/konveyor/move2kube/internal/starlark/gettransformdata"
)

func TestGettingAndParameterizingResources(t *testing.T) {
	relBaseDir := "testdata"
	baseDir, err := filepath.Abs(relBaseDir)
	if err != nil {
		t.Fatalf("Failed to make the base directory %s absolute path. Error: %q", relBaseDir, err)
	}

	parameterizersPath := filepath.Join(baseDir, "parameterizers")
	k8sResourcesPath := filepath.Join(baseDir, "k8s-resources")
	outputPath := t.TempDir()

	filesWritten, err := parameterizeAll(parameterizersPath, k8sResourcesPath, outputPath)
	if err != nil {
		t.Fatalf("Failed to apply all the parameterizations. Error: %q", err)
	}
	if len(filesWritten) != 4 {
		t.Fatalf("Expected %d files to be written. Actual: %d", 4, len(filesWritten))
	}
	wantDataDir := filepath.Join(baseDir, "want")
	for _, fileWritten := range filesWritten {
		filename := filepath.Base(fileWritten)
		wantDataPath := filepath.Join(wantDataDir, filename)
		var wantData interface{}
		if err := common.ReadYaml(wantDataPath, &wantData); err != nil {
			t.Fatalf("Failed to read the test data at path %s . Error: %q", wantDataPath, err)
		}
		var actualData interface{}
		if err := common.ReadYaml(fileWritten, &actualData); err != nil {
			t.Fatalf("Failed to read the parameterized k8s resource at path %s . Error: %q", fileWritten, err)
		}
		if !cmp.Equal(actualData, wantData) {
			t.Fatalf("The file %s is different from expected. Differences:\n%s", filename, cmp.Diff(wantData, actualData))
		}
	}
}

func parameterizeAll(parameterizersPath, k8sResourcesPath, outputPath string) ([]string, error) {
	parameterizers, err := getparameterizers.GetParameterizers(parameterizersPath)
	if err != nil {
		return nil, err
	}
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
