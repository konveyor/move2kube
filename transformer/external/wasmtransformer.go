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
	"os"
	"path/filepath"

	"github.com/dchest/uniuri"
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
	VM         *wasmedge.VM
}

// WASMYamlConfig is the format of wasm transformer yaml config
type WASMYamlConfig struct {
	WASMModule string        `yaml:"wasm_module"`
	EnvList    []core.EnvVar `yaml:"env,omitempty"`
}

var (
	// WASMSharedDir is the directory where wasm transformer detect and transform output is stored
	WASMSharedDir = "/var/tmp/m2k_detect_wasm_output"
)

// Init Initializes the transformer
func (t *WASM) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.WASMConfig = &WASMYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.WASMConfig); err != nil {
		return fmt.Errorf("unable to load config for Transformer %+v into %T . Error: %q", t.Config.Spec.Config, t.WASMConfig, err)
	}
	WASMSharedDir = filepath.Join(WASMSharedDir, uniuri.NewLen(5))
	os.MkdirAll(WASMSharedDir, common.DefaultDirectoryPermission)
	// load wasm module
	wasmedge.SetLogErrorLevel()
	conf := wasmedge.NewConfigure(wasmedge.WASI)
	vm := wasmedge.NewVMWithConfig(conf)

	wasi := vm.GetImportModule(wasmedge.HostRegistration(wasmedge.WASI))
	wasi.InitWasi(
		[]string{},
		t.prepareEnv(),
		[]string{".:" + WASMSharedDir},
	)

	err = vm.LoadWasmFile(t.WASMConfig.WASMModule)
	vm.Validate()
	vm.Instantiate()
	t.VM = vm
	return err
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
	allocateResult, _ := t.VM.Execute("malloc", int32(len(dir)+1))
	dirPointer := allocateResult[0].(int32)
	mod := t.VM.GetActiveModule()
	mem := mod.FindMemory("memory")
	memData, _ := mem.GetData(uint(dirPointer), uint(len(dir)+1))
	copy(memData, dir)
	memData[len(dir)] = 0
	directoryDetectOutput, dderr := t.VM.Execute("directoryDetect", dirPointer)
	var err error
	if dderr != nil {
		err = fmt.Errorf("failed to execute directoryDetect in the wasm module. Error : %s", dderr.Error())
	}
	directoryDetectOutputPointer := directoryDetectOutput[0].(int32)
	memData, _ = mem.GetData(uint(directoryDetectOutputPointer), 8)
	resultPointer := binary.LittleEndian.Uint32(memData[:4])
	resultLength := binary.LittleEndian.Uint32(memData[4:])
	memData, _ = mem.GetData(uint(resultPointer), uint(resultLength))
	logrus.Debug(string(memData))
	services := map[string][]transformertypes.Artifact{}
	// will handles errors
	_ = json.Unmarshal(memData, &services)
	return services, err
}

// Transform transforms the artifacts
func (t *WASM) Transform(newArtifacts []transformertypes.Artifact, alreadySeenArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	data := make(map[string]interface{})
	data["newArtifacts"] = newArtifacts
	data["oldArtifacts"] = alreadySeenArtifacts
	dataByt, _ := json.Marshal(data)
	dataStr := string(dataByt)
	allocateResult, _ := t.VM.Execute("malloc", int32(len(dataStr)+1))
	dataPointer := allocateResult[0].(int32)
	mod := t.VM.GetActiveModule()
	mem := mod.FindMemory("memory")
	memData, _ := mem.GetData(uint(dataPointer), uint(len(dataStr)+1))
	copy(memData, dataStr)
	memData[len(dataStr)] = 0
	transformOutput, err := t.VM.Execute("transform", dataPointer)
	transformOutputPointer := transformOutput[0].(int32)
	memData, _ = mem.GetData(uint(transformOutputPointer), 8)
	resultPointer := binary.LittleEndian.Uint32(memData[:4])
	resultLength := binary.LittleEndian.Uint32(memData[4:])
	memData, _ = mem.GetData(uint(resultPointer), uint(resultLength))
	logrus.Debug(string(memData))
	var output transformertypes.TransformOutput
	json.Unmarshal(memData, &output)
	pathMappings = append(pathMappings, output.PathMappings...)
	createdArtifacts = append(createdArtifacts, output.CreatedArtifacts...)
	return pathMappings, createdArtifacts, err
}
