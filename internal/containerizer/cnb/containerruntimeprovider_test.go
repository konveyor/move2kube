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

package cnb

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
)

func TestIsBuilderAvailable(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("normal use case", func(t *testing.T) {
		provider := containerRuntimeProvider{}
		builder := "cloudfoundry/cnb:cflinuxfs3"

		// Test
		if !provider.isBuilderAvailable(builder) {
			t.Fatalf("Failed to find the builder %q locally and/or pull it.", builder)
		}
	})

	t.Run("normal use case where we get result from cache", func(t *testing.T) {
		provider := containerRuntimeProvider{}
		builder := "cloudfoundry/cnb:cflinuxfs3"
		want := []string{builder}

		// Test
		if !provider.isBuilderAvailable(builder) {
			t.Fatalf("Failed to find the builder %q locally and/or pull it.", builder)
		}
		if !reflect.DeepEqual(availableBuilders, want) {
			t.Fatalf("Failed to add the builder %q to the list of available builders. Difference:\n%s", builder, cmp.Diff(want, availableBuilders))
		}
		if !provider.isBuilderAvailable(builder) {
			t.Fatalf("Failed to find the builder %q locally and/or pull it.", builder)
		}
	})

	t.Run("check for a non existent image", func(t *testing.T) {
		provider := containerRuntimeProvider{}
		builder := "this/doesnotexist:foobar"
		if provider.isBuilderAvailable(builder) {
			t.Fatalf("Should not have succeeded. The builder image %q does not exist", builder)
		}
	})
}
