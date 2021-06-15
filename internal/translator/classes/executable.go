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

package classes

import (
	"encoding/json"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
)

const (
	SimpleExecType ExecType = "simple"
	FullExecType   ExecType = "full"
)

// Executable implements Containerizer interface
type Executable struct {
	TConfig    translatortypes.Translator
	ExecConfig ExecutableYamlConfig
	Env        environment.Environment
}

type ExecutableYamlConfig struct {
	Type                   string                     `yaml:"type,omitempty"` //simple, configfile, full - default simple
	Artifacts              []string                   `yaml:"artifacts"`
	ExclusiveArtifacts     []string                   `yaml:"exclusiveArtifacts"`
	BaseDirectoryDetectCMD environmenttypes.Command   `yaml:"baseDetectCMD"`
	DirectoryDetectCMD     environmenttypes.Command   `yaml:"directoryDetectCMD"`
	TranslateCMD           environmenttypes.Command   `yaml:"translateCMD"`
	Container              environmenttypes.Container `yaml:"container,omitempty"`
}

type ExecType string

type PlanTranslators struct {
	NamedServices   map[string]plantypes.Service `json:"named"`
	UnNamedServices []plantypes.Translator       `json:"unnamed"`
}

func (t *Executable) Init(tc translatortypes.Translator, env environment.Environment) (err error) {
	t.TConfig = tc
	t.ExecConfig = ExecutableYamlConfig{}
	err = common.GetObjFromInterface(t.TConfig.Spec.Config, &t.ExecConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Translator %+v into %T : %s", t.TConfig.Spec.Config, t.ExecConfig, err)
		return err
	}

	t.Env, err = environment.NewEnvironment(env.Name, env.GetSourcePath(), env.Context, t.ExecConfig.Container)
	if err != nil {
		logrus.Errorf("Unable to create Exec environment : %s", err)
		return err
	}
	return nil
}

func (t *Executable) GetConfig() (translatortypes.Translator, environment.Environment) {
	return t.TConfig, t.Env
}

func (t *Executable) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	if t.ExecConfig.BaseDirectoryDetectCMD.CMD == "" {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.BaseDirectoryDetectCMD, dir)
}

func (t *Executable) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	if t.ExecConfig.DirectoryDetectCMD.CMD == "" {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.DirectoryDetectCMD, dir)
}

func (t *Executable) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *Executable) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *Executable) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan) ([]translatortypes.Patch, error) {
	if t.ExecConfig.TranslateCMD.CMD == "" {
		return []translatortypes.Patch{{
			PathMappings: []translatortypes.PathMapping{{
				Type:     translatortypes.DefaultPathMappingType,
				SrcPath:  t.TConfig.Spec.TemplatesDir,
				DestPath: translatorPlan.Paths[plantypes.ProjectPathSourceArtifact][0],
			}},
			Config: translatorPlan.Config,
		}}, nil
	}
	artifacts := ""
	for _, a := range translatorPlan.ArtifactTypes {
		artifacts += a + ","
	}
	tstring, err := json.Marshal(translatorPlan)
	if err != nil {
		logrus.Errorf("Unable to convert translator to string : %s", err)
		return nil, err
	}
	var stdout, stderr string
	var exitcode int
	Cmd := environmenttypes.Command{
		CMD: t.ExecConfig.TranslateCMD.CMD,
	}
	if strings.ToLower(t.ExecConfig.Type) == string(FullExecType) {
		Cmd.Args = []string{t.Env.EncodePath(t.Env.GetSourcePath()), string(tstring)}
		stdout, stderr, exitcode, err = t.Env.Exec(Cmd, "")
	} else {
		Cmd.Args = []string{t.Env.EncodePath(t.Env.GetSourcePath()), artifacts}
		stdout, stderr, exitcode, err = t.Env.Exec(Cmd, "")
	}
	if err != nil {
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil
	}

	trans := translatortypes.Patch{}
	err = json.Unmarshal([]byte(stdout), &trans)
	if err != nil {
		logrus.Debugf("Error in unmarshalling json %s: %s.", stdout, err)
		trans := []translatortypes.Patch{}
		err = json.Unmarshal([]byte(stdout), &trans)
		if err != nil {
			logrus.Debugf("Error in unmarshalling json %s: %s.", stdout, err)
			config := map[string]interface{}{}
			err = json.Unmarshal([]byte(stdout), &config)
			if err != nil {
				return []translatortypes.Patch{{
					PathMappings: []translatortypes.PathMapping{{
						Type:     translatortypes.TemplatePathMappingType,
						SrcPath:  t.TConfig.Spec.TemplatesDir,
						DestPath: translatorPlan.Paths[plantypes.ProjectPathSourceArtifact][0],
					}},
					Config: config,
				}}, nil
			}
		}
		return trans, nil
	}
	return []translatortypes.Patch{trans}, nil
}

func (t *Executable) TranslateIR(ir irtypes.IR, plan plantypes.Plan) ([]translatortypes.PathMapping, error) {
	return nil, nil
}

func (t *Executable) executeDetect(cmd environmenttypes.Command, dir string) (nameServices map[string]plantypes.Service, unservices []plantypes.Translator, err error) {
	ncmd := environmenttypes.Command{
		CMD:  cmd.CMD,
		Args: []string{t.Env.EncodePath(t.Env.EncodePath(dir))},
	}
	stdout, stderr, exitcode, err := t.Env.Exec(ncmd, "")
	if err != nil {
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, nil
	}
	logrus.Debugf("%s Detect succeeded in %s : %s, %s, %d", t.TConfig.Name, t.Env.DecodePath(dir), stdout, stderr, exitcode)
	stdout = strings.TrimSpace(stdout)
	if strings.ToLower(t.ExecConfig.Type) == string(FullExecType) {
		trans := PlanTranslators{}
		err = json.Unmarshal([]byte(stdout), &trans)
		if err != nil {
			logrus.Debugf("Error in unmarshalling json %s: %s.", stdout, err)
			return trans.NamedServices, trans.UnNamedServices, err
		} else {
			return trans.NamedServices, trans.UnNamedServices, nil
		}
	}
	config := map[string]interface{}{}
	err = json.Unmarshal([]byte(stdout), &config)
	if err != nil {
		logrus.Debugf("Error in unmarshalling json %s: %s.", stdout, err)
	}
	trans := plantypes.Translator{
		Mode:                   string(t.TConfig.Spec.Mode),
		ArtifactTypes:          t.ExecConfig.Artifacts,
		ExclusiveArtifactTypes: t.ExecConfig.ExclusiveArtifacts,
		Paths:                  map[string][]string{plantypes.ProjectPathSourceArtifact: {dir}},
		Config:                 config,
	}
	return nil, []plantypes.Translator{trans}, nil
}
