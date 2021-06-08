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

package translator

import (
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

type Patch struct {
	ServiceName string               `json:"serviceName,omitempty"`
	Translator  plantypes.Translator `json:"translator"`

	SourceDir     string `json:"sourceDir,omitempty"`     // Default is /source
	UseSourceDiff bool   `json:"useSourceDiff,omitempty"` // Default is true

	// Paths in machine/image - can use "template:" in source to apply template
	// If relative path is used after "template:", then the source directory is relateive to template directory
	// Destination directory is relative to output directory
	PathMappings []PathMapping `json:"pathMappings"`
	Config       interface{}   `json:"config"`

	IR irtypes.IR `json:"ir"`
}

type PathMapping struct {
	SrcPath  string `yaml:"sourcePath"`
	DestPath string `yaml:"destinationPath"`
}
