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

package containerexec

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
)

func TestIsBuilderAvailable(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("normal use case", func(t *testing.T) {
		provider := newDockerEngine()
		image := "cloudfoundry/cnb:cflinuxfs3"

		// Test
		if !provider.pullImage(image) {
			t.Fatalf("Failed to find the image %q locally and/or pull it.", image)
		}
	})

	t.Run("normal use case where we get result from cache", func(t *testing.T) {
		provider := newDockerEngine()
		image := "cloudfoundry/cnb:cflinuxfs3"
		want := map[string]bool{}
		want[image] = true

		// Test
		if !provider.pullImage(image) {
			t.Fatalf("Failed to find the builder %q locally and/or pull it.", image)
		}
		if !cmp.Equal(provider.availableImages, want) {
			t.Fatalf("Failed to add the builder %q to the list of available builders. Difference:\n%s", image, cmp.Diff(want, provider.availableImages))
		}
		if !provider.pullImage(image) {
			t.Fatalf("Failed to find the builder %q locally and/or pull it.", image)
		}
	})

	t.Run("check for a non existent image", func(t *testing.T) {
		provider := newDockerEngine()
		image := "this/doesnotexist:foobar"
		if provider.pullImage(image) {
			t.Fatalf("Should not have succeeded. The builder image %q does not exist", image)
		}
	})
}
