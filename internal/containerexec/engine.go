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
	"github.com/docker/docker/api/types"
)

const (
	testimage = "quay.io/konveyor/hello-world"
)

var (
	inited               = false
	workingEngine Engine = nil
	engines              = []Engine{newDockerEngine(), newPodmanEngine()}
)

func initContainerEngine() {
	for _, e := range engines {
		_, _, err := e.RunContainer(testimage, "", "", "")
		if err != nil {
			continue
		}
		workingEngine = e
		break
	}
}

// GetEngine gets a working container engine
func GetEngine() Engine {
	if !inited {
		initContainerEngine()
		inited = true
	}
	return workingEngine
}

// Engine defines interface to manage containers
type Engine interface {
	// RunContainer runs a container
	RunContainer(image string, cmd string, volsrc string, voldest string) (output string, containerStarted bool, err error)
	// InspectImage gets Inspect output for a container
	InspectImage(image string) (types.ImageInspect, error)
}
