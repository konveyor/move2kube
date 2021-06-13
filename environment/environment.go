/*
Copyright IBM Corporation 2021

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
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"
)

type Environment struct {
	Name      string
	Paths     map[string]string
	Container ContainerEnvironment
}

// Container stores container based execution information
type Container struct {
	Image          string         `yaml:"image"`
	ContainerBuild ContainerBuild `yaml:"build"`
}

type ContainerBuild struct {
	Dockerfile string `yaml:"dockerfile"` // Default : Look for Dockerfile in the same folder
	Context    string `yaml:"context"`    // Default : Same folder as the yaml
}

type ContainerEnvironment struct {
	ImageName     string
	ImageWithData string
	PID           int
	Root          string
}

func NewEnvironment(name string, paths map[string]string, container Container) (env Environment, err error) {
	env = Environment{
		Name:  name,
		Paths: paths,
	}
	if container.Image != "" {
		containerEnvironment := ContainerEnvironment{
			ImageName: container.Image,
		}
		envVariableName := common.MakeStringEnvNameCompliant(container.Image)
		// Check if image is part of the current environment.
		// It will be set as environment variable with root as base path of move2kube
		// When running in a process shared environment the environment variable will point to the base pid of the container for the image
		envvars := os.Environ()
		for _, envvar := range envvars {
			envvarpair := strings.SplitN(envvar, "=", 2)
			if len(envvarpair) > 0 {
				if envVariableName == container.Image {
					if len(envvarpair) > 1 {
						pid, err := strconv.Atoi(envvarpair[1])
						if err != nil {
							containerEnvironment.Root = envvarpair[1]
						} else {
							containerEnvironment.PID = pid
						}
					}
				}
			}
		}
		if containerEnvironment.PID == 0 && containerEnvironment.Root == "" {
			cengine := GetContainerEngine()
			newImageName := container.Image + env.Name + uniuri.NewLen(5)
			err := cengine.CopyDirsIntoImage(container.Image, newImageName, paths)
			if err != nil {
				logrus.Debugf("Unable to create new container image with new data")
				if container.ContainerBuild.Context != "" {
					err = cengine.BuildImage(container.Image, container.ContainerBuild.Context, container.ContainerBuild.Dockerfile)
					if err != nil {
						logrus.Errorf("Unable to buiold new container image for %s : %s", container.Image, err)
						return env, err
					}
					err = cengine.CopyDirsIntoImage(container.Image, newImageName, paths)
					if err != nil {
						logrus.Errorf("Unable to copy paths to new container image : %s", err)
					}
				} else {
					return env, err
				}
			}
			containerEnvironment.ImageWithData = newImageName
		}
		env.Container = containerEnvironment
	}
	if env.Container.ImageWithData != "" {
		for sp, dp := range paths {
			if dp == "" {
				dp, err = ioutil.TempDir(common.TempPath, "environment-"+name+"-*")
				if err != nil {
					logrus.Errorf("Unable to create ")
				}
			}
			if env.Container.PID != 0 {
				dp = filepath.Join("proc", fmt.Sprint(env.Container.PID), "root", dp)
				env.Paths[sp] = dp
			}
			if sp == dp {
				logrus.Warnf("Source and target paths are same in environment. Ignoring %s.", sp)
				continue
			}
			if err := filesystem.Replicate(sp, dp); err != nil {
				logrus.Errorf("Unable to copy contents to directory %s, dp: %s", sp, dp, err)
				continue
			}
		}
	}
	return env, nil
}

func (e *Environment) Destroy() error {
	if e.Container.ImageWithData != "" {
		cengine := GetContainerEngine()
		err := cengine.RemoveImage(e.Container.ImageWithData)
		if err != nil {
			logrus.Errorf("Unable to delete image %s : %s", e.Container.ImageWithData, err)
		}
	} else {
		for _, dp := range e.Paths {
			err := os.RemoveAll(dp)
			if err != nil {
				logrus.Errorf("Unable to remove directory %s : %s", dp, err)
			}
		}
	}
	return nil
}
