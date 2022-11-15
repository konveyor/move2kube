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

// IOMode is the type of the mode used for I/O of detect and transform scripts.
type IOMode string

const (
	envMode      IOMode = "env"
	argMode      IOMode = "arg"
	envDelimiter        = "="
)

// Executable implements transformer interface and is used to write simple external transformers
type Executable struct {
	Config     transformertypes.Transformer
	Env        *environment.Environment
	ExecConfig *ExecutableYamlConfig
}

// InputFormat input/output mode definition
type InputFormat struct {
	Mode    IOMode `yaml:"mode"`
	EnvName string `yaml:"envName,omitempty`
	Path    string
}

// OutputFormat input/output mode definition
type OutputFormat struct {
	Mode    IOMode `yaml:"mode"`
	EnvName string `yaml:"envName,omitempty`
	Path    string
}

// CommandFormat input/output mode and command definitions for detect and transform scripts
type CommandFormat struct {
	Cmd    environmenttypes.Command `yaml:"cmd"`
	Input  InputFormat              `yaml:"input"`
	Output OutputFormat             `yaml:"output"`
}

// ExecutableYamlConfig is the format of executable yaml config
type ExecutableYamlConfig struct {
	EnableQA           bool                       `yaml:"enableQA"`
	Platforms          []string                   `yaml:"platforms"`
	Container          environmenttypes.Container `yaml:"container,omitempty"`
	DetectCmdFormat    CommandFormat              `yaml:"directoryDetect"`
	TransformCmdFormat CommandFormat              `yaml:"transform"`
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
	if t.ExecConfig.DetectCmdFormat.Cmd == nil {
		return nil, nil
	}
	containerInputDir, err := t.uploadInput(map[string]string{"InputDirectory": dir}, detectInputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to copy the detect input path to container. Error: %w", err)
	}
	t.ExecConfig.DetectCmdFormat.Input.Path = filepath.Join(containerInputDir, detectInputFile)
	t.ExecConfig.DetectCmdFormat.Output.Path = filepath.Join(detectContainerOutputDir, detectOutputFile)
	services, err = t.executeDetect(t.ExecConfig.DetectCmdFormat.Cmd)
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
	if t.ExecConfig.TransformCmdFormat.Cmd == nil {
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
	t.ExecConfig.TransformCmdFormat.Input.Path = filepath.Join(containerInputDir, transformInputFile)
	t.ExecConfig.TransformCmdFormat.Output.Path = filepath.Join(transformContainerOutputDir, transformOutputFile)
	kvMap := map[string][]string{}
	kvMap["input"] = []string{"input" + envDelimiter + t.ExecConfig.TransformCmdFormat.Input.Path}
	kvMap["output"] = []string{"output" + envDelimiter + t.ExecConfig.TransformCmdFormat.Output.Path}
	kvMap["inputEnv"] = []string{
		t.ExecConfig.TransformCmdFormat.Input.EnvName +
			envDelimiter +
			t.ExecConfig.TransformCmdFormat.Input.Path,
	}
	kvMap["outputEnv"] = []string{
		t.ExecConfig.TransformCmdFormat.Output.EnvName +
			envDelimiter +
			t.ExecConfig.TransformCmdFormat.Output.Path,
	}
	cmdToRun, envList := t.configIO(
		t.ExecConfig.TransformCmdFormat,
		kvMap,
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

func (t *Executable) executeDetect(cmd environmenttypes.Command) (services map[string][]transformertypes.Artifact, err error) {
	kvMap := map[string][]string{}
	kvMap["input"] = []string{"input" + envDelimiter + t.ExecConfig.DetectCmdFormat.Input.Path}
	kvMap["output"] = []string{"output" + envDelimiter + t.ExecConfig.DetectCmdFormat.Output.Path}
	kvMap["inputEnv"] = []string{
		t.ExecConfig.DetectCmdFormat.Input.EnvName +
			envDelimiter +
			t.ExecConfig.DetectCmdFormat.Input.Path,
	}
	kvMap["outputEnv"] = []string{
		t.ExecConfig.DetectCmdFormat.Output.EnvName +
			envDelimiter +
			t.ExecConfig.DetectCmdFormat.Output.Path,
	}
	cmdToRun, envList := t.configIO(
		t.ExecConfig.DetectCmdFormat,
		kvMap,
	)
	stdout, stderr, exitcode, err := t.Env.Exec(cmdToRun, envList)
	if err != nil {
		return nil, fmt.Errorf("failed to execute the command in the environment.\nstdout: %s\nstderr: %s\nexit code: %d\nError: %w", stdout, stderr, exitcode, err)
	} else if exitcode != 0 {
		return nil, fmt.Errorf("the detect command failed with a non-zero exit code.\nstdout: %s\nstderr: %s\nexit code: %d", stdout, stderr, exitcode)
	}
	logrus.Debugf("%s Detect succeeded in %s : %s, %s, %d",
		t.Config.Name, t.ExecConfig.DetectCmdFormat.Input.Path, stdout, stderr, exitcode)
	outputPath, err := t.Env.Env.Download(detectContainerOutputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to download the json output at path '%s' from the environment. Error: %w", outputPath, err)
	}
	logrus.Debugf("Output detect JSON path: %v", outputPath)
	output := map[string][]transformertypes.Artifact{}
	jsonOutputPath := filepath.Join(outputPath, detectOutputFile)
	if err := common.ReadJSON(jsonOutputPath, &output); err != nil {
		logrus.Warnf("failed in unmarshal the detect output file at path '%s' as json. Trying with config type. Error: %q", jsonOutputPath, err)
		config := map[string]interface{}{}
		if err := common.ReadJSON(jsonOutputPath, &config); err != nil {
			logrus.Warnf("failed in unmarshal the detect output file at path '%s' as config json. Error: %q", jsonOutputPath, err)
		}
		trans := transformertypes.Artifact{
			Paths: map[transformertypes.PathType][]string{artifacts.ServiceDirPathType: {t.ExecConfig.DetectCmdFormat.Input.Path}},
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
	cmdFormat CommandFormat,
	kvMap map[string][]string) (
	environmenttypes.Command,
	[]string,
) {
	cmdToRun := cmdFormat.Cmd
	envList := []string{}
	if cmdFormat.Input.Mode == argMode {
		cmdToRun = append(cmdToRun, kvMap["input"]...)
	} else if cmdFormat.Input.Mode == envMode {
		envList = append(envList, kvMap["inputEnv"]...)
	}
	if cmdFormat.Output.Mode == argMode {
		cmdToRun = append(cmdToRun, kvMap["output"]...)
	} else if cmdFormat.Output.Mode == envMode {
		envList = append(envList, kvMap["outputEnv"]...)
	}
	return cmdToRun, envList
}
