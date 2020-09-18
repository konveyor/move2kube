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

package metadata_test

import (
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/metadata"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

func TestUpdatePlan(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("update plan when there are no files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}
		inputPath := t.TempDir()

		// Test
		if err := loader.UpdatePlan(inputPath, &p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !reflect.DeepEqual(p, want) {
			t.Fatal("The updated plan is incorrect. Expected", want, "Actual:", p)
		}
	})

	t.Run("update plan with some empty files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}
		inputPath := "testdata/emptyfiles"

		// Test
		if err := loader.UpdatePlan(inputPath, &p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !reflect.DeepEqual(p, want) {
			t.Fatal("The updated plan is incorrect. Expected", want, "Actual:", p)
		}
	})

	t.Run("update plan with some invalid files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}
		inputPath := "testdata/invalidfiles"

		// Test
		if err := loader.UpdatePlan(inputPath, &p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !reflect.DeepEqual(p, want) {
			t.Fatal("The updated plan is incorrect. Expected", want, "Actual:", p)
		}
	})

	t.Run("update plan with some valid files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		want.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType] = []string{"testdata/validfiles/test1.yaml", "testdata/validfiles/test2.yml"}
		want.Spec.Outputs.Kubernetes.ClusterType = "name1"
		want.Spec.Outputs.Kubernetes.IgnoreUnsupportedKinds = true
		loader := metadata.ClusterMDLoader{}
		inputPath := "testdata/validfiles"

		// Test
		if err := loader.UpdatePlan(inputPath, &p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !reflect.DeepEqual(p, want) {
			t.Fatal("The updated plan is incorrect. Expected", want, "Actual:", p)
		}
	})
}

func TestLoadToIR(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("load IR with an empty plan", func(t *testing.T) {
		p := plantypes.NewPlan()
		ir := irtypes.NewIR(p)
		loader := metadata.ClusterMDLoader{}
		if err := loader.LoadToIR(p, &ir); err != nil {
			t.Fatal("Failed to load IR. Error:", err)
		}
	})
}

func TestGetClusters(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get clusters from an empty plan", func(t *testing.T) {
		p := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}
		cmMap := loader.GetClusters(p)
		if _, ok := cmMap["Kubernetes"]; !ok {
			t.Fatal("Missing builtin kubernetes cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["Openshift"]; !ok {
			t.Fatal("Missing builtin openshift cluster metadata. The returned cluster info:", cmMap)
		}
		for k, v := range cmMap {
			if v.Kind != string(collecttypes.ClusterMetadataKind) {
				t.Fatal("The kind is incorrect for key", k, "Expected:", collecttypes.ClusterMetadataKind, " Actual:", v.Kind)
			} else if k != v.Name {
				t.Fatal("The cluster metadata was inserted under incorrect key. Expected:", v.Name, "Actual:", k)
			} else if len(v.Spec.StorageClasses) == 0 {
				t.Fatal("There are no storage classes in the cluster metadata. Excpected there to be at least 'default' storage class. Actual:", v.Spec.StorageClasses)
			}
		}
	})

	t.Run("get clusters from a filled plan", func(t *testing.T) {
		p := plantypes.NewPlan()
		p.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType] = []string{"testdata/validfiles/test1.yaml", "testdata/validfiles/test2.yml"}
		loader := metadata.ClusterMDLoader{}
		cmMap := loader.GetClusters(p)
		if _, ok := cmMap["Kubernetes"]; !ok {
			t.Fatal("Missing builtin kubernetes cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["Openshift"]; !ok {
			t.Fatal("Missing builtin openshift cluster metadata. The returned cluster info:", cmMap)
		}
		for k, v := range cmMap {
			if v.Kind != string(collecttypes.ClusterMetadataKind) {
				t.Fatal("The kind is incorrect for key", k, "Expected:", collecttypes.ClusterMetadataKind, " Actual:", v.Kind)
			} else if k != v.Name && !((k == "testdata/validfiles/test1.yaml" || k == "testdata/validfiles/test2.yml") && v.Name == "name1") {
				t.Fatal("The cluster metadata was inserted under incorrect key. Expected the key to be either the context name", v.Name, "or the file path. Actual:", k)
			} else if len(v.Spec.StorageClasses) == 0 {
				t.Fatal("There are no storage classes in the cluster metadata. Excpected there to be at least 'default' storage class. Actual:", v.Spec.StorageClasses)
			}
		}
	})

	t.Run("get clusters from a filled plan", func(t *testing.T) {
		p := plantypes.NewPlan()
		p.Spec.Inputs.TargetInfoArtifacts[plantypes.K8sClusterArtifactType] = []string{"testdata/validfilesnostorageclasses/test1.yaml", "testdata/validfilesnostorageclasses/test2.yml"}
		loader := metadata.ClusterMDLoader{}
		cmMap := loader.GetClusters(p)
		if _, ok := cmMap["Kubernetes"]; !ok {
			t.Fatal("Missing builtin kubernetes cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["Openshift"]; !ok {
			t.Fatal("Missing builtin openshift cluster metadata. The returned cluster info:", cmMap)
		}
		for k, v := range cmMap {
			if v.Kind != string(collecttypes.ClusterMetadataKind) {
				t.Fatal("The kind is incorrect for key", k, "Expected:", collecttypes.ClusterMetadataKind, " Actual:", v.Kind)
			} else if k != v.Name && !((k == "testdata/validfilesnostorageclasses/test1.yaml" || k == "testdata/validfilesnostorageclasses/test2.yml") && v.Name == "name1") {
				t.Fatal("The cluster metadata was inserted under incorrect key. Expected the key to be either the context name", v.Name, "or the file path. Actual:", k)
			} else if len(v.Spec.StorageClasses) == 0 {
				t.Fatal("There are no storage classes in the cluster metadata. Excpected there to be at least 'default' storage class. Actual:", v.Spec.StorageClasses)
			}
		}
	})
}
