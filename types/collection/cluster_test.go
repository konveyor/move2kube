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

package collection_test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/types"
	"github.com/konveyor/move2kube/types/collection"
)

func TestMerge(t *testing.T) {
	t.Run("merging 2 empty metadatas", func(t *testing.T) {
		cmeta1 := collection.NewClusterMetadata("")
		cmeta1.Kind = ""
		cmeta2 := collection.NewClusterMetadata("")
		cmeta2.Kind = ""
		want := collection.NewClusterMetadata("")
		want.Kind = ""
		if merged := cmeta1.Merge(cmeta2); !merged || !reflect.DeepEqual(cmeta1, want) {
			t.Fatalf("Failed to merge ClusterMetadata properly. Difference:\n%s:", cmp.Diff(want, cmeta1))
		}
	})

	t.Run("merging a non empty metadata into an empty metadata", func(t *testing.T) {
		cmeta1 := collection.NewClusterMetadata("")
		cmeta1.Kind = ""

		cmeta2 := collection.NewClusterMetadata("ctxname1")

		want := collection.NewClusterMetadata("")
		want.Name = "ctxname1"
		want.Spec.StorageClasses = []string{"default"}

		if merged := cmeta1.Merge(cmeta2); !merged || !reflect.DeepEqual(cmeta1, want) {
			t.Fatalf("Failed to merge ClusterMetadata properly. Difference:\n%s:", cmp.Diff(want, cmeta1))
		}
	})

	t.Run("merging metadata with different kinds", func(t *testing.T) {
		cmeta1 := collection.NewClusterMetadata("")
		cmeta1.Kind = "kind1"

		cmeta2 := collection.NewClusterMetadata("")
		cmeta2.Kind = "kind2"

		if merged := cmeta1.Merge(cmeta2); merged {
			t.Fatal("Should not have merged metadata with different kinds. The kinds are", cmeta1.Kind, "and", cmeta2.Kind)
		}
	})

	t.Run("merging version maps from filled metadata into filled metadata", func(t *testing.T) {
		key1 := "key1"
		val1 := []string{"1.0.0", "1.1.0", "1.1.1"}
		key2 := "key2"
		val2 := []string{"2.0.0", "2.2.0", "2.2.2"}

		cmeta1 := collection.NewClusterMetadata("")
		cmeta1.Spec.APIKindVersionMap = map[string][]string{key1: val1}

		cmeta2 := collection.NewClusterMetadata("")
		cmeta2.Spec.APIKindVersionMap = map[string][]string{key1: val2, key2: val2}

		want := collection.NewClusterMetadata("")
		want.Spec.StorageClasses = []string{"default"}
		want.Spec.APIKindVersionMap = map[string][]string{key1: val2}

		if merged := cmeta1.Merge(cmeta2); !merged || !reflect.DeepEqual(cmeta1, want) {
			t.Fatalf("Failed to merge ClusterMetadata properly. Difference:\n%s:", cmp.Diff(want, cmeta1))
		}
	})

	t.Run("merging storage classes from filled metadata into filled metadata", func(t *testing.T) {
		cmeta1 := collection.NewClusterMetadata("")
		cmeta1.Spec.StorageClasses = []string{"111", "222", "333"}

		cmeta2 := collection.NewClusterMetadata("")
		cmeta2.Spec.StorageClasses = []string{"222", "333", "444"}

		want := collection.NewClusterMetadata("")
		want.Spec.StorageClasses = []string{"222", "333"}

		if merged := cmeta1.Merge(cmeta2); !merged || !reflect.DeepEqual(cmeta1, want) {
			t.Fatalf("Failed to merge ClusterMetadata properly. Difference:\n%s:", cmp.Diff(want, cmeta1))
		}
	})
}

func TestGetSupportedVersions(t *testing.T) {
	t.Run("get nil for non existent key", func(t *testing.T) {
		key1 := "foobar_non_existent_key"
		cmeta := collection.NewClusterMetadata("")
		if arr := cmeta.Spec.GetSupportedVersions(key1); arr != nil {
			t.Fatal("The method should have returned nil since the key", key1, "is not present in the APIKindVersionMap.")
		}
	})

	t.Run("get nil for key with empty list of supported versions", func(t *testing.T) {
		key1 := "key1"
		cmeta := collection.NewClusterMetadata("")
		cmeta.Spec.APIKindVersionMap = map[string][]string{key1: {}}
		if arr := cmeta.Spec.GetSupportedVersions(key1); arr != nil {
			t.Fatal("The method should have returned nil since the supported versions for the", key1, "is an empty list.")
		}
	})

	t.Run("get a list of versions for a valid key", func(t *testing.T) {
		key1 := "key1"
		val1 := []string{"0.1.0", "0.1.1", "1.2.3"}
		cmeta := collection.NewClusterMetadata("")
		cmeta.Spec.APIKindVersionMap = map[string][]string{key1: val1}
		if arr := cmeta.Spec.GetSupportedVersions(key1); arr == nil {
			t.Fatal("The method did not return the correct list of supported versions for the", key1, "Expected:", val1, "Actual:", arr)
		}
	})
}

func TestNewClusterMetadata(t *testing.T) {
	cmeta := collection.NewClusterMetadata("")
	if cmeta.Kind != string(collection.ClusterMetadataKind) || cmeta.APIVersion != types.SchemeGroupVersion.String() {
		t.Fatal("Failed to initialize ClusterMetadata properly.")
	}
}
