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

package plan_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/common"
	plantypes "github.com/konveyor/move2kube/types/plan"
	yaml "gopkg.in/yaml.v3"
)

func setupAssets(t *testing.T) {
	assetsPath, tempPath, err := common.CreateAssetsData()
	if err != nil {
		t.Fatalf("Unable to create the assets directory. Error: %q", err)
	}

	common.TempPath = tempPath
	common.AssetsPath = assetsPath
}

func TestWritePlan(t *testing.T) {
	t.Run("read the plan and write it out again", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		tmpDir := t.TempDir()
		outputPath := filepath.Join(tmpDir, "actual.yaml")
		relTestDataPlanPath := "testdata/setrootdir/nodejsplan.yaml"
		testDataPlanPath, err := filepath.Abs(relTestDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to make the test data plan path %q absolute. Error: %q", relTestDataPlanPath, err)
		}

		plan, err := plantypes.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}
		wantBytes, err := ioutil.ReadFile(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}

		// Test
		if err := plantypes.WritePlan(outputPath, plan); err != nil {
			t.Fatalf("Failed to write the plan to the path %q Error %q", outputPath, err)
		}

		actualBytes, err := ioutil.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read the plan we wrote at path %q Error: %q", outputPath, err)
		}

		if !cmp.Equal(string(actualBytes), string(wantBytes)) {
			t.Fatalf("Failed to reset the root directory to the old root directory. Difference:\n%s", cmp.Diff(string(wantBytes), string(actualBytes)))
		}
	})
}

func TestSetRootDir(t *testing.T) {
	t.Run("set a new root directory", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		relNewRootDir := "new/root/directory"
		relTestDataPlanPath := "testdata/setrootdir/nodejsplan.yaml"
		testDataTemplatePlanPath := "testdata/setrootdir/templatizednodejsplan.yaml"

		pwd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get the current working directory. Error: %q", err)
		}

		newRootDir, err := filepath.Abs(relNewRootDir)
		if err != nil {
			t.Fatalf("Failed to make the new root directory path %q absolute. Error: %q", relNewRootDir, err)
		}
		testDataPlanPath, err := filepath.Abs(relTestDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to make the test data plan path %q absolute. Error: %q", relTestDataPlanPath, err)
		}

		plan, err := plantypes.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}
		templateBytes, err := ioutil.ReadFile(testDataTemplatePlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data templatized plan at path %q Error: %q", testDataTemplatePlanPath, err)
		}
		wantYaml, err := common.GetStringFromTemplate(string(templateBytes), struct {
			PWD     string
			TempDir string
		}{
			PWD:     pwd,
			TempDir: common.TempPath,
		})
		if err != nil {
			t.Fatalf("Failed to fill the template. Error: %q", err)
		}
		want := plantypes.Plan{}
		if err := yaml.Unmarshal([]byte(wantYaml), &want); err != nil {
			t.Fatalf("Error: %q", err)
		}

		// Test
		if err := plan.SetRootDir(newRootDir); err != nil {
			t.Fatalf("Failed to set the root directory properly. Error: %q", err)
		}
		if plan.Spec.RootDir != newRootDir {
			t.Fatalf("Failed to set the root directory properly. Expected: %s Actual: %s", newRootDir, plan.Spec.RootDir)
		}
		if !cmp.Equal(plan, want) {
			t.Fatalf("Failed to udpate the paths with the new root directory correctly. Difference:\n%s", cmp.Diff(want, plan))
		}
	})

	t.Run("set a new root directory and then reset to the old root directory", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		relOldRootDir := "../../samples/nodejs"
		relNewRootDir := "new/root/directory"
		relTestDataPlanPath := "testdata/setrootdir/nodejsplan.yaml"

		oldRootDir, err := filepath.Abs(relOldRootDir)
		if err != nil {
			t.Fatalf("Failed to make the old root directory path %q absolute. Error: %q", relOldRootDir, err)
		}
		newRootDir, err := filepath.Abs(relNewRootDir)
		if err != nil {
			t.Fatalf("Failed to make the new root directory path %q absolute. Error: %q", relNewRootDir, err)
		}
		testDataPlanPath, err := filepath.Abs(relTestDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to make the test data plan path %q absolute. Error: %q", relTestDataPlanPath, err)
		}

		plan, err := plantypes.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}
		planCopy, err := plantypes.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q , even though it succeeded the first time. Error: %q", testDataPlanPath, err)
		}

		// Test
		// Set to the new root
		if err := plan.SetRootDir(newRootDir); err != nil {
			t.Fatalf("Failed to set the root directory properly. Error: %q", err)
		}
		if plan.Spec.RootDir != newRootDir {
			t.Fatalf("Failed to set the root directory properly. Expected: %s Actual: %s", newRootDir, plan.Spec.RootDir)
		}
		// Reset to the old root
		if err := plan.SetRootDir(oldRootDir); err != nil {
			t.Fatalf("Failed to set the root directory properly the 2nd time. Error: %q", err)
		}
		if !cmp.Equal(plan, planCopy) {
			t.Fatalf("Failed to reset the root directory to the old root directory. Difference:\n%s", cmp.Diff(planCopy, plan))
		}
	})
}
