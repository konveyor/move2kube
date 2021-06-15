/*
Copyright IBM Corporation 2020, 2021

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

package environment

import (
	"fmt"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/konveyor/move2kube/environment/container"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

var (
	inited        bool
	workingEngine ContainerEngine
)

// Engine defines interface to manage containers
type ContainerEngine interface {
	// RunCmdInContainer runs a container
	RunCmdInContainer(image string, cmd environmenttypes.Command, workingdir string) (stdout, stderr string, exitcode int, err error)
	// InspectImage gets Inspect output for a container
	InspectImage(image string) (dockertypes.ImageInspect, error)
	CopyDirsIntoImage(image, newImageName string, paths map[string]string) (err error)
	CopyDirsIntoContainer(containerID string, paths map[string]string) (err error)
	CopyDirsFromContainer(containerID string, paths map[string]string) (err error)
	BuildImage(image, context, dockerfile string) (err error)
	RemoveImage(image string) (err error)
	CreateContainer(image string) (containerid string, err error)
	StopAndRemoveContainer(containerID string) (err error)
	// RunContainer runs a container from an image
	RunContainer(image string, cmd environmenttypes.Command, volsrc string, voldest string) (output string, containerStarted bool, err error)
}

func initContainerEngine() (err error) {
	workingEngine, err = container.NewDockerEngine()
	if err != nil {
		logrus.Debugf("Unable to use docker : %s", err)
		return err
	}
	if workingEngine == nil {
		err := fmt.Errorf("no working container runtime available")
		logrus.Errorf("%s", err)
		return err
	}
	return nil
}

// GetContainerEngine gets a working container engine
func GetContainerEngine() ContainerEngine {
	if !inited {
		initContainerEngine()
		inited = true
	}
	return workingEngine
}
