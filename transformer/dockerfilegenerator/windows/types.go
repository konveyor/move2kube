/*
 *  Copyright IBM Corporation 2022
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

package windows

import (
	"github.com/konveyor/move2kube-wasm/types"
)

const (
	versionMappingFilePath = "mappings/dotnetwindowsversionmapping.yaml"
	// DotNetWindowsVersionMappingKind defines the K8s kind for the version mapping file
	DotNetWindowsVersionMappingKind types.Kind = "DotNetWindowsVersionMapping"
)

// DotNetWindowsVersionMapping stores the dot net version mapping
type DotNetWindowsVersionMapping struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             DotNetWindowsVersionMappingSpec `yaml:"spec,omitempty"`
}

// DotNetWindowsVersionMappingSpec stores the dot net version mapping spec
type DotNetWindowsVersionMappingSpec struct {
	// imageTagToSupportedVersions is a mapping from image tag to dot net framework versions that image supports.
	// Version compatibility table taken from https://hub.docker.com/_/microsoft-dotnet-framework-aspnet
	ImageTagToSupportedVersions map[string][]string `yaml:"imageTagToSupportedVersions"`
}
