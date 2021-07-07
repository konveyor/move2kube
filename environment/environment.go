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

package environment

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/environment/container"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/deepcopy"
	"github.com/konveyor/move2kube/internal/common/pathconverters"
	"github.com/konveyor/move2kube/types"
	"github.com/konveyor/move2kube/types/collection"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

const workspaceDir = "workspace"

// GRPCEnvName represents the environment variable name used to pass the GRPC server information to the transformers
var GRPCEnvName = strings.ToUpper(types.AppNameShort) + "QA_GRPC_SERVER"

// Environment is used to manage EnvironmentInstances
type Environment struct {
	Name     string
	Env      EnvironmentInstance
	Children []*Environment

	ProjectName   string
	TargetCluster collection.ClusterMetadata

	Source                string
	Output                string
	Context               string
	CurrEnvOutputBasePath string
	RelTemplatesDir       string
	TempPath              string
}

// EnvironmentInstance represents a actual instance of an environment which the Environment manages
type EnvironmentInstance interface {
	Reset() error
	Download(envpath string) (outpath string, err error)
	Upload(outpath string) (envpath string, err error)
	Exec(cmd []string) (string, string, int, error)
	Destroy() error

	GetSource() string
	GetContext() string
}

// NewEnvironment creates a new environment
func NewEnvironment(name string, projectName string, targetCluster collection.ClusterMetadata, source string, output string, context string, relTemplatesDir string, grpcQAReceiver net.Addr, c environmenttypes.Container) (env *Environment, err error) {
	tempPath, err := ioutil.TempDir(common.TempPath, "environment-"+name+"-*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return env, err
	}
	env = &Environment{
		Name:            name,
		ProjectName:     projectName,
		TargetCluster:   targetCluster,
		Source:          source,
		Output:          output,
		Context:         context,
		RelTemplatesDir: relTemplatesDir,
		Children:        []*Environment{},
		TempPath:        tempPath,
	}
	if c.Image != "" {
		envVariableName := common.MakeStringEnvNameCompliant(c.Image)
		// Check if image is part of the current environment.
		// It will be set as environment variable with root as base path of move2kube
		// When running in a process shared environment the environment variable will point to the base pid of the container for the image
		envvars := os.Environ()
		for _, envvar := range envvars {
			envvarpair := strings.SplitN(envvar, "=", 2)
			if len(envvarpair) > 0 && envVariableName == c.Image && len(envvarpair) > 1 {
				_, err := strconv.Atoi(envvarpair[1])
				if err != nil {
					env.Env, err = NewLocal(name, source, envvarpair[1], tempPath, grpcQAReceiver)
					if err != nil {
						logrus.Errorf("Unable to create local environment : %s", err)
					}
					return env, err
				} /*else {
					env.Env, err = NewProcessSharedContainer(name, source, pid)
					if err != nil {
						logrus.Errorf("Unable to create process shared environment : %s", err)
					}
					return env, err
				}*/
			}
		}
		if env.Env == nil {
			env.Env, err = NewPeerContainer(name, source, context, tempPath, grpcQAReceiver, c)
			if err != nil && !container.IsDisabled() {
				logrus.Errorf("Unable to create peer container environment : %s", err)
			}
			return env, err
		}
	}
	env.Env, err = NewLocal(name, source, context, tempPath, grpcQAReceiver)
	if err != nil {
		logrus.Errorf("Unable to create Local environment : %s", err)
	}
	return env, err
}

// AddChild adds a child to the environment
func (e *Environment) AddChild(env *Environment) {
	e.Children = append(e.Children, env)
}

// Reset resets an environment
func (e *Environment) Reset() error {
	e.CurrEnvOutputBasePath = ""
	return e.Env.Reset()
}

// Exec executes an executable within the environment
func (e *Environment) Exec(cmd []string) (string, string, int, error) {
	return e.Env.Exec(cmd)
}

// Destroy destroys all artifacts specific to the environment
func (e *Environment) Destroy() error {
	e.Env.Destroy()
	for _, env := range e.Children {
		if err := env.Destroy(); err != nil {
			logrus.Errorf("Unable to destroy environment : %s", err)
		}
	}
	return nil
}

// Encode encodes all paths in the obj to be relevant to the environment
func (e *Environment) Encode(obj interface{}) interface{} {
	dupobj := deepcopy.DeepCopy(obj)
	function := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		if !filepath.IsAbs(path) {
			var err error
			if e.CurrEnvOutputBasePath == "" {
				e.CurrEnvOutputBasePath, err = e.Env.Upload(e.Output)
			}
			return filepath.Join(e.CurrEnvOutputBasePath, path), err
		}
		if common.IsParent(path, e.Source) {
			rel, err := filepath.Rel(e.Source, path)
			if err != nil {
				logrus.Errorf("Unable to make path (%s) relative to source (%s) : %s ", path, e.Source, err)
				return path, err
			}
			return filepath.Join(e.Env.GetSource(), rel), nil
		}
		if common.IsParent(path, e.Context) {
			rel, err := filepath.Rel(e.Context, path)
			if err != nil {
				logrus.Errorf("Unable to make path (%s) relative to source (%s) : %s ", path, e.Source, err)
				return path, err
			}
			return filepath.Join(e.Env.GetContext(), rel), nil
		}
		return e.Env.Upload(path)
	}
	if reflect.ValueOf(obj).Kind() == reflect.String {
		val, err := function(obj.(string))
		if err != nil {
			logrus.Errorf("Unable to process paths for obj %+v : %s", obj, err)
		}
		return val
	}
	err := pathconverters.ProcessPaths(dupobj, function)
	if err != nil {
		logrus.Errorf("Unable to process paths for obj %+v : %s", dupobj, err)
	}
	return dupobj
}

// Decode decodes all paths in the passed obj
func (e *Environment) Decode(obj interface{}) interface{} {
	dupobj := deepcopy.DeepCopy(obj)
	function := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		if !filepath.IsAbs(path) {
			err := fmt.Errorf("the input path %q is not an absolute path", path)
			logrus.Errorf("%s", err)
			return path, err
		}
		if common.IsParent(path, e.GetEnvironmentSource()) {
			rel, err := filepath.Rel(e.GetEnvironmentSource(), path)
			if err != nil {
				logrus.Errorf("Unable to make path (%s) relative to source (%s) : %s ", path, e.GetEnvironmentSource(), err)
				return path, err
			}
			return filepath.Join(e.Source, rel), nil
		}
		if common.IsParent(path, e.GetEnvironmentContext()) {
			rel, err := filepath.Rel(e.GetEnvironmentContext(), path)
			if err != nil {
				logrus.Errorf("Unable to make path (%s) relative to source (%s) : %s ", path, e.GetEnvironmentContext(), err)
				return path, err
			}
			return filepath.Join(e.Context, rel), nil
		}
		if common.IsParent(path, e.GetEnvironmentOutput()) {
			rel, err := filepath.Rel(e.GetEnvironmentOutput(), path)
			if err != nil {
				logrus.Errorf("Unable to make path (%s) relative to source (%s) : %s ", path, e.GetEnvironmentOutput(), err)
				return path, err
			}
			return rel, nil
		}
		return path, nil
	}
	if reflect.ValueOf(dupobj).Kind() == reflect.String {
		val, err := function(dupobj.(string))
		if err != nil {
			logrus.Errorf("Unable to process paths for obj %+v : %s", obj, err)
		}
		return val
	}
	err := pathconverters.ProcessPaths(dupobj, function)
	if err != nil {
		logrus.Errorf("Unable to process paths for obj %+v : %s", dupobj, err)
	}
	return dupobj
}

// DownloadAndDecode downloads and decodes the data from the paths in the object
func (e *Environment) DownloadAndDecode(obj interface{}, downloadSource bool) interface{} {
	dupobj := deepcopy.DeepCopy(obj)
	function := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		if !filepath.IsAbs(path) {
			logrus.Debugf("the input path %q is not an absolute path", path)
			return path, nil
		}
		if !downloadSource {
			if common.IsParent(path, e.GetEnvironmentSource()) {
				relPath, err := filepath.Rel(e.GetEnvironmentSource(), path)
				if err != nil {
					logrus.Errorf("Unable to convert source to rel path : %s", err)
					return path, err
				}
				return filepath.Join(e.Source, relPath), nil
			}
		}
		if common.IsParent(path, e.GetEnvironmentOutput()) {
			relPath, err := filepath.Rel(e.GetEnvironmentOutput(), path)
			if err != nil {
				logrus.Errorf("Unable to convert source to rel path : %s", err)
				return path, err
			}
			return relPath, nil
		}
		outpath, err := e.Env.Download(path)
		if err != nil {
			logrus.Errorf("Unable to copy data from path %s : %s", path, err)
			return path, err
		}
		return outpath, nil
	}
	err := pathconverters.ProcessPaths(dupobj, function)
	if err != nil {
		logrus.Errorf("Unable to process paths for obj %+v : %s", dupobj, err)
	}
	return dupobj
}

// ProcessPathMappings post processes the paths in the path mappings
func (e *Environment) ProcessPathMappings(pathMappings []transformertypes.PathMapping) []transformertypes.PathMapping {
	dupPathMappings := deepcopy.DeepCopy(pathMappings).([]transformertypes.PathMapping)
	for pmi, pm := range dupPathMappings {
		if strings.EqualFold(pm.Type, transformertypes.TemplatePathMappingType) && (pm.SrcPath == "" || !filepath.IsAbs(pm.SrcPath)) {
			dupPathMappings[pmi].SrcPath = filepath.Join(e.GetEnvironmentContext(), e.RelTemplatesDir, pm.SrcPath)
		}

		// Process destination Path
		destPathSplit := strings.SplitN(pm.DestPath, ":", 2)
		relPath := ""
		destPath := pm.DestPath
		if len(destPathSplit) > 1 {
			relPath = destPathSplit[0]
			destPath = destPathSplit[1]
		}
		if filepath.IsAbs(destPath) {
			dp, err := filepath.Rel(e.GetEnvironmentSource(), destPath)
			if err != nil {
				logrus.Errorf("Unable to convert destination path relative to env source : %s", err)
				continue
			}
			dupPathMappings[pmi].DestPath = filepath.Join(relPath, dp)
		}
	}
	return dupPathMappings
}

// GetEnvironmentSource returns the source path within the environment
func (e *Environment) GetEnvironmentSource() string {
	return e.Env.GetSource()
}

// GetEnvironmentContext returns the context path within the environment
func (e *Environment) GetEnvironmentContext() string {
	return e.Env.GetContext()
}

// GetEnvironmentOutput returns the output path within the environment
func (e *Environment) GetEnvironmentOutput() string {
	return e.CurrEnvOutputBasePath
}

// GetProjectName returns the project name
func (e *Environment) GetProjectName() string {
	return e.ProjectName
}
