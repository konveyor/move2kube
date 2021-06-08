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

package api_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/konveyor/move2kube/api"
	"github.com/konveyor/move2kube/assets"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/translator"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

func setupAssets(t *testing.T) {
	assetsPath, tempPath, err := common.CreateAssetsData(assets.Tar)
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
		if err := want.SetRootDir(inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}
		translator.Init(common.AssetsDir)

		// Test
		p := api.CreatePlan(inputPath, "", prjName)
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
		if err := want.SetRootDir(inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}
		translator.Init(common.AssetsPath)

		// Test
		p := api.CreatePlan(inputPath, "", prjName)
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
		want, err := plantypes.ReadPlan(testDataPlanPath)
		if err != nil {
			t.Fatalf("Cannot read the plan at path %q Error: %q", testDataPlanPath, err)
		}
		translator.Init(common.AssetsPath)

		// Test
		actual := api.CreatePlan(inputPath, "", prjName)

		if !cmp.Equal(actual, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to create the plan properly. Difference:\n%s", cmp.Diff(want, actual, cmpopts.EquateEmpty()))
		}
	})
}
