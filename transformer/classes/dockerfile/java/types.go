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

package java

import (
	"github.com/konveyor/move2kube/types"
)

// JavaPackageNamesMappingKind defines kind of JavaPackageNamesMappingKind
const JavaPackageNamesMappingKind types.Kind = "JavaPackageVersions"

// JavaPackageNamesMapping stores the java package version mappings
type JavaPackageNamesMapping struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             JavaPackageNamesMappingSpec `yaml:"spec,omitempty"`
}

// JavaPackageNamesMappingSpec stores the java package version spec
type JavaPackageNamesMappingSpec struct {
	PackageVersions map[string]string `yaml:"packageVersions"`
}

type packaging = string

const (
	// JarPackaging defines jar packaging
	JarPackaging packaging = "jar"
	// WarPackaging defines war packaging
	WarPackaging packaging = "war"
	// EarPackaging defines ear packaging
	EarPackaging packaging = "ear"
)
