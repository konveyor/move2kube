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
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestPodman(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("normal use case", func(t *testing.T) {
		provider := newPodmanEngine()
		image := "quay.io/konveyor/hello-world:latest"

		if err := provider.pullImage(image); !err {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %v", image, err)
		}
	})

	t.Run("normal use case where we get result from cache", func(t *testing.T) {
		provider := newPodmanEngine()
		image := "quay.io/konveyor/hello-world:latest"

		if err := provider.pullImage(image); !err {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %v", image, err)
		}
		if !provider.availableImages[image] {
			t.Fatalf("Failed to add the image %q to the list of available images", image)
		}
		if err := provider.pullImage(image); !err {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %v", image, err)
		}
	})

	t.Run("check for a non existent image", func(t *testing.T) {
		provider := newPodmanEngine()
		image := "this/doesnotexist:foobar"
		if err := provider.pullImage(image); err {
			t.Fatalf("Should not have succeeded. The image '%s' does not exist", image)
		}
	})

	t.Run("check for a running a container", func(t *testing.T) {
		provider := newPodmanEngine()
		image := "quay.io/konveyor/hello-world:latest"
		if err := provider.pullImage(image); !err {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %v", image, err)
		}

		output, containerStarted, err := provider.RunContainer(image, "", "", "", false)
		if err != nil {
			t.Fatalf("Failed to run the container '%s' locally. Output: %v , containerStarted: %v  Error: %v", image, output, containerStarted, err)
		}
	})

	t.Run("Check for InspectImage functionality ", func(t *testing.T) {
		provider := newPodmanEngine()
		image := "quay.io/konveyor/hello-world:latest"
		if err := provider.pullImage(image); !err {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %v", image, err)
		}
		outputInspect, err := provider.InspectImage(image)
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q . Output: %v", image, err, outputInspect)
		}
		found := false
		for _, i := range outputInspect.RepoTags {
			if strings.HasSuffix(i, "hello-world:latest") {
				found = true
			}
		}
		if !found {
			t.Fatalf("Ispect Repo Tag Mismatch Should be - hello-world:latest got - %+v", outputInspect)
		}
	})
}
