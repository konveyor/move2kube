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

package move2kube_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/move2kube"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

func setupAssets(t *testing.T) {
	assetsPath, tempPath, err := common.CreateAssetsData()
	if err != nil {
		t.Fatalf("Unable to create the assets directory. Error: %q", err)
	}

	common.TempPath = tempPath
	common.AssetsPath = assetsPath
}

func TestCreatePlan(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("create plan for empty app and without the cache folder", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		inputPath := t.TempDir()
		prjName := "project1"

		want := plantypes.NewPlan()
		want.Name = prjName
		if err := move2kube.SetRootDir(&want, inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}

		// Test
		p := move2kube.CreatePlan(inputPath, prjName)
		if !cmp.Equal(p, want) {
			t.Fatalf("Failed to create the plan properly. Difference:\n%s", cmp.Diff(want, p))
		}
	})

	t.Run("create plan for empty app", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		inputPath := t.TempDir()
		prjName := "project1"

		// If the cache folder exists delete it
		if _, err := os.Stat(common.AssetsPath); !os.IsNotExist(err) {
			if err := os.RemoveAll(common.AssetsPath); err != nil {
				t.Fatal("Failed to remove the cache folder from previous runs. Error:", err)
			}
		}
		// Create the cache folder (.m2k) it expects to find.
		if err := os.MkdirAll(common.AssetsPath, os.ModeDir|os.ModePerm); err != nil {
			t.Fatal("Failed to make the common.AssetsPath directory:", common.AssetsPath, "Error:", err)
		}
		defer os.RemoveAll(common.AssetsPath)

		want := plantypes.NewPlan()
		want.Name = prjName
		if err := move2kube.SetRootDir(&want, inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}

		// Test
		p := move2kube.CreatePlan(inputPath, prjName)
		if !cmp.Equal(p, want) {
			t.Fatalf("Failed to create the plan properly. Difference:\n%s", cmp.Diff(want, p))
		}
	})

	t.Run("create plan for a simple nodejs app", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		prjName := "nodejs-app"
		relInputPath := "../../samples/nodejs"
		inputPath, err := filepath.Abs(relInputPath)
		if err != nil {
			t.Fatalf("Failed to make the path %q Error: %q", relInputPath, err)
		}

		testDataPlanPath := "testdata/expectedplanfornodejsapp.yaml"
		want, err := move2kube.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Cannot read the plan at path %q Error: %q", testDataPlanPath, err)
		}

		// Test
		actual := move2kube.CreatePlan(inputPath, prjName)
		for _, services := range actual.Spec.Inputs.Services {
			for i := range services {
				services[i].RepoInfo = plantypes.RepoInfo{}
			}
		}

		if !cmp.Equal(actual, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to create the plan properly. Difference:\n%s", cmp.Diff(want, actual, cmpopts.EquateEmpty()))
		}
	})
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

		plan, err := move2kube.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}
		wantBytes, err := ioutil.ReadFile(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}

		// Test
		if err := move2kube.WritePlan(outputPath, plan); err != nil {
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

		newRootDir, err := filepath.Abs(relNewRootDir)
		if err != nil {
			t.Fatalf("Failed to make the new root directory path %q absolute. Error: %q", relNewRootDir, err)
		}
		testDataPlanPath, err := filepath.Abs(relTestDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to make the test data plan path %q absolute. Error: %q", relTestDataPlanPath, err)
		}

		plan, err := move2kube.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}

		// Test
		if err := move2kube.SetRootDir(&plan, newRootDir); err != nil {
			t.Fatalf("Failed to set the root directory properly. Error: %q", err)
		}
		if plan.Spec.Inputs.RootDir != newRootDir {
			t.Fatalf("Failed to set the root directory properly. Expected: %s Actual: %s", newRootDir, plan.Spec.Inputs.RootDir)
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

		plan, err := move2kube.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q Error: %q", testDataPlanPath, err)
		}
		planCopy, err := move2kube.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Failed to read the test data plan at path %q , even though it succeeded the first time. Error: %q", testDataPlanPath, err)
		}

		// Test
		// Set to the new root
		if err := move2kube.SetRootDir(&plan, newRootDir); err != nil {
			t.Fatalf("Failed to set the root directory properly. Error: %q", err)
		}
		if plan.Spec.Inputs.RootDir != newRootDir {
			t.Fatalf("Failed to set the root directory properly. Expected: %s Actual: %s", newRootDir, plan.Spec.Inputs.RootDir)
		}
		// Reset to the old root
		if err := move2kube.SetRootDir(&plan, oldRootDir); err != nil {
			t.Fatalf("Failed to set the root directory properly the 2nd time. Error: %q", err)
		}
		if !cmp.Equal(plan, planCopy) {
			t.Fatalf("Failed to reset the root directory to the old root directory. Difference:\n%s", cmp.Diff(planCopy, plan))
		}
	})
}
