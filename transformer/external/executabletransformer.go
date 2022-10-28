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
	"net"
	"os"
	"path/filepath"

	"github.com/dchest/uniuri"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/qaengine/questionreceivers"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// Executable implements transformer interface and is used to write simple external transformers
type Executable struct {
	Config     transformertypes.Transformer
	Env        *environment.Environment
	ExecConfig *ExecutableYamlConfig
}

// ExecutableYamlConfig is the format of executable yaml config
type ExecutableYamlConfig struct {
	EnableQA           bool                       `yaml:"enableQA"`
	Platforms          []string                   `yaml:"platforms"`
	DirectoryDetectCMD environmenttypes.Command   `yaml:"directoryDetectCMD"`
	TransformCMD       environmenttypes.Command   `yaml:"transformCMD"`
	Container          environmenttypes.Container `yaml:"container,omitempty"`
}

const (
	detectContainerOutputDir    = "/var/tmp/m2k_detect_output"
	detectInputFile             = "m2k_detect_input.json"
	detectOutputFile            = "m2k_detect_output.json"
	transformContainerOutputDir = "/var/tmp/m2k_transform_output"
	transformInputFile          = "m2k_transform_input.json"
	transformOutputFile         = "m2k_transform_output.json"
	// TemplateConfigType represents the template config type
	TemplateConfigType transformertypes.ConfigType = "TemplateConfig"
)

// Init Initializes the transformer
func (t *Executable) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.ExecConfig = &ExecutableYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, t.ExecConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.ExecConfig, err)
		return err
	}
	var qaRPCReceiverAddr net.Addr
	if t.ExecConfig.EnableQA {
		qaRPCReceiverAddr, err = questionreceivers.StartGRPCReceiver()
		if err != nil {
			logrus.Errorf("Unable to start QA RPC Receiver engine : %s", err)
			logrus.Infof("Starting transformer that requires QA without QA.")
		}
	}
	env.EnvInfo.EnvPlatformConfig = environmenttypes.EnvPlatformConfig{Container: t.ExecConfig.Container,
		Platforms: t.ExecConfig.Platforms}
	t.Env, err = environment.NewEnvironment(env.EnvInfo, qaRPCReceiverAddr)
	if err != nil {
		logrus.Errorf("Unable to create Exec environment : %s", err)
		return err
	}
	return nil
}

// GetConfig returns the transformer config
func (t *Executable) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *Executable) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	if t.ExecConfig.DirectoryDetectCMD == nil {
		return nil, nil
	}
	containerInputDir, err := t.uploadInput(map[string]string{"InputDirectory": dir}, detectInputFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to copy detect input path to container : %s", err)
	}
	services, err = t.executeDetect(t.ExecConfig.DirectoryDetectCMD,
		filepath.Join(containerInputDir, detectInputFile))
	if err != nil {
		return services, err
	}
	for sn, ns := range services {
		for nsi, nst := range ns {
			if len(nst.Paths) == 0 {
				nst.Paths = map[transformertypes.PathType][]string{
					artifacts.ServiceDirPathType: {dir},
				}
				ns[nsi] = nst
			}
		}
		services[sn] = ns
	}
	return services, err
}

// Transform transforms the artifacts
func (t *Executable) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	createdArtifacts = []transformertypes.Artifact{}
	if t.ExecConfig.TransformCMD == nil {
		return nil, nil, fmt.Errorf("No transform script specified : %s", err)
	}
	containerInputDir, err := t.uploadInput(transformertypes.TransformInput{NewArtifacts: newArtifacts,
		AlreadySeenArtifacts: alreadySeenArtifacts}, transformInputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("Unable to copy transform input path to container : %s", err)
	}
	stdout, stderr, exitcode, err := t.Env.Exec(append(t.ExecConfig.TransformCMD,
		filepath.Join(containerInputDir, transformInputFile)))
	if err != nil {
		if errors.Is(err, &environment.EnvironmentNotActiveError{}) {
			logrus.Errorf("%s", err)
			return nil, nil, err
		}
		logrus.Errorf("Transform failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Errorf("Transform did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	}
	logrus.Debugf("%s Transform succeeded in %s : %s, %d", t.Config.Name, stdout, stderr, exitcode)
	outputPath, err := t.Env.Env.Download(transformContainerOutputDir)
	if err != nil {
		logrus.Errorf("Error in downloading json %s: %s.", outputPath, err)
		return nil, nil, err
	}
	logrus.Debugf("Output containerized transformer JSON path: %v", outputPath)
	var output transformertypes.TransformOutput
	err = common.ReadJSON(filepath.Join(outputPath, transformOutputFile), &output)
	if err != nil {
		logrus.Errorf("Error in unmarshalling transformer output json %s: %s.", stdout, err)
		return nil, nil, err
	}
	pathMappings = append(pathMappings, output.PathMappings...)
	createdArtifacts = append(createdArtifacts, output.CreatedArtifacts...)

	return pathMappings, createdArtifacts, nil
}

func (t *Executable) executeDetect(cmd environmenttypes.Command, dir string) (services map[string][]transformertypes.Artifact, err error) {
	stdout, stderr, exitcode, err := t.Env.Exec(append(cmd, dir))
	if err != nil {
		if errors.Is(err, &environment.EnvironmentNotActiveError{}) {
			logrus.Debugf("%s", err)
			return nil, err
		}
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d", stdout, stderr, exitcode)
		return nil, nil
	}
	logrus.Debugf("%s Detect succeeded in %s : %s, %s, %d", t.Config.Name, dir, stdout, stderr, exitcode)
	outputPath, err := t.Env.Env.Download(detectContainerOutputDir)
	if err != nil {
		logrus.Errorf("Error in downloading json %s: %s.", outputPath, err)
		return nil, err
	}
	logrus.Debugf("Output detect JSON path: %v", outputPath)
	var output map[string][]transformertypes.Artifact
	err = common.ReadJSON(filepath.Join(outputPath, detectOutputFile), &output)
	if err != nil {
		logrus.Warnf("Error in unmarshalling detect output json %s: %s. Might be config type", stdout, err)
		config := map[string]interface{}{}
		err = common.ReadJSON(filepath.Join(outputPath, detectOutputFile), &config)
		if err != nil {
			logrus.Warnf("Error in unmarshalling config json %s.", err)
		}
		trans := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{artifacts.ServiceDirPathType: {dir}},
			Configs: map[transformertypes.ConfigType]interface{}{
				TemplateConfigType: config,
			},
		}
		return map[string][]transformertypes.Artifact{"": {trans}}, nil
	}
	return output, nil
}

func (t *Executable) uploadInput(data interface{}, inputFile string) (string, error) {
	inputDirPath := filepath.Join(t.Env.TempPath, uniuri.NewLen(5))
	os.MkdirAll(inputDirPath, common.DefaultDirectoryPermission)
	inputFilePath := filepath.Join(inputDirPath, inputFile)
	err := common.WriteJSON(inputFilePath, data)
	if err != nil {
		return "", fmt.Errorf("Unable to create input json : %s", err)
	}
	containerInputDir, err := t.Env.Env.Upload(inputDirPath)
	if err != nil {
		return "", fmt.Errorf("Unable to copy input dir to new container image : %s", err)
	}
	return containerInputDir, nil
}
