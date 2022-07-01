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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/qri-io/starlib"
	starutil "github.com/qri-io/starlib/util"
	"github.com/sirupsen/logrus"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

const (
	baseDirectoryDetectFnName = "base_directory_detect"
	directoryDetectFnName     = "directory_detect"
	transformFnName           = "transform"

	sourceDirVarName         = "source_dir"
	contextDirVarName        = "context_dir"
	tempDirVarName           = "temp_dir"
	templatesRelDirVarName   = "templates_reldir"
	transformerConfigVarName = "config"
	projectVarName           = "project"

	// Function names
	qaFnName = "query"
	// fs package
	fsexistsFnName               = "exists"
	fsreadFnName                 = "read"
	fsreaddirFnName              = "readdir"
	fsgetyamlswithtypemetaFnName = "getyamlswithtypemeta"
	fspathjoinFnName             = "pathjoin"
	fswriteFnName                = "write"
	fspathBaseFnName             = "pathbase"
	fspathRelFnName              = "pathrel"
)

// Starlark implements transformer interface and is used to write simple external transformers
type Starlark struct {
	Config      transformertypes.Transformer
	StarConfig  *StarYamlConfig
	StarThread  *starlark.Thread
	StarGlobals starlark.StringDict
	Env         *environment.Environment

	detectFn    *starlark.Function
	transformFn *starlark.Function
}

// StarYamlConfig defines yaml config for Starlark transformers
type StarYamlConfig struct {
	StarFile string `yaml:"starFile"`
}

// Init Initializes the transformer
func (t *Starlark) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.StarConfig = &StarYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.StarConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.StarConfig, err)
		return err
	}
	t.StarThread = &starlark.Thread{Name: tc.Name}
	t.setDefaultGlobals()
	tcmapobj, err := common.GetMapInterfaceFromObj(tc)
	if err != nil {
		logrus.Errorf("Unable to convert transformer config to map[string]interface{}")
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
	t.StarGlobals[tempDirVarName], err = starutil.Marshal(env.TempPath)
	if err != nil {
		logrus.Errorf("Unable to load temp path : %s", err)
		return err
	}
	t.StarGlobals[templatesRelDirVarName], err = starutil.Marshal(env.RelTemplatesDir)
	if err != nil {
		logrus.Errorf("Unable to load source : %s", err)
		return err
	}
	t.StarGlobals, err = starlark.ExecFile(t.StarThread, filepath.Join(t.Env.GetEnvironmentContext(), t.StarConfig.StarFile), nil, t.StarGlobals)
	if err != nil {
		if t.StarConfig.StarFile == "" {
			logrus.Error("no starlark file specified")
		} else {
			logrus.Errorf("Unable to load starlark file %s : %s", filepath.Join(t.Env.GetEnvironmentContext(), t.StarConfig.StarFile), err)
		}
		return err
	}
	err = t.loadFunctions()
	if err != nil {
		logrus.Errorf("Unable to load required functions : %s", err)
	}
	return err
}

// GetConfig returns the transformer config
func (t *Starlark) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *Starlark) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	return t.executeDetect(t.detectFn, dir)
}

// Transform transforms the artifacts
func (t *Starlark) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	naObj, err := common.GetMapInterfaceFromObj(newArtifacts)
	if err != nil {
		logrus.Errorf("Unable to convert new artifacts to map[string]interface{}")
		return nil, nil, err
	}
	starNewArtifacts, err := starutil.Marshal(naObj)
	if err != nil {
		logrus.Errorf("Unable to convert %s to starlark value : %s", newArtifacts, err)
		return nil, nil, err
	}
	oaObj, err := common.GetMapInterfaceFromObj(alreadySeenArtifacts)
	if err != nil {
		logrus.Errorf("Unable to convert new artifacts to map[string]interface{}")
		return nil, nil, err
	}
	starOldArtifacts, err := starutil.Marshal(oaObj)
	if err != nil {
		logrus.Errorf("Unable to convert %s to starlark value : %s", alreadySeenArtifacts, err)
		return nil, nil, err
	}
	val, err := starlark.Call(t.StarThread, t.transformFn, starlark.Tuple{starNewArtifacts, starOldArtifacts}, nil)
	if err != nil {
		logrus.Errorf("failed to call the starlark function: %s Error: %q", t.transformFn.String(), err)
		return nil, nil, err
	}
	valI, err := starutil.Unmarshal(val)
	if err != nil {
		logrus.Errorf("Unable to unmarshal starlark function result : %s", err)
		return nil, nil, err
	}
	transformOutput := transformertypes.TransformOutput{}
	err = common.GetObjFromInterface(valI, &transformOutput)
	if err != nil {
		logrus.Errorf("unable to load result for Transformer %+v into %T : %s", valI, transformOutput, err)
		return nil, nil, err
	}
	return transformOutput.PathMappings, transformOutput.CreatedArtifacts, nil
}

func (t *Starlark) executeDetect(fn *starlark.Function, dir string) (services map[string][]transformertypes.Artifact, err error) {
	if fn == nil {
		return nil, nil
	}
	starDir, err := starutil.Marshal(dir)
	if err != nil {
		logrus.Errorf("Unable to convert %s to starlark value : %s", dir, err)
		return nil, err
	}
	val, err := starlark.Call(t.StarThread, fn, starlark.Tuple{starDir}, nil)
	if err != nil {
		logrus.Errorf("Unable to execute starlark function : %s", err)
		return nil, err
	}
	valI, err := starutil.Unmarshal(val)
	if err != nil {
		logrus.Errorf("Unable to unmarshal starlark function result : %s", err)
		return nil, err
	}
	services = map[string][]transformertypes.Artifact{}
	err = common.GetObjFromInterface(valI, &services)
	if err != nil {
		logrus.Errorf("unable to load result for Transformer %+v into %T : %s", valI, services, err)
		return nil, err
	}
	return services, nil
}

func (t *Starlark) getStarlarkQuery() *starlark.Builtin {
	return starlark.NewBuiltin(qaFnName, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		argDictValue := &starlark.Dict{}
		if err := starlark.UnpackPositionalArgs(qaFnName, args, kwargs, 1, &argDictValue); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Expected a single dict argument. Error: %q", qaFnName, err)
		}
		argI, err := starutil.Unmarshal(argDictValue)
		if err != nil {
			return starlark.None, fmt.Errorf("failed to unmarshal the argument provided to '%s'. Expected a single dict argument. Error: %q", qaFnName, err)
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
			prob.ID = common.JoinQASubKeys(common.BaseKey, prob.ID)
		}
		// type
		if prob.Type == "" {
			prob.Type = qatypes.InputSolutionFormType
		}
		resolved, err := qaengine.FetchAnswer(prob)
		if err != nil {
			logrus.Fatalf("failed to ask the question. Error: %q", err)
		}

		var answerValue starlark.Value
		if ansList, ok := resolved.Answer.([]string); ok {
			var result []interface{}
			for _, ans := range ansList {
				result = append(result, ans)
			}
			answerValue, err = starutil.Marshal(result)
			if err != nil {
				return starlark.None, fmt.Errorf("failed to marshal the answer %+v of type %T into a starlark value. Error: %q", resolved.Answer, resolved.Answer, err)
			}
		} else {
			answerValue, err = starutil.Marshal(resolved.Answer)
			if err != nil {
				return starlark.None, fmt.Errorf("failed to marshal the answer %+v of type %T into a starlark value. Error: %q", resolved.Answer, resolved.Answer, err)
			}
		}
		return answerValue, err
	})
}

func (t *Starlark) setDefaultGlobals() {
	t.StarGlobals = starlark.StringDict{}
	t.addStarlibModules()
	t.addFSModules()
	t.addAppModules()
}

func (t *Starlark) addStarlibModules() {
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

func (t *Starlark) addFSModules() {
	t.StarGlobals["fs"] = &starlarkstruct.Module{
		Name: "fs",
		Members: starlark.StringDict{
			fsexistsFnName:               t.getStarlarkFSExists(),
			fsreadFnName:                 t.getStarlarkFSRead(),
			fsreaddirFnName:              t.getStarlarkFSReadDir(),
			fspathjoinFnName:             t.getStarlarkFSPathJoin(),
			fswriteFnName:                t.getStarlarkFSWrite(),
			fsgetyamlswithtypemetaFnName: t.getStarlarkFSGetYamlsWithTypeMeta(),
			fspathBaseFnName:             t.getStarlarkFSPathBase(),
			fspathRelFnName:              t.getStarlarkFSPathRel(),
		},
	}
}

func (t *Starlark) addAppModules() {
	t.StarGlobals[types.AppNameShort] = &starlarkstruct.Module{
		Name: types.AppNameShort,
		Members: starlark.StringDict{
			qaFnName: t.getStarlarkQuery(),
		},
	}
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

func (t *Starlark) getStarlarkFSGetYamlsWithTypeMeta() *starlark.Builtin {
	return starlark.NewBuiltin(fsgetyamlswithtypemetaFnName, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var inputPath string
		var kindFilter string
		if err := starlark.UnpackArgs(fsgetyamlswithtypemetaFnName, args, kwargs, "inputpath", &inputPath, "kind", &kindFilter); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Error: %q", fswriteFnName, err)
		}
		if kindFilter == "" {
			return starlark.None, fmt.Errorf("kind is missing in find parameters")
		}
		if !t.Env.IsPathValid(inputPath) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		fileList, err := common.GetYamlsWithTypeMeta(inputPath, kindFilter)
		if err != nil {
			return starlark.None, err
		}
		var result []interface{}
		for _, filePath := range fileList {
			result = append(result, filePath)
		}
		return starutil.Marshal(result)
	})
}

func (t *Starlark) getStarlarkFSWrite() *starlark.Builtin {
	return starlark.NewBuiltin(fswriteFnName, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var filePath, data string
		var permissions = common.DefaultFilePermission
		if err := starlark.UnpackArgs(fswriteFnName, args, kwargs, "filepath", &filePath, "data", &data, "perm?", &permissions); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Error: %q", fswriteFnName, err)
		}
		if filePath == "" {
			return starlark.None, fmt.Errorf("FilePath is missing in write parameters")
		}
		if !t.Env.IsPathValid(filePath) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		if len(data) == 0 {
			return starlark.None, fmt.Errorf("data is missing in write parameters")
		}
		numBytesWritten := len(data)
		err := os.WriteFile(filePath, []byte(data), fs.FileMode(permissions))
		if err != nil {
			return starlark.None, fmt.Errorf("could not write to file %s", filePath)
		}
		retValue, err := starutil.Marshal(numBytesWritten)
		if err != nil {
			return starlark.None, fmt.Errorf("failed to marshal the answer %+v of type %T into a starlark value. Error: %q", numBytesWritten, numBytesWritten, err)
		}
		return retValue, err
	})
}

func (t *Starlark) getStarlarkFSExists() *starlark.Builtin {
	return starlark.NewBuiltin(qaFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsexistsFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		_, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				return starlark.Bool(false), nil
			}
			logrus.Errorf("Unable to check if file exists : %s", err)
			return starlark.Bool(false), err
		}
		return starlark.Bool(true), nil
	})
}

func (t *Starlark) getStarlarkFSRead() *starlark.Builtin {
	return starlark.NewBuiltin(fsreadFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsreadFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return starlark.None, nil
			}

			return nil, err
		}
		return starlark.String(fileBytes), nil
	})
}

func (t *Starlark) getStarlarkFSReadDir() *starlark.Builtin {
	return starlark.NewBuiltin(fsreadFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsreadFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		fileInfos, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		var result []interface{}
		for _, fileInfo := range fileInfos {
			result = append(result, fileInfo.Name())
		}
		return starutil.Marshal(result)
	})
}

func (t *Starlark) getStarlarkFSPathJoin() *starlark.Builtin {
	return starlark.NewBuiltin(fspathjoinFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var pathelem1, pathelem2 string
		if err := starlark.UnpackPositionalArgs(fspathjoinFnName, args, kwargs, 2, &pathelem1, &pathelem2); err != nil {
			return nil, err
		}
		path := filepath.Join(pathelem1, pathelem2)
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		return starutil.Marshal(path)
	})
}

func (t *Starlark) getStarlarkFSPathBase() *starlark.Builtin {
	return starlark.NewBuiltin(fspathBaseFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fspathBaseFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		return starlark.String(filepath.Base(filepath.Clean(path))), nil
	})
}

func (t *Starlark) getStarlarkFSPathRel() *starlark.Builtin {
	return starlark.NewBuiltin(fspathRelFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var basePath, targetPath string
		if err := starlark.UnpackPositionalArgs(fspathRelFnName, args, kwargs, 2, &basePath, &targetPath); err != nil {
			return nil, err
		}
		basePath = filepath.Clean(basePath)
		targetPath = filepath.Clean(targetPath)
		if !t.Env.IsPathValid(basePath) {
			return starlark.None, fmt.Errorf("the base path '%s' is invalid", basePath)
		}
		if !t.Env.IsPathValid(targetPath) {
			return starlark.None, fmt.Errorf("the target path '%s' is invalid", targetPath)
		}
		path3, err := filepath.Rel(basePath, targetPath)
		if err != nil {
			return starlark.None, fmt.Errorf("failed to make the path '%s' to the base directory '%s' . Error: %q", targetPath, basePath, err)
		}
		return starlark.String(path3), nil
	})
}
