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
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/pathconverters"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

const (
	workspaceDir = "workspace"
)

type Environment struct {
	Name     string
	Env      EnvironmentInstance
	Children []Environment

	Source  string
	Context string

	TempPath string
}

type EnvironmentInstance interface {
	Reset() error
	Download(envpath string) (outpath string, err error)
	Exec(cmd []string) (string, string, int, error)
	Destroy() error

	GetSource() string
	GetContext() string
}

func NewEnvironment(name string, source string, context string, container environmenttypes.Container) (env Environment, err error) {
	tempPath, err := ioutil.TempDir(common.TempPath, "environment-"+name+"-*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return env, err
	}
	env = Environment{
		Name:     name,
		Source:   source,
		Children: []Environment{},
		TempPath: tempPath,
	}
	/*	if container.Image != "" {
		envVariableName := common.MakeStringEnvNameCompliant(container.Image)
		// Check if image is part of the current environment.
		// It will be set as environment variable with root as base path of move2kube
		// When running in a process shared environment the environment variable will point to the base pid of the container for the image
		envvars := os.Environ()
		for _, envvar := range envvars {
			envvarpair := strings.SplitN(envvar, "=", 2)
			if len(envvarpair) > 0 && envVariableName == container.Image && len(envvarpair) > 1 {
				pid, err := strconv.Atoi(envvarpair[1])
				if err != nil {
					env.Env, err = NewLocal(name, source, envvarpair[1])
					if err != nil {
						logrus.Errorf("Unable to create local environment : %s", err)
					}
					return env, err
				} else {
					env.Env, err = NewProcessSharedContainer(name, source, pid)
					if err != nil {
						logrus.Errorf("Unable to create process shared environment : %s", err)
					}
					return env, err
				}
			}
		}
		if env.Env == nil {
			env.Env, err = NewPeerContainer(name, source, container)
			if err != nil {
				logrus.Errorf("Unable to create peer container environment : %s", err)
			}
			return env, err
		}
	}*/
	env.Env, err = NewLocal(name, source, context, tempPath)
	if err != nil {
		logrus.Errorf("Unable to create peer container environment : %s", err)
	}
	return env, err
}

func (e *Environment) AddChild(env Environment) {
	e.Children = append(e.Children, env)
}

func (e *Environment) Reset() error {
	return e.Env.Reset()
}

func (e *Environment) Exec(cmd []string) (string, string, int, error) {
	return e.Env.Exec(cmd)
}

func (e *Environment) Destroy() error {
	e.Env.Destroy()
	for _, env := range e.Children {
		if err := env.Destroy(); err != nil {
			logrus.Errorf("Unable to destroy environment : %s", err)
		}
	}
	return nil
}

func (e *Environment) Encode(obj interface{}) interface{} {
	function := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		if !filepath.IsAbs(path) {
			err := fmt.Errorf("the input path %q is not an absolute path", path)
			logrus.Errorf("%s", err)
			return path, err
		}
		if common.IsParent(path, e.Source) {
			if rel, err := filepath.Rel(path, e.Source); err != nil {
				logrus.Errorf("Unable to make path (%s) relative to source (%s) : %s ", path, e.Source, err)
				return path, err
			} else {
				return filepath.Join(e.Env.GetSource(), rel), nil
			}
		}
		if common.IsParent(path, e.Context) {
			if rel, err := filepath.Rel(path, e.Context); err != nil {
				logrus.Errorf("Unable to make path (%s) relative to source (%s) : %s ", path, e.Source, err)
				return path, err
			} else {
				return filepath.Join(e.Env.GetContext(), rel), nil
			}
		}
		return path, nil
	}
	err := pathconverters.ProcessPaths(obj, function)
	if err != nil {
		logrus.Errorf("Unable to process paths for obj %+v : %s", obj, err)
	}
	return obj
}

func (e *Environment) DownloadAndDecode(obj interface{}) interface{} {
	function := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		if !filepath.IsAbs(path) {
			err := fmt.Errorf("the input path %q is not an absolute path", path)
			logrus.Errorf("%s", err)
			return path, err
		}
		outpath, err := e.Env.Download(path)
		if err != nil {
			logrus.Errorf("Unable to copy data from path %s : %s", path, err)
			return path, err
		}
		return outpath, nil
	}
	err := pathconverters.ProcessPaths(obj, function)
	if err != nil {
		logrus.Errorf("Unable to process paths for obj %+v : %s", obj, err)
	}
	return obj
}
