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
	"net"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/qaengine"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const (
	// TransformConfigType stores the config type representing transform
	TransformConfigType transformertypes.ConfigType = "TransformConfig"
)

// SimpleExecutable implements transformer interface and is used to write simple external transformers
type SimpleExecutable struct {
	TConfig    transformertypes.Transformer
	ExecConfig ExecutableYamlConfig
	Env        *environment.Environment
}

// TransformConfig defines the type of config for Simpleexecutable
type TransformConfig struct {
	PathMappings []transformertypes.PathMapping `json:"pathMappings,omitempty"`
	Artifacts    []transformertypes.Artifact    `json:"artifacts,omitempty"`
}

// ExecutableYamlConfig is the format of executable yaml config
type ExecutableYamlConfig struct {
	EnableQA               bool                       `yaml:"enableQA"`
	BaseDirectoryDetectCMD environmenttypes.Command   `yaml:"baseDetectCMD"`
	DirectoryDetectCMD     environmenttypes.Command   `yaml:"directoryDetectCMD"`
	TransformCMD           environmenttypes.Command   `yaml:"transformCMD"`
	Container              environmenttypes.Container `yaml:"container,omitempty"`
}

// Init Initializes the transformer
func (t *SimpleExecutable) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.TConfig = tc
	t.ExecConfig = ExecutableYamlConfig{}
	err = common.GetObjFromInterface(t.TConfig.Spec.Config, &t.ExecConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.TConfig.Spec.Config, t.ExecConfig, err)
		return err
	}
	var qaRPCReceiverAddr net.Addr = nil
	if t.ExecConfig.EnableQA {
		qaRPCReceiverAddr, err = qaengine.StartGRPCReceiver()
		if err != nil {
			logrus.Errorf("Unable to start QA RPC Receiver engine : %s", err)
			logrus.Infof("Starting transformer that requires QA without QA.")
		}
	}
	t.Env, err = environment.NewEnvironment(env.Name, env.ProjectName, env.Source, env.Output, env.Context, tc.Spec.TemplatesDir, qaRPCReceiverAddr, t.ExecConfig.Container)
	if err != nil {
		logrus.Errorf("Unable to create Exec environment : %s", err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *SimpleExecutable) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.TConfig, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *SimpleExecutable) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	if t.ExecConfig.BaseDirectoryDetectCMD == nil {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.BaseDirectoryDetectCMD, dir)
}

// DirectoryDetect runs detect in each sub directory
func (t *SimpleExecutable) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	if t.ExecConfig.DirectoryDetectCMD == nil {
		return nil, nil, nil
	}
	return t.executeDetect(t.ExecConfig.DirectoryDetectCMD, dir)
}

// Transform transforms the artifacts
func (t *SimpleExecutable) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}
		if t.ExecConfig.TransformCMD == nil {
			relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
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
		Mode:              t.TConfig.Spec.Mode,
		ArtifactTypes:     t.TConfig.Spec.Artifacts,
		BaseArtifactTypes: t.TConfig.Spec.GeneratedBaseArtifacts,
		Paths:             map[string][]string{artifacts.ProjectPathPathType: {dir}},
		Configs:           map[transformertypes.ConfigType]interface{}{},
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
