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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/second-state/WasmEdge-go/wasmedge"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

const (
	wasmEnvDelimiter              = "="
	detectInputPathWASMEnvKey     = "M2K_DETECT_INPUT_PATH"
	detectOutputPathWASMEnvKey    = "M2K_DETECT_OUTPUT_PATH"
	transformInputPathWASMEnvKey  = "M2K_TRANSFORM_INPUT_PATH"
	transformOutputPathWASMEnvKey = "M2K_TRANSFORM_OUTPUT_PATH"
)

// WASM implements wasm transformer interface and is used for wasm based transformers
type WASM struct {
	Config     transformertypes.Transformer
	Env        *environment.Environment
	WASMConfig *WASMYamlConfig
}

// WASMYamlConfig is the format of wasm transformer yaml config
type WASMYamlConfig struct {
	WASMModule string        `yaml:"wasm_module"`
	EnvList    []core.EnvVar `yaml:"env,omitempty"`
}

// Init Initializes the transformer
func (t *WASM) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.WASMConfig = &WASMYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.WASMConfig); err != nil {
		return fmt.Errorf("unable to load config for Transformer %+v into %T . Error: %w", t.Config.Spec.Config, t.WASMConfig, err)
	}

	return nil
}

func (t *WASM) prepareEnv() []string {
	envList := []string{}
	for _, env := range t.WASMConfig.EnvList {
		envList = append(envList, env.Name+wasmEnvDelimiter+env.Value)
	}
	return envList
}

// GetConfig returns the transformer config
func (t *WASM) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *WASM) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	vm, err := t.initVm([]string{dir + ":" + dir})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize WASM VM: %w", err)
	}

	allocateResult, err := vm.Execute("malloc", int32(len(dir)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to alloc memory for directory: %w", err)
	}
	dirPointer := allocateResult[0].(int32)
	mod := vm.GetActiveModule()
	mem := mod.FindMemory("memory")
	memData, err := mem.GetData(uint(dirPointer), uint(len(dir)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to load wasm memory region: %w", err)
	}
	copy(memData, dir)
	memData[len(dir)] = 0
	directoryDetectOutput, dderr := vm.Execute("directoryDetect", dirPointer)
	if dderr != nil {
		err = fmt.Errorf("failed to execute directoryDetect in the wasm module. Error : %w", dderr)
		return nil, err
	}
	directoryDetectOutputPointer := directoryDetectOutput[0].(int32)
	memData, err = mem.GetData(uint(directoryDetectOutputPointer), 8)
	if err != nil {
		return nil, fmt.Errorf("failed to load directoryDetect output: %w", err)
	}
	resultPointer := binary.LittleEndian.Uint32(memData[:4])
	resultLength := binary.LittleEndian.Uint32(memData[4:])
	memData, err = mem.GetData(uint(resultPointer), uint(resultLength))
	if err != nil {
		return nil, fmt.Errorf("failed to read directoryDetect output: %w", err)
	}

	services := map[string][]transformertypes.Artifact{}
	err = json.Unmarshal(memData, &services)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal directoryDetect output: %w", err)
	}
	return services, err
}

// Transform transforms the artifacts
func (t *WASM) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	data := make(map[string]interface{})
	data["newArtifacts"] = newArtifacts
	data["oldArtifacts"] = alreadySeenArtifacts
	dataByt, err := json.Marshal(data)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal transform input: %w", err)
	}
	dataStr := string(dataByt)

	preopens := []string{}
	for _, artifact := range newArtifacts {
		for _, paths := range artifact.Paths {
			for _, path := range paths {
				preopens = append(preopens, path)
			}
		}
	}

	sort.Slice(preopens, func(i, j int) bool {
		l1, l2 := len(preopens[i]), len(preopens[j])
		if l1 != l2 {
			return l1 < l2
		}
		return preopens[i] < preopens[j]
	})

	deduplicatedPreopens := []string{}
	for _, path := range preopens {
		shouldSkip := false
		for _, existingPath := range deduplicatedPreopens {
			if strings.HasPrefix(path, existingPath) {
				shouldSkip = true
				break
			}
		}

		if !shouldSkip {
			deduplicatedPreopens = append(deduplicatedPreopens, path)
		}
	}

	finalPreopens := []string{}
	for _, path := range deduplicatedPreopens {
		finalPreopens = append(finalPreopens, path+":"+path)
	}

	vm, err := t.initVm(finalPreopens)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize VM (transform): %w", err)
	}

	allocateResult, err := vm.Execute("malloc", int32(len(dataStr)+1))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to alloc memory for transform input: %w", err)
	}
	dataPointer := allocateResult[0].(int32)
	mod := vm.GetActiveModule()
	mem := mod.FindMemory("memory")
	memData, err := mem.GetData(uint(dataPointer), uint(len(dataStr)+1))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load wasm memory region: %w", err)
	}
	copy(memData, dataStr)
	memData[len(dataStr)] = 0
	transformOutput, err := vm.Execute("transform", dataPointer)
	transformOutputPointer := transformOutput[0].(int32)
	memData, err = mem.GetData(uint(transformOutputPointer), 8)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load transform output: %w", err)
	}
	resultPointer := binary.LittleEndian.Uint32(memData[:4])
	resultLength := binary.LittleEndian.Uint32(memData[4:])
	memData, err = mem.GetData(uint(resultPointer), uint(resultLength))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read transform output: %w", err)
	}
	logrus.Debug(string(memData))
	var output transformertypes.TransformOutput
	err = json.Unmarshal(memData, &output)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal transformer output: %w", err)
	}
	pathMappings = append(pathMappings, output.PathMappings...)
	createdArtifacts = append(createdArtifacts, output.CreatedArtifacts...)
	return pathMappings, createdArtifacts, err
}

func (t *WASM) initVm(preopens []string) (*wasmedge.VM, error) {
	wasmedge.SetLogErrorLevel()
	conf := wasmedge.NewConfigure(wasmedge.WASI)
	vm := wasmedge.NewVMWithConfig(conf)

	wasi := vm.GetImportModule(wasmedge.WASI)
	wasi.InitWasi(
		[]string{},
		t.prepareEnv(),
		preopens,
	)

	err := vm.LoadWasmFile(filepath.Join(t.Env.GetEnvironmentContext(), t.WASMConfig.WASMModule))
	if err != nil {
		return nil, fmt.Errorf("failed to load wasm module %s: %w", t.WASMConfig.WASMModule, err)
	}
	err = vm.Validate()
	if err != nil {
		return nil, fmt.Errorf("failed to validate VM: %w", err)
	}
	err = vm.Instantiate()
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate VM: %w", err)
	}
	_, err = vm.Execute("_start")
	if err != nil {
		return nil, fmt.Errorf("failed to execute _start: %w", err)
	}

	return vm, nil
}
