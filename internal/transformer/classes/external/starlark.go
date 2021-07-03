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

package external

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/qaengine"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/qri-io/starlib"
	starutil "github.com/qri-io/starlib/util"
	"github.com/sirupsen/logrus"
	"go.starlark.net/starlark"
)

const (
	baseDirectoryDetectFnName = "base_directory_detect"
	directoryDetectFnName     = "directory_detect"
	transformFnName           = "transform"

	qaFunctionName           = "query"
	sourceDirVarName         = "source"
	contextDirVarName        = "context"
	transformerConfigVarName = "config"
	projectVarName           = "project"
)

// Starlark implements transformer interface and is used to write simple external transformers
type Starlark struct {
	TConfig     transformertypes.Transformer
	StarConfig  StarYamlConfig
	StarThread  *starlark.Thread
	StarGlobals starlark.StringDict
	Env         environment.Environment

	baseDetectFn *starlark.Function
	detectFn     *starlark.Function
	transformFn  *starlark.Function
}

// StarYamlConfig defines yaml config for Starlark transformers
type StarYamlConfig struct {
	StarFile string `yaml:"starFile"`
}

// Init Initializes the transformer
func (t *Starlark) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	t.StarConfig = StarYamlConfig{}
	err = common.GetObjFromInterface(t.TConfig.Spec.Config, &t.StarConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.TConfig.Spec.Config, t.StarConfig, err)
		return err
	}
	t.StarThread = &starlark.Thread{Name: tc.Name}
	t.setDefaultGlobals()
	t.StarGlobals[qaFunctionName] = starlark.NewBuiltin(qaFunctionName, t.query)
	tcmapobj, err := common.GetMapInterfaceFromObj(tc)
	if err != nil {
		logrus.Errorf("Unable to conver transformer config to map[string]interface{}")
		return err
	}
	t.StarGlobals[env.ProjectName], err = starutil.Marshal(env.ProjectName)
	if err != nil {
		logrus.Errorf("Unable to load transformer config : %s", err)
		return err
	}
	t.StarGlobals[transformerConfigVarName], err = starutil.Marshal(tcmapobj)
	if err != nil {
		logrus.Errorf("Unable to load transformer config : %s", err)
		return err
	}
	t.StarGlobals[contextDirVarName], err = starutil.Marshal(env.GetEnvironmentContext())
	if err != nil {
		logrus.Errorf("Unable to load context : %s", err)
		return err
	}
	t.StarGlobals[sourceDirVarName], err = starutil.Marshal(env.GetEnvironmentSource())
	if err != nil {
		logrus.Errorf("Unable to load source : %s", err)
		return err
	}
	t.StarGlobals, err = starlark.ExecFile(t.StarThread, filepath.Join(t.Env.GetEnvironmentContext(), t.StarConfig.StarFile), nil, t.StarGlobals)
	if err != nil {
		logrus.Errorf("Unable to load starlark file %s : %s", filepath.Join(t.Env.GetEnvironmentContext(), t.StarConfig.StarFile), err)
		return err
	}
	err = t.loadFunctions()
	if err != nil {
		logrus.Errorf("Unable to load required functions : %s", err)
	}
	return err
}

// GetConfig returns the transformer config
func (t *Starlark) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

// DetectOutput structure is the data format for receiving data from starlark detect functions
type DetectOutput struct {
	NamedServices   map[string]plantypes.Service `yaml:"namedServices,omitempty" json:"namedServices,omitempty"`
	UnNamedServices []plantypes.Transformer      `yaml:"unnamedServices,omitempty" json:"unnamedServices,omitempty"`
}

// BaseDirectoryDetect runs detect in base directory
func (t *Starlark) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return t.executeDetect(t.baseDetectFn, dir)
}

// DirectoryDetect runs detect in each sub directory
func (t *Starlark) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return t.executeDetect(t.detectFn, dir)
}

// TransformOutput structure is the data format for receiving data from starlark transform functions
type TransformOutput struct {
	PathMappings     []transformertypes.PathMapping `yaml:"pathMappings,omitempty" json:"pathMappings,omitempty"`
	CreatedArtifacts []transformertypes.Artifact    `yaml:"createdArtifacts,omitempty" json:"createdArtifacts,omitempty"`
}

// Transform transforms the artifacts
func (t *Starlark) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	naObj, err := common.GetMapInterfaceFromObj(newArtifacts)
	if err != nil {
		logrus.Errorf("Unable to conver new artifacts to map[string]interface{}")
		return nil, nil, err
	}
	starNewArtifacts, err := starutil.Marshal(naObj)
	if err != nil {
		logrus.Errorf("Unable to convert %s to starlark value : %s", newArtifacts, err)
		return nil, nil, err
	}
	oaObj, err := common.GetMapInterfaceFromObj(oldArtifacts)
	if err != nil {
		logrus.Errorf("Unable to conver new artifacts to map[string]interface{}")
		return nil, nil, err
	}
	starOldArtifacts, err := starutil.Marshal(oaObj)
	if err != nil {
		logrus.Errorf("Unable to convert %s to starlark value : %s", oldArtifacts, err)
		return nil, nil, err
	}
	val, err := starlark.Call(t.StarThread, t.transformFn, starlark.Tuple{starNewArtifacts, starOldArtifacts}, nil)
	if err != nil {
		logrus.Errorf("Unable to execute starlark function : %s", err)
		return nil, nil, err
	}
	valI, err := starutil.Unmarshal(val)
	if err != nil {
		logrus.Errorf("Unable to unmarshal starlark function result : %s", err)
		return nil, nil, err
	}
	transformOutput := TransformOutput{}
	err = common.GetObjFromInterface(valI, &transformOutput)
	if err != nil {
		logrus.Errorf("unable to load result for Transformer %+v into %T : %s", valI, transformOutput, err)
		return nil, nil, err
	}
	return transformOutput.PathMappings, transformOutput.CreatedArtifacts, nil
}

func (t *Starlark) executeDetect(fn *starlark.Function, dir string) (nameServices map[string]plantypes.Service, unservices []plantypes.Transformer, err error) {
	if fn == nil {
		return nil, nil, nil
	}
	starDir, err := starutil.Marshal(dir)
	if err != nil {
		logrus.Errorf("Unable to convert %s to starlark value : %s", dir, err)
		return nil, nil, err
	}
	val, err := starlark.Call(t.StarThread, fn, starlark.Tuple{starDir}, nil)
	if err != nil {
		logrus.Errorf("Unable to execute starlark function : %s", err)
		return nil, nil, err
	}
	valI, err := starutil.Unmarshal(val)
	if err != nil {
		logrus.Errorf("Unable to unmarshal starlark function result : %s", err)
		return nil, nil, err
	}
	detectOutput := DetectOutput{}
	err = common.GetObjFromInterface(valI, &detectOutput)
	if err != nil {
		logrus.Errorf("unable to load result for Transformer %+v into %T : %s", valI, detectOutput, err)
		return nil, nil, err
	}
	return detectOutput.NamedServices, detectOutput.UnNamedServices, nil
}

func (t *Starlark) query(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	argDictValue := &starlark.Dict{}
	if err := starlark.UnpackPositionalArgs(qaFunctionName, args, kwargs, 1, &argDictValue); err != nil {
		return starlark.None, fmt.Errorf("invalid args provided to '%s'. Expected a single dict argument. Error: %q", qaFunctionName, err)
	}
	argI, err := starutil.Unmarshal(argDictValue)
	if err != nil {
		return starlark.None, fmt.Errorf("failed to unmarshal the argument provided to '%s'. Expected a single dict argument. Error: %q", qaFunctionName, err)
	}
	prob := qatypes.Problem{}
	err = common.GetObjFromInterface(argI, &prob)
	if err != nil {
		logrus.Errorf("Unable to convert interface %+v to problem %T : %s", argI, prob, err)
		return starlark.None, err
	}
	// key
	if prob.ID == "" {
		return starlark.None, fmt.Errorf("the key 'id' is missing from the question object %+v", argI)
	}
	if !strings.HasPrefix(prob.ID, common.BaseKey) {
		prob.ID = common.BaseKey + common.Delim + prob.ID
	}
	// type
	if prob.Type == "" {
		prob.Type = qatypes.InputSolutionFormType
	}
	resolved, err := qaengine.FetchAnswer(prob)
	if err != nil {
		logrus.Fatalf("failed to ask the question. Error: %q", err)
	}
	answerValue, err := starutil.Marshal(resolved.Answer)
	if err != nil {
		return starlark.None, fmt.Errorf("failed to marshal the answer %+v of type %T into a starlark value. Error: %q", resolved.Answer, resolved.Answer, err)
	}
	return answerValue, err
}

func (t *Starlark) setDefaultGlobals() {
	t.StarGlobals = starlark.StringDict{}
	t.addModules("encoding/json")
	t.addModules("math")
	t.addModules("time")
	t.addModules("xlsx")
	t.addModules("html")
	t.addModules("bsoup")
	t.addModules("zipfile")
	t.addModules("re")
	t.addModules("encoding/base64")
	t.addModules("encoding/csv")
	t.addModules("encoding/yaml")
	t.addModules("geo")
	t.addModules("hash")
}

func (t *Starlark) addModules(modName string) {
	mod, err := starlib.Loader(t.StarThread, modName+".star")
	if err != nil {
		logrus.Errorf("Unable to load starlarkmodule : %s", err)
		return
	}
	for key, module := range mod {
		t.StarGlobals[key] = module
	}
}

func (t *Starlark) loadFunctions() (err error) {
	err = t.loadBaseDetectFn()
	if err != nil {
		logrus.Errorf("Unable to load base detect function : %s", err)
		return err
	}
	err = t.loadDetectFn()
	if err != nil {
		logrus.Errorf("Unable to load detect function : %s", err)
		return err
	}
	err = t.loadTransformFn()
	if err != nil {
		logrus.Errorf("Unable to load transform function : %s", err)
		return err
	}
	return nil
}

func (t *Starlark) loadBaseDetectFn() (err error) {
	if !t.StarGlobals.Has(baseDirectoryDetectFnName) {
		return nil
	}
	baseDirectoryDetectFn := t.StarGlobals[baseDirectoryDetectFnName]
	fn, ok := baseDirectoryDetectFn.(*starlark.Function)
	if !ok {
		err = fmt.Errorf("%s is not a function", baseDirectoryDetectFn)
		logrus.Errorf("%s", err)
		return err
	}
	if fn.NumParams() != 1 {
		err = fmt.Errorf("%s does not have the required number of paramters. It has %d, expected %d", baseDirectoryDetectFn, fn.NumParams(), 1)
		logrus.Errorf("%s", err)
		return err
	}
	t.baseDetectFn = fn
	return nil
}

func (t *Starlark) loadDetectFn() (err error) {
	if !t.StarGlobals.Has(directoryDetectFnName) {
		return nil
	}
	directoryDetectFn := t.StarGlobals[directoryDetectFnName]
	fn, ok := directoryDetectFn.(*starlark.Function)
	if !ok {
		err = fmt.Errorf("%s is not a function", directoryDetectFn)
		logrus.Errorf("%s", err)
		return err
	}
	if fn.NumParams() != 1 {
		err = fmt.Errorf("%s does not have the required number of paramters. It has %d, expected %d", directoryDetectFn, fn.NumParams(), 1)
		logrus.Errorf("%s", err)
		return err
	}
	t.detectFn = fn
	return nil
}

func (t *Starlark) loadTransformFn() (err error) {
	if !t.StarGlobals.Has(transformFnName) {
		err = fmt.Errorf("no %s function found", transformFnName)
		logrus.Errorf("%s", err)
		return err
	}
	transformFn := t.StarGlobals[transformFnName]
	fn, ok := transformFn.(*starlark.Function)
	if !ok {
		err = fmt.Errorf("%s is not a function", transformFn)
		logrus.Errorf("%s", err)
		return err
	}
	if fn.NumParams() != 2 {
		err = fmt.Errorf("%s does not have the required number of paramters. It has %d, expected %d", transformFn, fn.NumParams(), 2)
		logrus.Errorf("%s", err)
		return err
	}
	t.transformFn = fn
	return nil
}
