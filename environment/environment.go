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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/internal/common"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

type Environment struct {
	Name      string
	Context   string
	Source    string
	Paths     map[string]string
	Container ContainerEnvironment
	TempDir   string
	Children  []Environment
}

type ContainerEnvironment struct {
	ImageName     string
	ImageWithData string
	CID           string // A started instance of ImageWithData
	PID           int
	Root          string
}

func NewEnvironment(name string, source string, context string, container environmenttypes.Container) (env Environment, err error) {
	env = Environment{
		Name:     name,
		Source:   source,
		Context:  context,
		Paths:    map[string]string{},
		Children: []Environment{},
	}
	env.TempDir, err = ioutil.TempDir(common.TempPath, "environment-"+name+"-*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
	}
	tempSrc := filepath.Join(env.TempDir, "src")
	err = os.MkdirAll(tempSrc, common.DefaultDirectoryPermission)
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
	}
	env.Paths[source] = tempSrc
	if (container != environmenttypes.Container{}) {
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
			if cengine == nil {
				return env, fmt.Errorf("no working container runtime found")
			}
			newImageName := container.Image + strings.ToLower(env.Name+uniuri.NewLen(5))
			err := cengine.CopyDirsIntoImage(container.Image, newImageName, env.Paths)
			if err != nil {
				logrus.Debugf("Unable to create new container image with new data")
				if container.ContainerBuild.Context != "" {
					err = cengine.BuildImage(container.Image, container.ContainerBuild.Context, container.ContainerBuild.Dockerfile)
					if err != nil {
						logrus.Errorf("Unable to buiold new container image for %s : %s", container.Image, err)
						return env, err
					}
					err = cengine.CopyDirsIntoImage(container.Image, newImageName, env.Paths)
					if err != nil {
						logrus.Errorf("Unable to copy paths to new container image : %s", err)
					}
				} else {
					return env, err
				}
			}
			containerEnvironment.ImageWithData = newImageName
			cid, err := cengine.CreateContainer(newImageName)
			if err != nil {
				logrus.Errorf("Unable to start container with image %s : %s", newImageName, cid)
				return env, err
			}
			containerEnvironment.CID = cid
		}
		env.Container = containerEnvironment
	}
	if env.Container.CID == "" {
		for sp, dp := range env.Paths {
			if dp == "" {
				dp, err = ioutil.TempDir(common.TempPath, "environment-"+name+"-*")
				if err != nil {
					logrus.Errorf("Unable to create temp dir : %s", err)
				}
			}
			if env.Container.PID != 0 {
				logrus.Infof("Inside %+v", env.Container)
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
	env.Source = env.EncodePath(env.Source)
	return env, nil
}

func (e *Environment) AddChild(env Environment) {
	e.Children = append(e.Children, env)
}

func (e *Environment) Sync() error {
	if e.Container.ImageWithData != "" {
		cengine := GetContainerEngine()
		err := cengine.StopAndRemoveContainer(e.Container.CID)
		if err != nil {
			logrus.Errorf("Unable to delete image %s : %s", e.Container.ImageWithData, err)
		}
		cid, err := cengine.CreateContainer(e.Container.ImageWithData)
		if err != nil {
			logrus.Errorf("Unable to start container with image %s : %s", e.Container.ImageWithData, cid)
			return err
		}
		e.Container.CID = cid
	} else {
		for sp, dp := range e.Paths {
			err := filesystem.Replicate(sp, dp)
			if err != nil {
				logrus.Errorf("Unable to remove directory %s : %s", dp, err)
			}
		}
	}
	return nil
}

func (e *Environment) SyncOutput(path string) (outputPath string) {
	output, err := ioutil.TempDir(e.TempDir, "*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return path
	}
	if e.Container.CID != "" {
		cengine := GetContainerEngine()
		err = cengine.CopyDirsFromContainer(e.Container.CID, map[string]string{path: output})
		if err != nil {
			logrus.Errorf("Unable to copy paths to new container image : %s", err)
		}
		return output
	} else if e.Container.PID != 0 {
		nsp := filepath.Join("proc", fmt.Sprint(e.Container.PID), "root", path)
		err = filesystem.Replicate(nsp, output)
		if err != nil {
			logrus.Errorf("Unable to replicate in syncoutput : %s", err)
			return path
		}
		return output
	}
	return path
}

func (e *Environment) Destroy() error {
	if e.Container.ImageWithData != "" {
		cengine := GetContainerEngine()
		err := cengine.RemoveImage(e.Container.ImageWithData)
		if err != nil {
			logrus.Errorf("Unable to delete image %s : %s", e.Container.ImageWithData, err)
		}
		err = cengine.StopAndRemoveContainer(e.Container.CID)
		if err != nil {
			logrus.Errorf("Unable to stop and remove container %s : %s", e.Container.CID, err)
		}
	} else {
		for _, dp := range e.Paths {
			err := os.RemoveAll(dp)
			if err != nil {
				logrus.Errorf("Unable to remove directory %s : %s", dp, err)
			}
		}
	}
	for _, env := range e.Children {
		if err := env.Destroy(); err != nil {
			logrus.Errorf("Unable to destroy environment : %s", err)
		}
	}
	return nil
}

func (e *Environment) EncodePath(outsidepath string) string {
	for sp, dp := range e.Paths {
		if common.IsParent(outsidepath, sp) {
			relPath, err := filepath.Rel(outsidepath, sp)
			if err != nil {
				logrus.Errorf("Unable to map rel paths : %s", err)
			} else {
				return filepath.Join(dp, relPath)
			}
		}
	}
	return outsidepath
}

func (e *Environment) DecodePath(envpath string) string {
	for sp, dp := range e.Paths {
		if common.IsParent(envpath, dp) {
			relPath, err := filepath.Rel(envpath, dp)
			if err != nil {
				logrus.Errorf("Unable to map rel paths : %s", err)
			} else {
				return filepath.Join(sp, relPath)
			}
		}
	}
	return envpath
}

func (e *Environment) GetSourcePath() string {
	return e.Source
}

func (e *Environment) Exec(cmd environmenttypes.Command, workingDir string) (string, string, int, error) {
	if workingDir == "" {
		workingDir = e.Context
	}
	if (e.Container != ContainerEnvironment{}) {
		if e.Container.CID != "" {
			cengine := GetContainerEngine()
			return cengine.RunCmdInContainer(e.Container.CID, cmd, "")
		}
		if e.Container.PID != 0 {
			//TODO : Fix me
			workingDir = filepath.Join("proc", fmt.Sprint(e.Container.PID), "root", workingDir)
		} else if workingDir == "" && e.Container.Root != "" {
			workingDir = e.Container.Root
		}
	}
	var exitcode int
	var outb, errb bytes.Buffer
	execcmd := exec.Command(cmd.CMD, cmd.Args...)
	execcmd.Dir = e.Context
	execcmd.Dir = workingDir
	execcmd.Stdout = &outb
	execcmd.Stderr = &errb
	err := execcmd.Run()
	if err != nil {
		var ee *exec.ExitError
		var pe *os.PathError
		if errors.As(err, &ee) {
			exitcode = ee.ExitCode()
			err = nil
		} else if errors.As(err, &pe) {
			logrus.Errorf("PathError during execution of command: %v", pe)
			err = pe
		} else {
			logrus.Errorf("Generic error during execution of command: %v", err)
		}
	}
	return outb.String(), errb.String(), exitcode, err
}
