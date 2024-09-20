/*
 *  Copyright IBM Corporation 2020, 2021
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
	"errors"
	"fmt"
	"io/fs"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

var (
	inited        bool
	enabled       bool
	workingEngine ContainerEngine
	// ErrNoContainerRuntime is an error that indicates that no container runtime was found (Docker, Podman, etc.).
	ErrNoContainerRuntime = errors.New("no working container runtime found")
)

// ContainerEngine defines interface to manage containers
type ContainerEngine interface {
	// RunCmdInContainer runs a container
	RunCmdInContainer(image string, cmd environmenttypes.Command, workingdir string, env []string) (stdout, stderr string, exitcode int, err error)
	// InspectImage gets Inspect output for a container
	InspectImage(image string) (dockertypes.ImageInspect, error)
	// TODO: Change paths from map to array
	CopyDirsIntoImage(image, newImageName string, paths map[string]string, copyUIDGID bool) (err error)
	CopyDirsIntoContainer(containerID string, paths map[string]string, copyUIDGID bool) (err error) // See here
	CopyDirsFromContainer(containerID string, paths map[string]string) (err error)
	BuildImage(image, context, dockerfile string) (err error)
	RemoveImage(image string) (err error)
	CreateContainer(container environmenttypes.Container) (containerid string, err error)
	StopAndRemoveContainer(containerID string) (err error)
	// RunContainer runs a container from an image
	RunContainer(image string, cmd environmenttypes.Command, volsrc string, voldest string, copyUIDGID bool) (output string, containerStarted bool, err error)
	Stat(containerID, name string) (fs.FileInfo, error)
}

func initContainerEngine() (err error) {
	logrus.Trace("initContainerEngine start")
	defer logrus.Trace("initContainerEngine end")
	workingEngine, err = newDockerEngine()
	if err != nil {
		return fmt.Errorf("failed to use docker as the container engine. Error: %w", err)
	}
	//TODO: Add Support for podman
	if workingEngine == nil {
		return fmt.Errorf("no working container runtime available")
	}
	return nil
}

// GetContainerEngine gets a working container engine
func GetContainerEngine(spawnContainers bool) (ContainerEngine, error) {
	logrus.Trace("GetContainerEngine start")
	defer logrus.Trace("GetContainerEngine end")
	if !inited {
		enabled = qaengine.FetchBoolAnswer(
			common.ConfigSpawnContainersKey,
			"Allow spawning containers?",
			[]string{"If this setting is set to false, those transformers that rely on containers will not work."},
			spawnContainers,
			nil,
		)
		if enabled {
			if err := initContainerEngine(); err != nil {
				return nil, fmt.Errorf("failed to initialize the container engine. Error: %w", err)
			}
		}
		inited = true
	}
	if workingEngine == nil {
		return nil, ErrNoContainerRuntime
	}
	return workingEngine, nil
}

// IsDisabled returns whether the container environment is disabled
func IsDisabled() bool {
	return !enabled
}
