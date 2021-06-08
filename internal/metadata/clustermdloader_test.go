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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/metadata"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

func TestUpdatePlan(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("update plan when there are no files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}

		// Test
		if err := loader.UpdatePlan(&p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !cmp.Equal(p, want) {
			t.Fatalf("The updated plan is incorrect. Difference:\n%s", cmp.Diff(want, p))
		}
	})

	t.Run("check if all clusters in constant were loaded", func(t *testing.T) {
		p := plantypes.NewPlan()
		cmMap := new(metadata.ClusterMDLoader).GetClusters(p)

		relInputPath := "clusters/"
		inputPath, err := filepath.Abs(relInputPath)
		if err != nil {
			t.Fatalf("Failed to make the path %s absolute. Error: %q", relInputPath, err)
		}

		yfiles, err := common.GetFilesByExt(inputPath, []string{".yml", ".yaml"})
		if err != nil {
			t.Fatalf("Unable to fetch yaml files and recognize cluster metadata yamls. Error : %q", err)
		}
		objectMetaNames := []string{}
		for _, yfile := range yfiles {
			cm := collecttypes.ClusterMetadata{}
			if common.ReadMove2KubeYaml(yfile, &cm) == nil {
				objectMetaNames = append(objectMetaNames, cm.ObjectMeta.Name)
			}
		}

		//TODO: Read all .yaml files in internal/metadata/clusters, and find the value in metadata.name using say a regex

		// check cluster names in the yaml files are in the cmMap
		for _, clustername := range objectMetaNames {
			if _, ok := cmMap[clustername]; !ok {
				t.Fatal("Missing builtin "+clustername+" cluster metadata. The returned cluster info:", cmMap)
			}
		}

		// check cluster names in the cmap are in the yaml files

		// transform slice into map
		objectMetaNamesMap := make(map[string]string)
		for _, name := range objectMetaNames {
			objectMetaNamesMap[name] = name
		}
		// get names from cmMap
		cmMapKeys := []string{}
		for k := range cmMap {
			cmMapKeys = append(cmMapKeys, k)
		}

		for _, clustername := range cmMapKeys {
			if _, ok := objectMetaNamesMap[clustername]; !ok {
				t.Fatal("Missing builtin " + clustername + " cluster metadata")
			}
		}

	})

	t.Run("update plan with some empty files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}

		// Test
		if err := loader.UpdatePlan(&p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !cmp.Equal(p, want) {
			t.Fatalf("The updated plan is incorrect. Difference:\n%s", cmp.Diff(want, p))
		}
	})

	t.Run("update plan with some invalid files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}

		// Test
		if err := loader.UpdatePlan(&p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !cmp.Equal(p, want) {
			t.Fatalf("The updated plan is incorrect. Difference:\n%s", cmp.Diff(want, p))
		}
	})

	t.Run("update plan with some valid files", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		want := plantypes.NewPlan()
		want.Spec.Configuration.TargetClusters = map[string]string{"name1": "testdata/validfiles/test1.yaml", "name2": "testdata/validfiles/test2.yml"}
		want.Spec.TargetCluster = plantypes.TargetClusterType{Type: "name1"}
		loader := metadata.ClusterMDLoader{}

		// Test
		if err := loader.UpdatePlan(&p); err != nil {
			t.Fatal("Failed to update the plan. Error:", err)
		}
		if !cmp.Equal(p, want) {
			t.Fatalf("The updated plan is incorrect. Difference:\n%s", cmp.Diff(want, p))
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

	t.Run("check default cluster type is valid", func(t *testing.T) {
		p := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}
		cmMap := loader.GetClusters(p)
		if _, ok := cmMap[common.DefaultClusterType]; !ok {
			t.Fatal("Missing builtin "+common.DefaultClusterType+" cluster metadata. The returned cluster info:", cmMap)
		}
	})

	t.Run("get clusters from an empty plan", func(t *testing.T) {
		p := plantypes.NewPlan()
		loader := metadata.ClusterMDLoader{}
		cmMap := loader.GetClusters(p)
		if _, ok := cmMap[common.DefaultClusterType]; !ok {
			t.Fatal("Missing builtin "+common.DefaultClusterType+" cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["Kubernetes"]; !ok {
			t.Fatal("Missing builtin kubernetes cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["Openshift"]; !ok {
			t.Fatal("Missing builtin openshift cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["IBM-IKS"]; !ok {
			t.Fatal("Missing builtin kubernetes cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["IBM-Openshift"]; !ok {
			t.Fatal("Missing builtin openshift cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["AWS-EKS"]; !ok {
			t.Fatal("Missing builtin AWS-EKS cluster metadata. The returned cluster info:", cmMap)
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
		p.Spec.Configuration.TargetClusters = map[string]string{"name1": "testdata/validfiles/test1.yaml", "name2": "testdata/validfiles/test2.yml"}
		loader := metadata.ClusterMDLoader{}
		cmMap := loader.GetClusters(p)
		if _, ok := cmMap["IBM-IKS"]; !ok {
			t.Fatal("Missing builtin IBM-IKS cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["IBM-Openshift"]; !ok {
			t.Fatal("Missing builtin IBM-Openshift cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["AWS-EKS"]; !ok {
			t.Fatal("Missing builtin AWS-EKS cluster metadata. The returned cluster info:", cmMap)
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
		p.Spec.Configuration.TargetClusters = map[string]string{"name1": "testdata/validfilesnostorageclasses/test1.yaml", "name2": "testdata/validfilesnostorageclasses/test2.yml"}
		loader := metadata.ClusterMDLoader{}
		cmMap := loader.GetClusters(p)
		if _, ok := cmMap["IBM-IKS"]; !ok {
			t.Fatal("Missing builtin IBM-IKS cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["IBM-Openshift"]; !ok {
			t.Fatal("Missing builtin IBM-Openshift cluster metadata. The returned cluster info:", cmMap)
		}
		if _, ok := cmMap["AWS-EKS"]; !ok {
			t.Fatal("Missing builtin AWS-EKS cluster metadata. The returned cluster info:", cmMap)
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
