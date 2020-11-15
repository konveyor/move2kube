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

package containerizer_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

// Helper functions
var join = filepath.Join

func mustreadyaml(t *testing.T, path string, x interface{}) {
	if err := common.ReadYaml(path, x); err != nil {
		t.Fatalf("Failed to read the testdata at path %q Error: %q", path, err)
	}
}

func setupAssets(t *testing.T) {
	assetsPath, tempPath, err := common.CreateAssetsData()
	if err != nil {
		t.Fatalf("Unable to create the assets directory. Error: %q", err)
	}

	common.TempPath = tempPath
	common.AssetsPath = assetsPath
}

// Tests
func TestS2IGetContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get container for the sample nodejs app", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		// Test data
		testdatapath := "testdata/s2icontainerizer/getcontainer/normal/"
		want := irtypes.Container{}
		mustreadyaml(t, join(testdatapath, "container.yaml"), &want)
		planPath := join(testdatapath, "plan.yaml")
		plan, err := plantypes.ReadPlan(planPath)
		if err != nil {
			t.Fatalf("Failed to read the plan at path %q Error: %q", planPath, err)
		}
		service := plan.Spec.Inputs.Services["nodejs"][0]

		s2icontainerizer := new(containerizer.S2IContainerizer)

		// Test
		cont, err := s2icontainerizer.GetContainer(plan, service)
		if err != nil {
			t.Fatal("Failed to get the container. Error:", err)
		}
		if !cmp.Equal(cont, want) {
			t.Fatalf("Failed to create the container properly. Difference:\n%s", cmp.Diff(want, cont))
		}
	})

	t.Run("get container for the nodejs app when the service is wrong", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		// Test data
		testdatapath := "testdata/s2icontainerizer/getcontainer/incorrectservice/"
		plan := plantypes.Plan{}
		mustreadyaml(t, join(testdatapath, "plan.yaml"), &plan)
		service := plantypes.Service{}
		mustreadyaml(t, join(testdatapath, "service.yaml"), &service)

		s2icontainerizer := new(containerizer.S2IContainerizer)

		// Test
		if _, err := s2icontainerizer.GetContainer(plan, service); err == nil {
			t.Fatal("Should not have succeeded since the service has the incorrect target options.")
		}
	})

	t.Run("get container for the nodejs app when the container build type is wrong", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		// Test data
		testdatapath := "testdata/s2icontainerizer/getcontainer/incorrectbuilder/"
		plan := plantypes.Plan{}
		mustreadyaml(t, join(testdatapath, "plan.yaml"), &plan)
		service := plantypes.Service{}
		mustreadyaml(t, join(testdatapath, "service.yaml"), &service)

		s2icontainerizer := new(containerizer.S2IContainerizer)

		// Test
		if _, err := s2icontainerizer.GetContainer(plan, service); err == nil {
			t.Fatal("Should not have succeeded since the service has the wrong builder type.")
		}
	})
}
