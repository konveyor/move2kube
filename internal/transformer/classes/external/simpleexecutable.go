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

package external

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

const (
	TransformConfigType plantypes.ConfigType = "TransformConfig"
)

// Executable implements Containerizer interface
type SimpleExecutable struct {
	TConfig    transformertypes.Transformer
	ExecConfig ExecutableYamlConfig
	Env        environment.Environment
}

type TransformConfig struct {
	PathMappings []transformertypes.PathMapping `json:"pathMappings,omitempty"`
	Artifacts    []transformertypes.Artifact    `json:"artifacts,omitempty"`
}

type ExecutableYamlConfig struct {
	BaseDirectoryDetectCMD environmenttypes.Command   `yaml:"baseDetectCMD"`
	DirectoryDetectCMD     environmenttypes.Command   `yaml:"directoryDetectCMD"`
	TransformCMD           environmenttypes.Command   `yaml:"transformCMD"`
	Container              environmenttypes.Container `yaml:"container,omitempty"`
}

func (t *SimpleExecutable) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.TConfig = tc
	t.ExecConfig = ExecutableYamlConfig{}
	err = common.GetObjFromInterface(t.TConfig.Spec.Config, &t.ExecConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.TConfig.Spec.Config, t.ExecConfig, err)
		return err
	}
	t.Env, err = environment.NewEnvironment(env.Name, env.Source, env.Context, t.ExecConfig.Container)
	if err != nil {
		logrus.Errorf("Unable to create Exec environment : %s", err)
		return err
	}
	return nil
}

func (t *SimpleExecutable) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

func (t *SimpleExecutable) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	if t.ExecConfig.BaseDirectoryDetectCMD == nil {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.BaseDirectoryDetectCMD, dir)
}

func (t *SimpleExecutable) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	if t.ExecConfig.DirectoryDetectCMD == nil {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.DirectoryDetectCMD, dir)
}

func (t *SimpleExecutable) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		if a.Artifact != transformertypes.ServiceArtifactType {
			continue
		}
		if t.ExecConfig.TransformCMD == nil {
			relSrcPath, err := filepath.Rel(t.Env.GetWorkspaceSource(), a.Paths[plantypes.ProjectPathPathType][0])
			if err != nil {
				logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[plantypes.ProjectPathPathType][0], err)
			}
			var config interface{}
			if a.Configs != nil {
				config = a.Configs[transformertypes.TemplateConfigType]
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:           transformertypes.TemplatePathMappingType,
				SrcPath:        filepath.Join(t.Env.Context, t.TConfig.Spec.TemplatesDir),
				DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
				TemplateConfig: config,
			}, transformertypes.PathMapping{
				Type:     transformertypes.SourcePathMappingType,
				SrcPath:  "",
				DestPath: common.DefaultSourceDir,
			})
		} else {
			path := ""
			if a.Paths != nil && a.Paths[plantypes.ProjectPathPathType] != nil {
				path = a.Paths[plantypes.ProjectPathPathType][0]
			}
			return t.executeTransform(t.ExecConfig.TransformCMD, path)
		}
	}
	return pathMappings, nil, nil
}

func (t *SimpleExecutable) executeDetect(cmd environmenttypes.Command, dir string) (nameServices map[string]plantypes.Service, unservices []plantypes.Transformer, err error) {
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
		Mode:                   string(t.TConfig.Spec.Mode),
		ArtifactTypes:          t.TConfig.Spec.Artifacts,
		ExclusiveArtifactTypes: t.TConfig.Spec.ExclusiveArtifacts,
		Paths:                  map[string][]string{plantypes.ProjectPathPathType: {dir}},
		Configs:                map[plantypes.ConfigType]interface{}{},
	}
	var config map[string]interface{}
	if stdout != "" {
		config = map[string]interface{}{}
		err = json.Unmarshal([]byte(stdout), &config)
		if err != nil {
			logrus.Debugf("Error in unmarshalling json %s: %s.", stdout, err)
		}
		trans.Configs[transformertypes.TemplateConfigType] = config
	}
	return nil, []plantypes.Transformer{trans}, nil
}

func (t *SimpleExecutable) executeTransform(cmd environmenttypes.Command, dir string) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
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
