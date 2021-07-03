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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/qaengine"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/qri-io/starlib"
	"github.com/qri-io/starlib/bsoup"
	"github.com/qri-io/starlib/encoding/base64"
	"github.com/qri-io/starlib/encoding/csv"
	"github.com/qri-io/starlib/encoding/yaml"
	"github.com/qri-io/starlib/geo"
	"github.com/qri-io/starlib/hash"
	"github.com/qri-io/starlib/html"
	"github.com/qri-io/starlib/http"
	"github.com/qri-io/starlib/math"
	"github.com/qri-io/starlib/re"
	"github.com/qri-io/starlib/time"
	starutil "github.com/qri-io/starlib/util"
	"github.com/qri-io/starlib/xlsx"
	"github.com/qri-io/starlib/zipfile"
	"github.com/sirupsen/logrus"
	"go.starlark.net/starlark"
	starjson "go.starlark.net/starlarkjson"
)

const (
	baseDirectoryDetectFnName = "BaseDirectoryDetect"
	qaFunctionName            = "query"
)

// Starlark implements transformer interface and is used to write simple external transformers
type Starlark struct {
	TConfig     transformertypes.Transformer
	StarConfig  StarYamlConfig
	StarThread  *starlark.Thread
	StarGlobals starlark.StringDict
	Env         environment.Environment
}

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
	starGlobals, err := starlib.Loader()
	if err != nil {
		logrus.Errorf("Unable to load default modules for starlark : %s", err)
		return err
	}
	starGlobals["json"] = starjson.Module
	starGlobals[qaFunctionName] = starlark.NewBuiltin(qaFunctionName, t.query)

	t.StarThread = &starlark.Thread{Name: tc.Name}
	t.StarGlobals, err = starlark.ExecFile(t.StarThread, filepath.Join(t.Env.GetWorkspaceContext(), t.StarConfig.StarFile), nil, starGlobals)
	if err != nil {
		logrus.Errorf("Unable to load starlark file %s : %s", filepath.Join(t.Env.GetWorkspaceContext(), t.StarConfig.StarFile), err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Starlark) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *Starlark) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	if !t.StarGlobals.Has(baseDirectoryDetectFnName) {
		return nil, nil, nil
	}
	starDir, err := starutil.Marshal(dir)
	if err != nil {
		logrus.Errorf("Unable to concert %s to starlark value : %s", err)
		return nil, nil, err
	}
	starlark.Call(t.StarThread, t.StarGlobals[baseDirectoryDetectFnName], starlark.Tuple{starDir}, nil)
	return t.executeDetect(t.ExecConfig.BaseDirectoryDetectCMD, dir)
}

// DirectoryDetect runs detect in each sub directory
func (t *Starlark) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	if t.ExecConfig.DirectoryDetectCMD == nil {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.DirectoryDetectCMD, dir)
}

// Transform transforms the artifacts
func (t *Starlark) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
		if t.ExecConfig.TransformCMD == nil {
			relSrcPath, err := filepath.Rel(t.Env.GetWorkspaceSource(), a.Paths[artifacts.ProjectPathPathType][0])
			if err != nil {
				logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
			}
			var config interface{}
			if a.Configs != nil {
				config = a.Configs[artifacts.TemplateConfigType]
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.TemplatePathMappingType,
				SrcPath:        filepath.Join(t.Env.Context, t.Env.RelTemplatesDir),
				DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
				TemplateConfig: config,
			}, transformertypes.PathMapping{
				Type:     transformertypes.SourcePathMappingType,
				SrcPath:  "",
				DestPath: common.DefaultSourceDir,
			})
		} else {
			path := ""
			if a.Paths != nil && a.Paths[artifacts.ProjectPathPathType] != nil {
				path = a.Paths[artifacts.ProjectPathPathType][0]
			}
			return t.executeTransform(t.ExecConfig.TransformCMD, path)
		}
	}
	return pathMappings, nil, nil
}

func (t *Starlark) executeDetect(cmd environmenttypes.Command, dir string) (nameServices map[string]plantypes.Service, unservices []plantypes.Transformer, err error) {
	stdout, stderr, exitcode, err := t.Env.Exec(append(cmd, dir))
	if err != nil {
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, nil
	}
	logrus.Debugf("%s Detect succeeded in %s : %s, %s, %d", t.TConfig.Name, t.Env.Decode(dir), stdout, stderr, exitcode)
	stdout = strings.TrimSpace(stdout)
	trans := plantypes.Transformer{
		Mode:                   t.TConfig.Spec.Mode,
		ArtifactTypes:          t.TConfig.Spec.Artifacts,
		ExclusiveArtifactTypes: t.TConfig.Spec.ExclusiveArtifacts,
		Paths:                  map[string][]string{artifacts.ProjectPathPathType: {dir}},
		Configs:                map[transformertypes.ConfigType]interface{}{},
	}
	var config map[string]interface{}
	if stdout != "" {
		config = map[string]interface{}{}
		err = json.Unmarshal([]byte(stdout), &config)
		if err != nil {
			logrus.Debugf("Error in unmarshalling json %s: %s.", stdout, err)
		}
		trans.Configs[artifacts.TemplateConfigType] = config
	}
	return nil, []plantypes.Transformer{trans}, nil
}

func (t *Starlark) executeTransform(cmd environmenttypes.Command, dir string) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	stdout, stderr, exitcode, err := t.Env.Exec(append(cmd, dir))
	if err != nil {
		logrus.Errorf("Transform failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Transform did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, nil
	}
	logrus.Debugf("%s Transform succeeded in %s : %s, %s, %d", t.TConfig.Name, t.Env.Decode(dir), stdout, stderr, exitcode)
	stdout = strings.TrimSpace(stdout)
	var config TransformConfig
	err = json.Unmarshal([]byte(stdout), &config)
	if err != nil {
		logrus.Errorf("Error in unmarshalling json %s: %s.", stdout, err)
	}
	return config.PathMappings, config.Artifacts, nil
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
		return prob, fmt.Errorf("the key 'id' is missing from the question object %+v", argI)
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

func (t *Starlark) defaultGlobals() *starlark.StringDict {
	dict := starlark.StringDict{}
	switch module {
	case time.ModuleName:
		return starlark.StringDict{"time": time.Module}, nil
	case http.ModuleName:
		return http.LoadModule()
	case xlsx.ModuleName:
		return xlsx.LoadModule()
	case html.ModuleName:
		return html.LoadModule()
	case bsoup.ModuleName:
		return bsoup.LoadModule()
	case zipfile.ModuleName:
		return zipfile.LoadModule()
	case re.ModuleName:
		return re.LoadModule()
	case base64.ModuleName:
		return base64.LoadModule()
	case csv.ModuleName:
		return csv.LoadModule()
	case json.ModuleName:
		return starlark.StringDict{"json": json.Module}, nil
	case yaml.ModuleName:
		return yaml.LoadModule()
	case geo.ModuleName:
		return geo.LoadModule()
	case math.ModuleName:
		return starlark.StringDict{"math": math.Module}, nil
	case hash.ModuleName:
		return hash.LoadModule()
	case dataframe.ModuleName:
		return starlark.StringDict{"dataframe": dataframe.Module}, nil
	}
}
