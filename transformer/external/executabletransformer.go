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

const (
	envDelimiter              = "="
	detectInputPathEnvKey     = "M2K_DETECT_INPUT_PATH"
	detectOutputPathEnvKey    = "M2K_DETECT_OUTPUT_PATH"
	transformInputPathEnvKey  = "M2K_TRANSFORM_INPUT_PATH"
	transformOutputPathEnvKey = "M2K_TRANSFORM_OUTPUT_PATH"
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
func (t *Executable) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.ExecConfig = &ExecutableYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.ExecConfig); err != nil {
		return fmt.Errorf("unable to load config for Transformer %+v into %T . Error: %q", t.Config.Spec.Config, t.ExecConfig, err)
	}
	var qaRPCReceiverAddr net.Addr
	var err error
	if t.ExecConfig.EnableQA {
		qaRPCReceiverAddr, err = questionreceivers.StartGRPCReceiver()
		if err != nil {
			logrus.Errorf("failed to start the QA GRPC Receiver engine. Error: %q", err)
			logrus.Infof("Starting transformer that requires QA without QA.")
		}
	}
	env.EnvInfo.EnvPlatformConfig = environmenttypes.EnvPlatformConfig{
		Container: t.ExecConfig.Container,
		Platforms: t.ExecConfig.Platforms,
	}
	t.Env, err = environment.NewEnvironment(env.EnvInfo, qaRPCReceiverAddr)
	if err != nil {
		return fmt.Errorf("failed to create the environment for the executable transformer. Error: %w", err)
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
		return nil, fmt.Errorf("failed to copy the detect input path to container. Error: %w", err)
	}
	services, err = t.executeDetect(
		t.ExecConfig.DirectoryDetectCMD,
		filepath.Join(containerInputDir, detectInputFile),
		filepath.Join(detectContainerOutputDir, detectOutputFile),
	)
	if err != nil {
		return services, fmt.Errorf("failed to execute the detect script. Error: %w", err)
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
	return services, nil
}

// Transform transforms the artifacts
func (t *Executable) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	if t.ExecConfig.TransformCMD == nil {
		return nil, nil, fmt.Errorf("no transform script specified")
	}
	containerInputDir, err := t.uploadInput(
		transformertypes.TransformInput{
			NewArtifacts:         newArtifacts,
			AlreadySeenArtifacts: alreadySeenArtifacts,
		},
		transformInputFile,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to upload the transform input into the environment at the path '%s' . Error: %w", transformInputFile, err)
	}
	transformInputPath := filepath.Join(containerInputDir, transformInputFile)
	transformOutputPath := filepath.Join(transformContainerOutputDir, transformOutputFile)
	cmdToRun, envList := t.configIO(
		t.ExecConfig.TransformCMD,
		map[string]string{
			transformInputPathEnvKey:  transformInputPath,
			transformOutputPathEnvKey: transformOutputPath,
		},
	)
	stdout, stderr, exitcode, err := t.Env.Exec(cmdToRun, envList)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to run the transform.\nstdout: %s\nstderr: %s\nexit code: %d . Error: %w", stdout, stderr, exitcode, err)
	}
	if exitcode != 0 {
		return nil, nil, fmt.Errorf("the transform script failed with non-zero exit code.\nstdout: %s\nstderr: %s\nexit code: %d", stdout, stderr, exitcode)
	}
	logrus.Debugf("the transform script '%s' succeeded.\nstdout: %s\nstderr: %s\nexit code: %d", t.Config.Name, stdout, stderr, exitcode)
	outputPath, err := t.Env.Env.Download(transformContainerOutputDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download the json %s . Error: %q", outputPath, err)
	}
	logrus.Debugf("Output containerized transformer JSON path: %v", outputPath)
	var output transformertypes.TransformOutput
	jsonOutputPath := filepath.Join(outputPath, transformOutputFile)
	if err := common.ReadJSON(jsonOutputPath, &output); err != nil {
		return nil, nil, fmt.Errorf("failed to parse the transformer output file at path '%s' as json. Error: %w", jsonOutputPath, err)
	}
	pathMappings = append(pathMappings, output.PathMappings...)
	createdArtifacts = append(createdArtifacts, output.CreatedArtifacts...)
	return pathMappings, createdArtifacts, nil
}

func (t *Executable) executeDetect(
	cmd environmenttypes.Command,
	inputPath string,
	outputPath string,
) (services map[string][]transformertypes.Artifact, err error) {
	cmdToRun, envList := t.configIO(
		t.ExecConfig.DirectoryDetectCMD,
		map[string]string{
			detectInputPathEnvKey:  inputPath,
			detectOutputPathEnvKey: outputPath,
		},
	)
	stdout, stderr, exitcode, err := t.Env.Exec(cmdToRun, envList)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the command in the environment.\nstdout: %s\nstderr: %s\nexit code: %d\nError: %w", stdout, stderr, exitcode, err)
	} else if exitcode != 0 {
		return nil, fmt.Errorf("the detect command failed with a non-zero exit code.\nstdout: %s\nstderr: %s\nexit code: %d", stdout, stderr, exitcode)
	}
	logrus.Debugf("%s Detect succeeded in %s : %s, %s, %d",
		t.Config.Name, inputPath, stdout, stderr, exitcode)
	outputPathFromContainer, err := t.Env.Env.Download(detectContainerOutputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download the json output at path '%s' from the environment. Error: %w", outputPathFromContainer, err)
	}
	logrus.Debugf("Output detect JSON path: %v", outputPathFromContainer)
	output := map[string][]transformertypes.Artifact{}
	jsonOutputPath := filepath.Join(outputPathFromContainer, detectOutputFile)
	if err := common.ReadJSON(jsonOutputPath, &output); err != nil {
		logrus.Warnf("failed in unmarshal the detect output file at path '%s' as json. Trying with config type. Error: %q", jsonOutputPath, err)
		config := map[string]interface{}{}
		if err := common.ReadJSON(jsonOutputPath, &config); err != nil {
			logrus.Warnf("failed in unmarshal the detect output file at path '%s' as config json. Error: %q", jsonOutputPath, err)
		}
		trans := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{artifacts.ServiceDirPathType: {inputPath}},
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
	if err := common.WriteJSON(inputFilePath, data); err != nil {
		return "", fmt.Errorf("failed to create the input json. Error: %w", err)
	}
	containerInputDir, err := t.Env.Env.Upload(inputDirPath)
	if err != nil {
		return "", fmt.Errorf("failed to copy input dir to new container image. Error: %w", err)
	}
	return containerInputDir, nil
}

func (t *Executable) configIO(
	cmd environmenttypes.Command,
	kvMap map[string]string) (
	environmenttypes.Command,
	[]string,
) {
	const dollarPrefix = "$"
	cmdToRun := cmd
	for envKey, value := range kvMap {
		for index, token := range cmdToRun {
			if token == dollarPrefix+envKey {
				cmdToRun[index] = value
			}
		}
	}
	envList := []string{}
	for envKey, value := range kvMap {
		envList = append(envList, envKey+envDelimiter+value)
	}
	return cmdToRun, envList
}
