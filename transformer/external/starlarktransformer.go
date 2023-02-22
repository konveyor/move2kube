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
	"strconv"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/antchfx/xpath"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/magiconair/properties"
	"github.com/qri-io/starlib"
	starutil "github.com/qri-io/starlib/util"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

const (
	directoryDetectFnName = "directory_detect"
	transformFnName       = "transform"

	sourceDirVarName         = "source_dir"
	contextDirVarName        = "context_dir"
	tempDirVarName           = "temp_dir"
	templatesRelDirVarName   = "templates_reldir"
	transformerConfigVarName = "config"
	projectVarName           = "project"
	resourcesDirVarName      = "resources_dir"
	outputDirVarName         = "output_dir"

	// Function names
	qaFnName = "query"
	// fs package
	fsExistsFnName               = "exists"
	fsReadFnName                 = "read"
	fsReadDirFnName              = "read_dir"
	fsIsDirFnName                = "is_dir"
	fsGetYamIsWithTypeMetaFnName = "get_yamls_with_type_meta"
	fsFindXmlPathFnName          = "find_xml_path"
	fsGetFilesWithPatternFnName  = "get_files_with_pattern"
	fsPathJoinFnName             = "path_join"
	fsReadPropertiesFnName       = "read_properties"
	fsWriteFnName                = "write"
	fsPathBaseFnName             = "path_base"
	fsPathRelFnName              = "path_rel"

	// encryption functions
	encAesCbcPbkdfFnName = "enc_aes_cbc_pbkdf"
	encRsaCertFnName     = "enc_rsa_cert"
	// archival functions
	archTarGZipStrFnName = "arch_tar_gzip_str"
	archTarStrFnName     = "arch_tar_str"
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
		return fmt.Errorf("failed to load config for Transformer %+v into %T . Error: %w", t.Config.Spec.Config, t.StarConfig, err)
	}
	t.StarThread = &starlark.Thread{Name: tc.Name}
	t.setDefaultGlobals()
	tcmapobj, err := common.GetMapInterfaceFromObj(tc)
	if err != nil {
		return fmt.Errorf("failed to convert transformer config to map[string]interface{}. Error: %w", err)
	}
	t.StarGlobals[env.ProjectName], err = starutil.Marshal(env.ProjectName)
	if err != nil {
		return fmt.Errorf("failed to load transformer config. Error: %w", err)
	}
	t.StarGlobals[transformerConfigVarName], err = starutil.Marshal(tcmapobj)
	if err != nil {
		return fmt.Errorf("failed to load transformer config. Error: %w", err)
	}
	t.StarGlobals[contextDirVarName], err = starutil.Marshal(env.GetEnvironmentContext())
	if err != nil {
		return fmt.Errorf("failed to load context. Error: %w", err)
	}
	t.StarGlobals[sourceDirVarName], err = starutil.Marshal(env.GetEnvironmentSource())
	if err != nil {
		return fmt.Errorf("failed to load source. Error: %w", err)
	}
	t.StarGlobals[outputDirVarName], err = starutil.Marshal(env.GetEnvironmentOutput())
	if err != nil {
		return fmt.Errorf("failed to load output. Error: %w", err)
	}
	t.StarGlobals[tempDirVarName], err = starutil.Marshal(env.TempPath)
	if err != nil {
		return fmt.Errorf("failed to load temp path. Error: %w", err)
	}
	t.StarGlobals[templatesRelDirVarName], err = starutil.Marshal(env.RelTemplatesDir)
	if err != nil {
		return fmt.Errorf("failed to load source. Error: %w", err)
	}
	t.StarGlobals[resourcesDirVarName], err = starutil.Marshal(filepath.Join(env.GetEnvironmentContext(), "resources"))
	if err != nil {
		return fmt.Errorf("failed to load source. Error: %w", err)
	}
	starlarkFilePath := filepath.Join(t.Env.GetEnvironmentContext(), t.StarConfig.StarFile)
	t.StarGlobals, err = starlark.ExecFile(t.StarThread, starlarkFilePath, nil, t.StarGlobals)
	if err != nil {
		if t.StarConfig.StarFile == "" {
			err = fmt.Errorf("no starlark file specified. Error: %w", err)
		} else {
			err = fmt.Errorf("failed to load starlark file at the path '%s' . Error: %w", starlarkFilePath, err)
		}
		return err
	}
	if err := t.loadFunctions(); err != nil {
		return fmt.Errorf("failed to load the required functions. Error: %w", err)
	}
	return nil
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
func (t *Starlark) Transform(
	newArtifacts []transformertypes.Artifact,
	alreadySeenArtifacts []transformertypes.Artifact,
) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	naObj, err := common.GetMapInterfaceFromObj(newArtifacts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert new artifacts to map[string]interface{} . Error: %w", err)
	}
	starNewArtifacts, err := starutil.Marshal(naObj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal the new artifacts %+v to starlark value. Error: %w", newArtifacts, err)
	}
	oaObj, err := common.GetMapInterfaceFromObj(alreadySeenArtifacts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert new artifacts to map[string]interface{} . Error: %w", err)
	}
	starOldArtifacts, err := starutil.Marshal(oaObj)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal already seen artifacts %+v to starlark value. Error: %w", alreadySeenArtifacts, err)
	}
	val, err := starlark.Call(t.StarThread, t.transformFn, starlark.Tuple{starNewArtifacts, starOldArtifacts}, nil)
	if err != nil {
		switch err := err.(type) {
		case *starlark.EvalError:
			return nil, nil, fmt.Errorf("failed to call the starlark function '%s' . The call stack is:\n%s\nError: %w", t.transformFn.String(), err.Backtrace(), err)
		}
		return nil, nil, fmt.Errorf("failed to call the starlark function '%s' . Error: %w", t.transformFn.String(), err)
	}
	valI, err := starutil.Unmarshal(val)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal the starlark value back to Golang value. Error: %w", err)
	}
	transformOutput := transformertypes.TransformOutput{}
	if err := common.GetObjFromInterface(valI, &transformOutput); err != nil {
		return nil, nil, fmt.Errorf("failed to convert the object of type %T and value %+v into the struct %T . Error: %w", valI, valI, transformOutput, err)
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
		var validation string
		if err := starlark.UnpackPositionalArgs(qaFnName, args, kwargs, 1, &argDictValue, &validation); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Expected a single dict argument. Error: %q", qaFnName, err)
		}
		argI, err := starutil.Unmarshal(argDictValue)
		if err != nil {
			return starlark.None, fmt.Errorf("failed to unmarshal the argument provided to '%s'. Expected a single dict argument. Error: %q", qaFnName, err)
		}
		prob := qatypes.Problem{}
		if err := common.GetObjFromInterface(argI, &prob); err != nil {
			return starlark.None, fmt.Errorf("failed to get the qa problem of type %T from the object of type %T and value %+v . Error: %w", prob, argI, argI, err)
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
		if validation != "" {
			validationFn, ok := t.StarGlobals[validation]
			if !ok {
				return starlark.None, fmt.Errorf("provided validation function not found : %s", validation)
			}
			fn, ok := validationFn.(*starlark.Function)
			if !ok {
				return starlark.None, fmt.Errorf("%s is not a function", validationFn)
			}
			prob.Validator = func(ans interface{}) error {
				answer, err := starutil.Marshal(ans)
				if err != nil {
					return fmt.Errorf("unable to convert %s to starlark value : %s", ans, err)
				}
				val, err := starlark.Call(t.StarThread, fn, starlark.Tuple{answer}, nil)
				if err != nil {
					return fmt.Errorf("unable to execute the starlark function: Error : %s", err)
				}
				value, err := starutil.Unmarshal(val)
				if err != nil {
					return fmt.Errorf("unable to unmarshal starlark function result : %s", err)
				}
				// if empty string is returned then we assume validation is successful
				if value.(string) != "" {
					return fmt.Errorf("validation failed : %s", value.(string))
				}
				return nil
			}
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
	t.addCryptoModules()
	t.addArchiveModules()
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
			fsExistsFnName:               t.getStarlarkFSExists(),
			fsReadFnName:                 t.getStarlarkFSRead(),
			fsReadDirFnName:              t.getStarlarkFSReadDir(),
			fsIsDirFnName:                t.getStarlarkFSIsDir(),
			fsGetFilesWithPatternFnName:  t.getStarlarkFSGetFilesWithPattern(),
			fsPathJoinFnName:             t.getStarlarkFSPathJoin(),
			fsReadPropertiesFnName:       t.getStarlarkFSReadProperties(),
			fsWriteFnName:                t.getStarlarkFSWrite(),
			fsGetYamIsWithTypeMetaFnName: t.getStarlarkFSGetYamlsWithTypeMeta(),
			fsPathBaseFnName:             t.getStarlarkFSPathBase(),
			fsPathRelFnName:              t.getStarlarkFSPathRel(),
			fsFindXmlPathFnName:          t.getStarlarkFindXmlPath(),
		},
	}
}

func (t *Starlark) addCryptoModules() {
	t.StarGlobals["crypto"] = &starlarkstruct.Module{
		Name: "crypto",
		Members: starlark.StringDict{
			encAesCbcPbkdfFnName: t.getStarlarkEncAesCbcPbkdf(),
			encRsaCertFnName:     t.getStarlarkEncRsaCert(),
		},
	}
}

func (t *Starlark) addArchiveModules() {
	t.StarGlobals["archive"] = &starlarkstruct.Module{
		Name: "archive",
		Members: starlark.StringDict{
			archTarGZipStrFnName: t.getStarlarkArchTarGZipStr(),
			archTarStrFnName:     t.getStarlarkArchTarStr(),
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

func (t *Starlark) loadFunctions() error {
	if err := t.loadDetectFn(); err != nil {
		return fmt.Errorf("failed to load detect function. Error: %w", err)
	}
	if err := t.loadTransformFn(); err != nil {
		return fmt.Errorf("failed to load transform function. Error: %w", err)
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
		return fmt.Errorf("%s is not a function", directoryDetectFn)
	}
	if fn.NumParams() != 1 {
		return fmt.Errorf("%s does not have the required number of paramters. It has %d, expected %d", directoryDetectFn, fn.NumParams(), 1)
	}
	t.detectFn = fn
	return nil
}

func (t *Starlark) loadTransformFn() (err error) {
	if !t.StarGlobals.Has(transformFnName) {
		return fmt.Errorf("no %s function found", transformFnName)
	}
	transformFn := t.StarGlobals[transformFnName]
	fn, ok := transformFn.(*starlark.Function)
	if !ok {
		return fmt.Errorf("%s is not a function", transformFn)
	}
	if fn.NumParams() != 2 {
		return fmt.Errorf("%s does not have the required number of paramters. It has %d, expected %d", transformFn, fn.NumParams(), 2)
	}
	t.transformFn = fn
	return nil
}

func (t *Starlark) getStarlarkFSGetYamlsWithTypeMeta() *starlark.Builtin {
	return starlark.NewBuiltin(fsGetYamIsWithTypeMetaFnName, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var inputPath string
		var kindFilter string
		if err := starlark.UnpackArgs(fsGetYamIsWithTypeMetaFnName, args, kwargs, "inputpath", &inputPath, "kind", &kindFilter); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Error: %w", fsGetYamIsWithTypeMetaFnName, err)
		}
		if kindFilter == "" {
			return starlark.None, fmt.Errorf("kind is missing in find parameters")
		}
		if !t.Env.IsPathValid(inputPath) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", inputPath)
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

func (t *Starlark) getStarlarkFindXmlPath() *starlark.Builtin {
	return starlark.NewBuiltin(fsFindXmlPathFnName, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var inputXmlFilePath string
		var xmlPathExpr string
		if err := starlark.UnpackArgs(fsFindXmlPathFnName, args, kwargs, "inputXmlFilePath", &inputXmlFilePath, "xmlpathexpr", &xmlPathExpr); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Error: %w", fsFindXmlPathFnName, err)
		}
		if xmlPathExpr == "" {
			return starlark.None, fmt.Errorf("XML path expression is missing in find parameters")
		}
		if !t.Env.IsPathValid(inputXmlFilePath) {
			return starlark.None, fmt.Errorf("invalid path: %s", inputXmlFilePath)
		}
		fileHandle, err := os.Open(inputXmlFilePath)
		if err != nil {
			return starlark.None, fmt.Errorf("could not read file in path: %s", inputXmlFilePath)
		}
		doc, err := xmlquery.Parse(fileHandle)
		if err != nil {
			return starlark.None, fmt.Errorf("could not parse xml file in path: %s", inputXmlFilePath)
		}
		expr, err := xpath.Compile(xmlPathExpr)
		if err != nil {
			return starlark.None, fmt.Errorf("could not compile the xml path expression: %s", xmlPathExpr)
		}
		data := expr.Evaluate(xmlquery.CreateXPathNavigator(doc))
		var result []interface{}
		switch d := data.(type) {
		case bool:
			result = append(result, cast.ToString(d))
		case float64:
			result = append(result, strconv.FormatFloat(d, 'E', -1, 64))
		case string:
			result = append(result, d)
		case *xpath.NodeIterator:
			iterator := data.(*xpath.NodeIterator)
			for iterator.MoveNext() {
				result = append(result, iterator.Current().Value())
			}
		}
		return starutil.Marshal(result)
	})
}

func (t *Starlark) getStarlarkFSWrite() *starlark.Builtin {
	return starlark.NewBuiltin(fsWriteFnName, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var filePath, data string
		var permissions = common.DefaultFilePermission
		if err := starlark.UnpackArgs(fsWriteFnName, args, kwargs, "filepath", &filePath, "data", &data, "perm?", &permissions); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Error: %w", fsWriteFnName, err)
		}
		if filePath == "" {
			return starlark.None, fmt.Errorf("FilePath is missing in write parameters")
		}
		if !t.Env.IsPathValid(filePath) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", filePath)
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
			return starlark.None, fmt.Errorf("failed to marshal the answer %+v of type %T into a starlark value. Error: %w", numBytesWritten, numBytesWritten, err)
		}
		return retValue, err
	})
}

func (t *Starlark) getStarlarkFSExists() *starlark.Builtin {
	return starlark.NewBuiltin(fsExistsFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsExistsFnName, args, kwargs, 1, &path); err != nil {
			return nil, fmt.Errorf("failed to unpack the positional arguments. Error: %w", err)
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", path)
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return starlark.Bool(false), nil
			}
			return starlark.Bool(false), fmt.Errorf("failed to stat the file at path '%s' . Error: %w", path, err)
		}
		return starlark.Bool(true), nil
	})
}

func (t *Starlark) getStarlarkFSIsDir() *starlark.Builtin {
	return starlark.NewBuiltin(fsIsDirFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsIsDirFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", path)
		}
		fileInfo, err := os.Stat(path)
		if err != nil {
			return starlark.None, fmt.Errorf("failed to stat the file at path '%s' . Error: %w", path, err)
		}
		return starlark.Bool(fileInfo.IsDir()), nil
	})
}

func (t *Starlark) getStarlarkFSRead() *starlark.Builtin {
	return starlark.NewBuiltin(fsReadFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsReadFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", path)
		}
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return starlark.None, nil
			}
			return nil, fmt.Errorf("failed to read the file at path '%s' . Error: %w", path, err)
		}
		return starlark.String(fileBytes), nil
	})
}

func (t *Starlark) getStarlarkFSReadDir() *starlark.Builtin {
	return starlark.NewBuiltin(fsReadDirFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsReadDirFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", path)
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

func (t *Starlark) getStarlarkFSGetFilesWithPattern() *starlark.Builtin {
	return starlark.NewBuiltin(fsGetFilesWithPatternFnName, func(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var filePath, extension string
		if err := starlark.UnpackArgs(fsGetFilesWithPatternFnName, args, kwargs, "filepath", &filePath, "ext", &extension); err != nil {
			return starlark.None, fmt.Errorf("invalid args provided to '%s'. Error: %w", fsGetFilesWithPatternFnName, err)
		}
		if filePath == "" {
			return starlark.None, fmt.Errorf("FilePath is missing in write parameters")
		}
		if !t.Env.IsPathValid(filePath) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		extList := []string{}
		extList = append(extList, extension)
		fileList, err := common.GetFilesByExt(filePath, extList)
		if err != nil {
			return starlark.None, fmt.Errorf("file list for given pattern could not be retrieved")
		}
		var result []interface{}
		for _, file := range fileList {
			result = append(result, file)
		}
		return starutil.Marshal(result)
	})
}

func (t *Starlark) getStarlarkFSPathJoin() *starlark.Builtin {
	return starlark.NewBuiltin(fsPathJoinFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var pathelem1, pathelem2 string
		if err := starlark.UnpackPositionalArgs(fsPathJoinFnName, args, kwargs, 2, &pathelem1, &pathelem2); err != nil {
			return nil, err
		}
		path := filepath.Join(pathelem1, pathelem2)
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", path)
		}
		return starutil.Marshal(path)
	})
}

func (t *Starlark) getStarlarkFSPathBase() *starlark.Builtin {
	return starlark.NewBuiltin(fsPathBaseFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var path string
		if err := starlark.UnpackPositionalArgs(fsPathBaseFnName, args, kwargs, 1, &path); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(path) {
			return starlark.None, fmt.Errorf("the path '%s' is invalid", path)
		}
		return starlark.String(filepath.Base(filepath.Clean(path))), nil
	})
}

func (t *Starlark) getStarlarkFSPathRel() *starlark.Builtin {
	return starlark.NewBuiltin(fsPathRelFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var basePath, targetPath string
		if err := starlark.UnpackPositionalArgs(fsPathRelFnName, args, kwargs, 2, &basePath, &targetPath); err != nil {
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
			return starlark.None, fmt.Errorf("failed to make the path '%s' to the base directory '%s' . Error: %w", targetPath, basePath, err)
		}
		return starlark.String(path3), nil
	})
}

func (t *Starlark) getStarlarkEncAesCbcPbkdf() *starlark.Builtin {
	return starlark.NewBuiltin(encAesCbcPbkdfFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var key, data string
		if err := starlark.UnpackPositionalArgs(encAesCbcPbkdfFnName, args, kwargs, 2, &key, &data); err != nil {
			return nil, err
		}
		return starlark.String(common.EncryptAesCbcWithPbkdfWrapper(key, data)), nil
	})
}

func (t *Starlark) getStarlarkEncRsaCert() *starlark.Builtin {
	return starlark.NewBuiltin(encRsaCertFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var certificate, data string
		if err := starlark.UnpackPositionalArgs(encRsaCertFnName, args, kwargs, 2, &certificate, &data); err != nil {
			return nil, err
		}
		return starlark.String(common.EncryptRsaCertWrapper(certificate, data)), nil
	})
}

func (t *Starlark) getStarlarkArchTarGZipStr() *starlark.Builtin {
	return starlark.NewBuiltin(archTarGZipStrFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var srcDir string
		if err := starlark.UnpackPositionalArgs(archTarGZipStrFnName, args, kwargs, 1, &srcDir); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(srcDir) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		return starlark.String(common.CreateTarArchiveGZipStringWrapper(srcDir)), nil
	})
}

func (t *Starlark) getStarlarkArchTarStr() *starlark.Builtin {
	return starlark.NewBuiltin(archTarStrFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var srcDir string
		if err := starlark.UnpackPositionalArgs(archTarStrFnName, args, kwargs, 1, &srcDir); err != nil {
			return nil, err
		}
		if !t.Env.IsPathValid(srcDir) {
			return starlark.None, fmt.Errorf("invalid path")
		}
		return starlark.String(common.CreateTarArchiveNoCompressionStringWrapper(srcDir)), nil
	})
}

func (t *Starlark) getStarlarkFSReadProperties() *starlark.Builtin {
	return starlark.NewBuiltin(fsReadPropertiesFnName, func(thread *starlark.Thread, fn *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
		var propertiesFilePath string
		if err := starlark.UnpackPositionalArgs(fsReadPropertiesFnName, args, kwargs, 1, &propertiesFilePath); err != nil {
			return nil, err
		}
		properties, err := properties.LoadFile(propertiesFilePath, properties.UTF8)
		if err != nil {
			logrus.Errorf("failed to parse the properties at path %s . Error: %q", propertiesFilePath, err)
		}
		propertiesMap := properties.Map()
		propertiesMap1 := make(map[string]interface{}, len(propertiesMap))
		for k, v := range propertiesMap {
			propertiesMap1[k] = v
		}
		return starutil.Marshal(propertiesMap1)
	})
}
