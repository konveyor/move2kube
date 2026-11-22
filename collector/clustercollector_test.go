/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package collector

import (
	"reflect"
	"testing"
)

func TestSortVersionList(t *testing.T) {
	c := &ClusterCollector{}

	t.Run("sorts newest first with stable before beta before alpha", func(t *testing.T) {
		vList := []string{"v1alpha1", "v1", "v1beta1", "v2alpha1", "v2"}
		c.sortVersionList(&vList)
		want := []string{"v2", "v2alpha1", "v1", "v1beta1", "v1alpha1"}
		if !reflect.DeepEqual(vList, want) {
			t.Fatalf("expected %v, got %v", want, vList)
		}
	})

	t.Run("keeps versions that semver cannot parse instead of panicking", func(t *testing.T) {
		vList := []string{"v1-alpha", "v1beta1", "v1"}
		c.sortVersionList(&vList)
		want := []string{"v1", "v1beta1", "v1-alpha"}
		if !reflect.DeepEqual(vList, want) {
			t.Fatalf("expected %v, got %v", want, vList)
		}
	})
}

func TestClusterByGroupsAndSortVersions(t *testing.T) {
	c := &ClusterCollector{}
	got := c.clusterByGroupsAndSortVersions([]string{"addon.open-cluster-management.io/v1-alpha", "addon.open-cluster-management.io/v1alpha1"})
	if len(got) != 2 {
		t.Fatalf("expected both group versions to be kept, got %v", got)
	}
}
