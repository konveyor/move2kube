/*
 *  Copyright IBM Corporation 2024
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
	"fmt"
	"path/filepath"
	"unsafe"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/environment"
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/sirupsen/logrus"
)

// WasmTransformer implements transformer interface and is used to write simple external transformers
type WasmTransformer struct {
	Config     transformertypes.Transformer
	Env        *environment.Environment
	WasmConfig *WasmYamlConfig
	ModuleId   int32
}

// WasmYamlConfig defines yaml config for WasmTransformer transformers
type WasmYamlConfig struct {
	WasmFile string `yaml:"wasmFile"`
}
type DirDetectInput struct {
	SourceDir string `yaml:"sourceDir" json:"sourceDir"`
}

type DirDetectOutput struct {
	Services map[string][]transformertypes.Artifact `yaml:"services" json:"services"`
}

type TransformInput struct {
	NewArtifacts         []transformertypes.Artifact `yaml:"newArtifacts,omitempty" json:"newArtifacts,omitempty"`
	AlreadySeenArtifacts []transformertypes.Artifact `yaml:"alreadySeenArtifacts,omitempty" json:"alreadySeenArtifacts,omitempty"`
}

type TransformOutput struct {
	NewPathMappings []transformertypes.PathMapping `yaml:"newPathMappings,omitempty" json:"newPathMappings,omitempty"`
	NewArtifacts    []transformertypes.Artifact    `yaml:"newArtifacts,omitempty" json:"newArtifacts,omitempty"`
}

const (
	// maxOutputLength TODO: this is hardcoded since we can't export myAllocate yet
	maxOutputLength uint32 = 8192
)

//go:wasmimport mym2kmodule load_wasm_module
func load_wasm_module(ptr unsafe.Pointer, len uint32) int32

//go:wasmimport mym2kmodule run_dir_detect
func run_dir_detect(
	moduleId int32,
	ptr unsafe.Pointer,
	len uint32,
	outPtr unsafe.Pointer,
) int32

//go:wasmimport mym2kmodule run_transform
func run_transform(
	moduleId int32,
	ptr unsafe.Pointer,
	len uint32,
	outPtr unsafe.Pointer,
) int32

// https://github.com/tinygo-org/tinygo/issues/411#issuecomment-503066868
var keyToAllocatedBytes = map[uint32][]byte{}
var nextKey uint32 = 41

func myAllocate(size uint32) *byte {
	nextKey += 1
	newArr := make([]byte, size)
	keyToAllocatedBytes[nextKey] = newArr
	return &newArr[0]
}

func toPtr(path []byte) unsafe.Pointer {
	return unsafe.Pointer(&path[0])
}

func loadWasmModule(path string) (int32, error) {
	result := load_wasm_module(toPtr([]byte(path)), uint32(len(path)))
	if result < 0 {
		return -1, fmt.Errorf("failed to load the wasm module, got module id: %d", result)
	}
	return result, nil
}

// Init Initializes the transformer
func (t *WasmTransformer) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env
	t.WasmConfig = &WasmYamlConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.WasmConfig); err != nil {
		return fmt.Errorf("failed to load config for Transformer %+v into %T . Error: %w", t.Config.Spec.Config, t.WasmConfig, err)
	}
	// logrus.Infof("DEBUG t.WasmConfig %+v", t.WasmConfig)
	// logrus.Infof("DEBUG t.Env %+v", t.Env)
	if t.WasmConfig.WasmFile != "" {
		wasmFilePath := filepath.Join(t.Env.GetEnvironmentContext(), t.WasmConfig.WasmFile)
		// contents, err := os.ReadFile(wasmFilePath)
		// if err != nil {
		// 	return fmt.Errorf("failed to read the wasm file. Error: %w", err)
		// }
		// logrus.Infof("wasm file contents size: %d", len(contents))
		moduleId, err := loadWasmModule(wasmFilePath)
		if err != nil {
			return fmt.Errorf("failed to load the wasm module from path '%s' . error: %w", wasmFilePath, err)
		}
		// logrus.Infof("DEBUG wasm file moduleId: %d", moduleId)
		t.ModuleId = moduleId
	}
	return nil
}

// GetConfig returns the transformer config
func (t *WasmTransformer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *WasmTransformer) DirectoryDetect(dir string) (map[string][]transformertypes.Artifact, error) {
	logrus.Info("WasmTransformer.DirectoryDetect start")
	defer logrus.Info("WasmTransformer.DirectoryDetect end")
	input := DirDetectInput{
		SourceDir: dir,
	}
	inputJson, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the input as json. Error: %w", err)
	}
	ptr := myAllocate(maxOutputLength)
	len := run_dir_detect(
		t.ModuleId,
		toPtr(inputJson),
		uint32(len(inputJson)),
		unsafe.Pointer(ptr),
	)
	if len < 0 {
		return nil, fmt.Errorf("failed to run the transform using the custom transformer module")
	}
	outputBytes := unsafe.Slice(ptr, len)
	output := DirDetectOutput{}
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the transform output as json. Error: %w", err)
	}
	return output.Services, nil
}

// Transform transforms the artifacts
func (t *WasmTransformer) Transform(
	newArtifacts []transformertypes.Artifact,
	alreadySeenArtifacts []transformertypes.Artifact,
) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	logrus.Info("WasmTransformer.Transform start")
	defer logrus.Info("WasmTransformer.Transform end")
	input := TransformInput{
		NewArtifacts:         newArtifacts,
		AlreadySeenArtifacts: alreadySeenArtifacts,
	}
	inputJson, err := json.Marshal(input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal the input as json. Error: %w", err)
	}
	ptr := myAllocate(maxOutputLength)
	len := run_transform(
		t.ModuleId,
		toPtr(inputJson),
		uint32(len(inputJson)),
		unsafe.Pointer(ptr),
	)
	if len < 0 {
		return nil, nil, fmt.Errorf("failed to run the transform using the custom transformer module")
	}
	outputBytes := unsafe.Slice(ptr, len)
	output := TransformOutput{}
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal the transform output as json. Error: %w", err)
	}
	return output.NewPathMappings, output.NewArtifacts, nil
}
