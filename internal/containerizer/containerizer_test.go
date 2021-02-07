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
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/types/plan"
	plantypes "github.com/konveyor/move2kube/types/plan"

	"github.com/konveyor/move2kube/internal/containerizer"
)

func TestComesBefore(t *testing.T) {
	serviceOptions := []plantypes.Service{
		plan.Service{ContainerBuildType: plantypes.ReuseContainerBuildTypeValue},
		plan.Service{ContainerBuildType: plantypes.DockerFileContainerBuildTypeValue},
	}
	want := []plantypes.Service{
		plan.Service{ContainerBuildType: plantypes.DockerFileContainerBuildTypeValue},
		plan.Service{ContainerBuildType: plantypes.ReuseContainerBuildTypeValue},
	}
	// sort the service options in order of priority
	sort.Slice(serviceOptions, func(i, j int) bool {
		return containerizer.ComesBefore(serviceOptions[i].ContainerBuildType, serviceOptions[j].ContainerBuildType)
	})
	if !cmp.Equal(serviceOptions, want) {
		t.Fatalf("Failed to sort the service options properly. Difference:\n%s", cmp.Diff(want, serviceOptions))
	}
}
