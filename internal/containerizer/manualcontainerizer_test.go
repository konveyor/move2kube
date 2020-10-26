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
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

func TestManualGetContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get container for the sample nodejs app", func(t *testing.T) {
		// Setup

		// Test data
		testdatapath := "testdata/manualcontainerizer/getcontainer/normal/"
		plan := plantypes.Plan{}
		mustreadyaml(t, join(testdatapath, "plan.yaml"), &plan)
		service := plantypes.Service{}
		mustreadyaml(t, join(testdatapath, "service.yaml"), &service)
		want := irtypes.NewContainer(plantypes.ManualContainerBuildTypeValue, service.Image, true)

		manualcontainerizer := new(containerizer.ManualContainerizer)

		// Test
		cont, err := manualcontainerizer.GetContainer(plan, service)
		if err != nil {
			t.Fatal("Failed to get the container. Error:", err)
		}
		if !cmp.Equal(cont, want) {
			t.Fatalf("Failed to create the container properly. Difference:\n%s", cmp.Diff(want, cont))
		}
	})

	t.Run("get container for the nodejs app when the service is wrong", func(t *testing.T) {
		// Setup

		// Test data
		testdatapath := "testdata/manualcontainerizer/getcontainer/incorrectservice/"
		plan := plantypes.Plan{}
		mustreadyaml(t, join(testdatapath, "plan.yaml"), &plan)
		service := plantypes.Service{}
		mustreadyaml(t, join(testdatapath, "service.yaml"), &service)

		manualcontainerizer := new(containerizer.ManualContainerizer)

		// Test
		if _, err := manualcontainerizer.GetContainer(plan, service); err == nil {
			t.Fatal("Should not have succeeded since the service has the incorrect target options.")
		}
	})
}
