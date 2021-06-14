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
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
)

const (
	SimpleExecType ExecType = "simple"
)

// Executable implements Containerizer interface
type Executable struct {
	TConfig    translatortypes.Translator
	ExecConfig ExecutableYamlConfig
	Env        environment.Environment
}

type ExecutableYamlConfig struct {
	Type                   string                `yaml:"type,omitempty"` //simple, configfile, full - default simple
	BaseDirectoryDetectCMD string                `yaml:"baseDetectCMD"`
	DirectoryDetectCMD     string                `yaml:"directoryDetectCMD"`
	TranslateCMD           string                `yaml:"translateCMD"`
	Container              environment.Container `yaml:"container,omitempty"`
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

	t.Env, err = environment.NewEnvironment(tc.Name, t.Env.GetSourcePath(), t.Env.Context, &t.ExecConfig.Container)
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
	if t.ExecConfig.BaseDirectoryDetectCMD == "" {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.BaseDirectoryDetectCMD, dir)
}

func (t *Executable) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	if t.ExecConfig.DirectoryDetectCMD == "" {
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
	if t.ExecConfig.TranslateCMD == "" {
		return []translatortypes.Patch{{
			PathMappings: []translatortypes.PathMapping{{
				Type:     translatortypes.DefaultPathMappingType,
				SrcPath:  "files",
				DestPath: translatorPlan.Paths[plantypes.ProjectPathSourceArtifact][0],
			}},
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
	if strings.ToLower(t.ExecConfig.Type) != string(SimpleExecType) {
		stdout, stderr, exitcode, err = t.Env.Exec(t.ExecConfig.TranslateCMD+" "+t.Env.EncodePath(t.Env.GetSourcePath())+" "+string(tstring), "")
	} else {
		stdout, stderr, exitcode, err = t.Env.Exec(t.ExecConfig.TranslateCMD+" "+t.Env.EncodePath(t.Env.GetSourcePath())+" "+artifacts, "")
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
						SrcPath:  "templates",
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

func (t *Executable) executeDetect(cmd string, dir string) (nameServices map[string]plantypes.Service, unservices []plantypes.Translator, err error) {
	stdout, stderr, exitcode, err := t.Env.Exec(cmd+" "+t.Env.EncodePath(t.Env.GetSourcePath()), "")
	if err != nil {
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, nil
	}
	stdout = strings.TrimSpace(stdout)
	if strings.ToLower(t.ExecConfig.Type) != string(SimpleExecType) {
		trans := PlanTranslators{}
		err = json.Unmarshal([]byte(stdout), &trans)
		if err != nil {
			logrus.Debugf("Error in unmarshalling json file %s: %s.", stdout, err)
			return trans.NamedServices, trans.UnNamedServices, err
		} else {
			return trans.NamedServices, trans.UnNamedServices, nil
		}
	}
	trans := plantypes.Translator{
		Mode:                   string(t.TConfig.Spec.Mode),
		ArtifactTypes:          strings.Fields(stdout),
		ExclusiveArtifactTypes: strings.Fields(stdout),
		Paths:                  map[string][]string{plantypes.ProjectPathSourceArtifact: {dir}},
	}
	return nil, []plantypes.Translator{trans}, nil
}
