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

func TestDockerFileGetContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get container for the sample nodejs app using dockerfile", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		// Test data
		reltestdatapath := "testdata/dockerfilecontainerizer/getcontainer/normal/"
		testdatapath, err := filepath.Abs(reltestdatapath)
		if err != nil {
			t.Fatalf("Failed to make the test data path %q absolute. Error: %q", reltestdatapath, err)
		}
		want := irtypes.Container{}
		mustreadyaml(t, join(testdatapath, "container.yaml"), &want)
		planPath := join(testdatapath, "plan.yaml")
		plan, err := plantypes.ReadPlan(planPath)
		if err != nil {
			t.Fatalf("Failed to read the plan at path %q Error: %q", planPath, err)
		}
		service := plan.Spec.Inputs.Services["dockerfile"][0]

		// Test
		cont, err := new(containerizer.DockerfileContainerizer).GetContainer(plan, service)
		if err != nil {
			t.Fatal("Failed to get the container. Error:", err)
		}
		if filepath.Base(cont.RepoInfo.TargetPath) != want.RepoInfo.TargetPath {
			t.Fatal("The target dockerfile path is incorrect")
		}
		cont.RepoInfo.TargetPath = want.RepoInfo.TargetPath
		if !cmp.Equal(cont, want) {
			t.Fatalf("Failed to create the container properly. Difference:\n%s", cmp.Diff(want, cont))
		}
	})

	t.Run("get container for the dockerfile sample when the service is wrong", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		// Test data
		testdatapath := "testdata/dockerfilecontainerizer/getcontainer/incorrectservice/"
		plan := plantypes.Plan{}
		mustreadyaml(t, join(testdatapath, "plan.yaml"), &plan)
		service := plantypes.Service{}
		mustreadyaml(t, join(testdatapath, "service.yaml"), &service)

		dockerfilecontainerizer := new(containerizer.DockerfileContainerizer)

		// Test
		if _, err := dockerfilecontainerizer.GetContainer(plan, service); err == nil {
			t.Fatal("Should not have succeeded since the service has the incorrect target options.")
		}
	})

	t.Run("get container for the dockerfile sample when the container build type is wrong", func(t *testing.T) {
		// Setup
		setupAssets(t)
		defer os.RemoveAll(common.TempPath)

		// Test data
		testdatapath := "testdata/dockerfilecontainerizer/getcontainer/incorrectbuilder/"
		plan := plantypes.Plan{}
		mustreadyaml(t, join(testdatapath, "plan.yaml"), &plan)
		service := plantypes.Service{}
		mustreadyaml(t, join(testdatapath, "service.yaml"), &service)

		dockerfilecontainerizer := new(containerizer.DockerfileContainerizer)

		// Test
		if _, err := dockerfilecontainerizer.GetContainer(plan, service); err == nil {
			t.Fatal("Should not have succeeded since the service has the wrong builder type.")
		}
	})
}
