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

package transformer_test

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/transformer/gettransformdata"
	"github.com/konveyor/move2kube/transformer/runtransforms"
	"github.com/konveyor/move2kube/transformer/transformations"
	log "github.com/sirupsen/logrus" // TODO
)

// var (
// 	answers = map[string]interface{}{}
// )

func TestGettingAndTransformingResources(t *testing.T) {
	relBaseDir := "testdata"
	baseDir, err := filepath.Abs(relBaseDir)
	if err != nil {
		t.Fatalf("Failed to make the base directory %s absolute path. Error: %q", relBaseDir, err)
	}

	qaengine.AddEngine(qaengine.NewDefaultEngine())
	qaengine.SetupConfigFile(t.TempDir(), nil, []string{filepath.Join(baseDir, "m2kconfig.yaml")}, nil)

	transformsPath := filepath.Join(baseDir, "transforms")
	transformsPaths := []string{
		transformsPath + "/t1.star",
		transformsPath + "/t2.star",
	}
	k8sResourcesPath := filepath.Join(baseDir, "k8s-resources")
	outputPath := t.TempDir()

	filesWritten, err := transformAll(transformsPaths, k8sResourcesPath, outputPath)
	if err != nil {
		t.Fatalf("Failed to apply all the transformations. Error: %q", err)
	}
	if len(filesWritten) != 3 {
		t.Fatalf("Expected %d files to be written. Actual: %d", 3, len(filesWritten))
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
			t.Fatalf("Failed to read the transformed k8s resource at path %s . Error: %q", fileWritten, err)
		}
		if !cmp.Equal(actualData, wantData) {
			t.Fatalf("Failed to transform the k8s resource %s properly. Differences:\n%s", filename, cmp.Diff(wantData, actualData))
		}
	}
}

func transformAll(transformsPaths []string, k8sResourcesPath, outputPath string) ([]string, error) {
	log.Trace("start TransformAll")
	defer log.Trace("end TransformAll")
	transforms, err := transformations.GetTransformsFromPathsUsingDefaults(transformsPaths)
	if err != nil {
		return nil, err
	}
	k8sResources, err := gettransformdata.GetK8sResources(k8sResourcesPath)
	if err != nil {
		return nil, err
	}
	transformedK8sResources, err := runtransforms.ApplyTransforms(transforms, k8sResources)
	if err != nil {
		return nil, err
	}
	return starlark.WriteResources(transformedK8sResources, outputPath)
}

// func myDynamicAskQuestion(questionObjI interface{}) (interface{}, error) {
// 	log.Trace("start myDynamicAskQuestion")
// 	defer log.Trace("end myDynamicAskQuestion")
// 	questionObj, ok := questionObjI.(types.MapT)
// 	if !ok {
// 		return nil, fmt.Errorf("Excpted questions to be of map type. Actual value is %+v of type %T", questionObjI, questionObjI)
// 	}
// 	qakeyI, ok := questionObj["key"]
// 	if !ok {
// 		return nil, fmt.Errorf("The key 'key' is missing from the question object %+v", questionObj)
// 	}
// 	qakey, ok := qakeyI.(string)
// 	if !ok {
// 		return nil, fmt.Errorf("The key 'key' is not a string. The question object %+v", questionObj)
// 	}
// 	descI, ok := questionObj["description"]
// 	if !ok {
// 		return nil, fmt.Errorf("The key 'description' is missing from the question object %+v", questionObj)
// 	}
// 	desc, ok := descI.(string)
// 	if !ok {
// 		return nil, fmt.Errorf("The key 'description' is not a string. The question object %+v", questionObj)
// 	}
// 	defaultAnswer := ""
// 	defaultAnswerI, ok := questionObj["default"]
// 	if ok {
// 		newDefaultAnswer, ok := defaultAnswerI.(string)
// 		if !ok {
// 			return nil, fmt.Errorf("The key 'default' is not a string. The question object %+v", questionObj)
// 		}
// 		defaultAnswer = newDefaultAnswer
// 	}
// 	hints := []string{}
// 	log.Debugf("key %+v desc %+v hints %+v default %+v", qakey, desc, hints, defaultAnswer)
// 	answer := fmt.Sprintf("dynamic question: [%s]", qakey)
// 	answers[qakey] = answer
// 	return answer, nil
// }
