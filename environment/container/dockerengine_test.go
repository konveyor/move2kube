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

package container

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestIsBuilderAvailable(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("normal use case", func(t *testing.T) {
		provider, _ := NewDockerEngine()
		image := "quay.io/konveyor/move2kube"

		// Test
		if !provider.pullImage(image) {
			t.Fatalf("Failed to find the image %q locally and/or pull it.", image)
		}
	})

	t.Run("normal use case where we get result from cache", func(t *testing.T) {
		provider, _ := NewDockerEngine()
		image := "quay.io/konveyor/move2kube"

		// Test
		if !provider.pullImage(image) {
			t.Fatalf("Failed to find the image %q locally and/or pull it.", image)
		}
		if !provider.availableImages[image] {
			t.Fatalf("Failed to add the image %q to the list of available images", image)
		}
		if !provider.pullImage(image) {
			t.Fatalf("Failed to find the image %q locally and/or pull it.", image)
		}
	})

	t.Run("check for a non existent image", func(t *testing.T) {
		provider, _ := NewDockerEngine()
		image := "this/doesnotexist:foobar"
		if provider.pullImage(image) {
			t.Fatalf("Should not have succeeded. The image %q does not exist", image)
		}
	})
}
