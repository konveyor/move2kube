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
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/move2kube"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

func TestCreatePlan(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("create plan for empty app and without the cache folder", func(t *testing.T) {
		// Setup
		inputPath := t.TempDir()
		prjName := "project1"

		want := plantypes.NewPlan()
		want.Name = prjName
		want.Spec.Inputs.RootDir = inputPath

		// Test
		p := move2kube.CreatePlan(inputPath, prjName)
		if !reflect.DeepEqual(p, want) {
			t.Fatalf("Failed to create the plan properly. Difference:\n%s", cmp.Diff(want, p))
		}
	})

	t.Run("create plan for empty app", func(t *testing.T) {
		// Setup
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
		want.Spec.Inputs.RootDir = inputPath

		// Test
		p := move2kube.CreatePlan(inputPath, prjName)
		if !reflect.DeepEqual(p, want) {
			t.Fatalf("Failed to create the plan properly. Difference:\n%s", cmp.Diff(want, p))
		}
	})

	t.Run("create plan for a simple nodejs app", func(t *testing.T) {
		// Setup
		inputPath := "../../samples/nodejs"
		prjName := "nodejs-app"

		// If the cache folder exists delete it
		if _, err := os.Stat(common.AssetsPath); !os.IsNotExist(err) {
			if err := os.RemoveAll(common.AssetsPath); err != nil {
				t.Fatal("Failed to remove the cache folder from previous runs. Error:", err)
			}
		}
		// Create the cache folder (.m2k) it expects to find.
		if err := os.Mkdir(common.AssetsPath, os.ModeDir|os.ModePerm); err != nil {
			t.Fatal("Failed to make the common.AssetsPath directory:", common.AssetsPath, "Error:", err)
		}
		defer os.RemoveAll(common.AssetsPath)

		want := plantypes.NewPlan()
		if err := common.ReadYaml("testdata/expectedplanfornodejsapp.yaml", &want); err != nil {
			t.Fatal("failed to read the expected output plan from yaml. Error:", err)
		}

		// Test
		p := move2kube.CreatePlan(inputPath, prjName)
		if !reflect.DeepEqual(p, want) {
			t.Fatalf("Failed to create the plan properly. Difference:\n%s", cmp.Diff(want, p))
		}
	})
}
