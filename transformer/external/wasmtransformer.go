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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
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
	CompileAOT bool          `yaml:"compile_aot"`
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

func unpack(combinedResult uint64) (ptr uint32, size uint32) {
	// The pointer is the upper 32 bits of the combinedResult.
	ptr = uint32(combinedResult >> 32)
	// The size is the lower 32 bits of the combinedResult.
	size = uint32(combinedResult)
	return ptr, size
}

func pack(pointer uint32, size uint32) uint64 {
	return (uint64(pointer) << 32) | uint64(size)
}

// DirectoryDetect runs detect in each sub directory
func (t *WASM) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	mod, ctx, rt, err := t.initVm([][]string{{dir, dir}})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize WASM VM: %w", err)
	}
	defer rt.Close(ctx)
	directoryDetectFunc := mod.ExportedFunction("directoryDetect")
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")

	allocateResult, err := malloc.Call(ctx, uint64(len(dir)+1))
	if err != nil {
		return nil, fmt.Errorf("failed to alloc memory for directory: %w", err)
	}
	dirPointer := int32(allocateResult[0])

	defer free.Call(ctx, uint64(dirPointer))

	dirBytes := []byte(dir)
	dirPointerSize := uint32(len(dirBytes))

	if !mod.Memory().Write(uint32(dirPointer), dirBytes) {
		return nil, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d",
			dirPointer, len(dir)+1, mod.Memory().Size())
	}

	packedPointerSize := pack(uint32(dirPointer), dirPointerSize)
	directoryDetectResultPtrSize, err := directoryDetectFunc.Call(ctx, uint64(packedPointerSize))
	if err != nil {
		return nil, fmt.Errorf("failed to execute directory detect function: %w", err)
	}

	directoryDetectResultPtr, directoryDetectResultSize := unpack(directoryDetectResultPtrSize[0])

	bytes, ok := mod.Memory().Read(directoryDetectResultPtr, directoryDetectResultSize)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			directoryDetectResultPtr, directoryDetectResultSize, mod.Memory().Size())
	}
	services := map[string][]transformertypes.Artifact{}
	err = json.Unmarshal(bytes, &services)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal directoryDetect output: %w", err)
	}
	return services, nil
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
			preopens = append(preopens, paths...)
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

	finalPreopens := [][]string{}
	for _, path := range deduplicatedPreopens {
		finalPreopens = append(finalPreopens, []string{path, path})
	}

	mod, ctx, rt, err := t.initVm(finalPreopens)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize WASM VM: %w", err)
	}
	defer rt.Close(ctx)

	transformFunc := mod.ExportedFunction("transform")
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")

	allocateResult, err := malloc.Call(ctx, uint64(len(dataStr)+1))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to alloc memory for directory: %w", err)
	}
	dataPointer := allocateResult[0]
	defer free.Call(ctx, dataPointer)

	dataBytes := []byte(dataStr)
	dataPointerSize := uint32(len(dataBytes))

	if !mod.Memory().Write(uint32(dataPointer), []byte(dataStr)) {
		return nil, nil, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d",
			dataPointer, len(dataStr)+1, mod.Memory().Size())
	}

	packedPointerSize := pack(uint32(dataPointer), dataPointerSize)

	transformResultPtrSize, err := transformFunc.Call(ctx, packedPointerSize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to execute transform function: %w", err)
	}

	transformResultPtr := uint32(transformResultPtrSize[0] >> 32)
	transformResultSize := uint32(transformResultPtrSize[0])

	if transformResultPtr != 0 {
		defer func() error {
			_, err := free.Call(ctx, uint64(transformResultPtr))
			if err != nil {
				return fmt.Errorf("failed to free pointer memory: %w", err)
			}
			return nil
		}()
	}

	bytes, ok := mod.Memory().Read(uint32(transformResultPtr), uint32(transformResultSize))
	if !ok {
		return nil, nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			transformResultPtr, transformResultSize, mod.Memory().Size())
	}

	var output transformertypes.TransformOutput
	err = json.Unmarshal(bytes, &output)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal transformer output: %w", err)
	}
	pathMappings = append(pathMappings, output.PathMappings...)
	createdArtifacts = append(createdArtifacts, output.CreatedArtifacts...)
	return pathMappings, createdArtifacts, nil
}

func (t *WASM) initVm(preopens [][]string) (api.Module, context.Context, wazero.Runtime, error) {
	ctx := context.Background()
	var rt wazero.Runtime
	if t.WASMConfig.CompileAOT {
		rt = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	} else {
		rt = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	}
	fsconfig := wazero.NewFSConfig()
	for i := range preopens {
		fsconfig = wazero.NewFSConfig().WithDirMount(preopens[i][0], preopens[i][1])
	}

	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).WithStderr(os.Stderr).WithFSConfig(fsconfig)

	envVars := t.prepareEnv()
	for _, envVar := range envVars {
		keyValue := strings.SplitN(envVar, wasmEnvDelimiter, 2)
		if len(keyValue) == 2 {
			config = config.WithEnv(keyValue[0], keyValue[1])
		}
	}

	wasi_snapshot_preview1.MustInstantiate(ctx, rt)
	wasmBinary, err := os.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.WASMConfig.WASMModule))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to open wasm file: %w", err)
	}
	mod, err := rt.InstantiateWithConfig(ctx, wasmBinary, config)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to instantiate wasm binary with the given config: %w", err)
	}
	return mod, ctx, rt, nil
}
