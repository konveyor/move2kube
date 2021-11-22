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
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"text/template"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/common/deepcopy"
	"github.com/konveyor/move2kube/common/pathconverters"
	"github.com/konveyor/move2kube/environment/container"
	"github.com/konveyor/move2kube/types"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

const (
	workspaceDir    = "workspace"
	templatePattern = "{{"
)

var (
	// GRPCEnvName represents the environment variable name used to pass the GRPC server information to the transformers
	GRPCEnvName = strings.ToUpper(types.AppNameShort) + "_QA_GRPC_SERVER"
	// ProjectNameEnvName stores the project name
	ProjectNameEnvName = strings.ToUpper(types.AppNameShort) + "_PROJECT_NAME"
	// SourceEnvName stores the source path
	SourceEnvName = strings.ToUpper(types.AppNameShort) + "_SOURCE"
	// OutputEnvName stores the output path
	OutputEnvName = strings.ToUpper(types.AppNameShort) + "_OUTPUT"
	// ContextEnvName stores the context
	ContextEnvName = strings.ToUpper(types.AppNameShort) + "_CONTEXT"
	// CurrOutputEnvName stores the location of output from the previous iteration
	CurrOutputEnvName = strings.ToUpper(types.AppNameShort) + "_CURRENT_OUTPUT"
	// RelTemplatesDirEnvName stores the rel templates directory
	RelTemplatesDirEnvName = strings.ToUpper(types.AppNameShort) + "_RELATIVE_TEMPLATES_DIR"
	// TempPathEnvName stores the temp path
	TempPathEnvName = strings.ToUpper(types.AppNameShort) + "_TEMP"
	// EnvNameEnvName stores the environment name
	EnvNameEnvName = strings.ToUpper(types.AppNameShort) + "_ENV_NAME"
)

// Environment is used to manage EnvironmentInstances
type Environment struct {
	EnvInfo
	Env          EnvironmentInstance
	Children     []*Environment
	TempPathsMap map[string]string
	active       bool
}

// EnvironmentInstance represents a actual instance of an environment which the Environment manages
type EnvironmentInstance interface {
	Reset() error
	Download(envpath string) (outpath string, err error)
	Upload(outpath string) (envpath string, err error)
	Exec(cmd []string) (stdout string, stderr string, exitcode int, err error)
	Destroy() error

	GetSource() string
	GetContext() string
}

// NewEnvironment creates a new environment
func NewEnvironment(envInfo EnvInfo, grpcQAReceiver net.Addr, c environmenttypes.Container) (env *Environment, err error) {
	tempPath, err := ioutil.TempDir(common.TempPath, "environment-"+envInfo.Name+"-*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return env, err
	}
	envInfo.TempPath = tempPath
	env = &Environment{
		EnvInfo:      envInfo,
		Children:     []*Environment{},
		TempPathsMap: map[string]string{},
		active:       true,
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
					envInfo.Context = envvarpair[1]
					env.Env, err = NewLocal(envInfo, grpcQAReceiver)
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
			env.Env, err = NewPeerContainer(envInfo, grpcQAReceiver, c)
			if err != nil && !container.IsDisabled() {
				logrus.Errorf("Unable to create peer container environment : %s", err)
			}
			return env, err
		}
	}
	env.Env, err = NewLocal(envInfo, grpcQAReceiver)
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
	if !e.active {
		logrus.Debug("environment not active. Process is terminating")
		return nil
	}
	e.CurrEnvOutputBasePath = ""
	return e.Env.Reset()
}

// Exec executes an executable within the environment
func (e *Environment) Exec(cmd []string) (stdout string, stderr string, exitcode int, err error) {
	if !e.active {
		err = &EnvironmentNotActiveError{}
		logrus.Debug(err)
		return "", "", 0, err
	}
	return e.Env.Exec(cmd)
}

// Destroy destroys all artifacts specific to the environment
func (e *Environment) Destroy() error {
	e.active = false
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
	if !e.active {
		logrus.Debug("environment not active. Process is terminating")
		return dupobj
	}
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
	if !e.active {
		logrus.Debug("environment not active. Process is terminating")
		return dupobj
	}
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
	if !e.active {
		logrus.Debug("environment not active. Process is terminating")
		return dupobj
	}
	function := func(path string) (string, error) {
		if path == "" {
			return path, nil
		}
		if tempPath, err := common.GetStringFromTemplate(path, e.TempPathsMap); err == nil && strings.Contains(path, templatePattern) {
			path = tempPath
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
		if _, err := os.Stat(path); err != nil {
			logrus.Debugf("Path [%s] does not exist", path)
			return path, nil
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
	if !e.active {
		logrus.Debug("environment not active. Process is terminating")
		return dupPathMappings
	}

	tempMappings := []transformertypes.PathMapping{}
	for pmi, pm := range dupPathMappings {
		if strings.EqualFold(pm.Type, transformertypes.PathTemplatePathMappingType) {
			// Process path template
			methodMap := template.FuncMap{
				"SourceRel":    e.SourceRel,
				"OutputRel":    e.OutputRel,
				"TempRoot":     e.CreateTempRoot,
				"FilePathBase": filepath.Base,
			}
			tpl, err := template.Must(template.New("pathTpl"), nil).Funcs(methodMap).Parse(pm.SrcPath)
			if err != nil {
				logrus.Errorf("Error while parsing path template : %s", err)
				continue
			}
			var path bytes.Buffer
			err = tpl.Execute(&path, pm.TemplateConfig)
			if err != nil {
				logrus.Errorf("Error while processing path template : %s", err)
				continue
			}
			pathStr := path.String()
			logrus.Debugf("Output of environment template: %s\n", pathStr)
			if filepath.IsAbs(pathStr) {
				tempOutputPath, err := ioutil.TempDir(e.TempPath, "*")
				if err != nil {
					logrus.Errorf("Unable to create temp dir : %s", err)
					continue
				}
				pathStr = filepath.Join(tempOutputPath, filepath.Base(pathStr))
			}
			pathTplName, err := common.GetStringFromTemplate("{{ .PathTemplateName }}", pm.TemplateConfig)
			if err != nil {
				logrus.Errorf("Unable to create temp dir : %s", err)
				continue
			}
			e.TempPathsMap[pathTplName] = pathStr
		} else {
			if filepath.IsAbs(pm.SrcPath) && common.IsParent(pm.SrcPath, e.GetEnvironmentOutput()) {
				var err error
				dupPathMappings[pmi].SrcPath, err = e.Env.Download(pm.SrcPath)
				if err != nil {
					logrus.Errorf("Error while processing path mappings : %s", err)
				}
			}
			if (strings.EqualFold(pm.Type, transformertypes.TemplatePathMappingType) ||
				strings.EqualFold(pm.Type, transformertypes.SpecialTemplatePathMappingType)) &&
				(pm.SrcPath == "" || !filepath.IsAbs(pm.SrcPath)) {
				dupPathMappings[pmi].SrcPath = filepath.Join(e.GetEnvironmentContext(), e.RelTemplatesDir, pm.SrcPath)
			}
			tempMappings = append(tempMappings, dupPathMappings[pmi])
		}
	}
	dupPathMappings = tempMappings
	return dupPathMappings
}

// SourceRel makes the path base-dir relative. Exposed to be used within path-mapping destination-path template.
func (e *Environment) SourceRel(destPath string) (string, error) {
	if !common.IsParent(destPath, e.GetEnvironmentSource()) {
		return "", fmt.Errorf("%s not parent of %s", destPath, e.GetEnvironmentSource())
	}
	dp, err := filepath.Rel(e.GetEnvironmentSource(), destPath)
	if err != nil {
		logrus.Errorf("Unable to convert destination path relative to env source : %s", err)
		return "", err
	}
	return dp, nil
}

// OutputRel makes the path output-dir relative. Exposed to be used within path-mapping destination-path template.
func (e *Environment) OutputRel(destPath string) (string, error) {
	if !common.IsParent(destPath, e.CurrEnvOutputBasePath) {
		return "", fmt.Errorf("%s not parent of %s", destPath, e.GetEnvironmentSource())
	}
	dp, err := filepath.Rel(e.CurrEnvOutputBasePath, destPath)
	if err != nil {
		logrus.Errorf("Unable to convert destination path relative to env source : %s", err)
		return "", err
	}
	return dp, nil
}

// CreateTempRoot returns the "/" to indicate the temp-root creation. Exposed to be used within path-mapping destination-path template.
func (e *Environment) CreateTempRoot() string {
	return "/"
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
