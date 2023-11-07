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

package artifacts

import (
	transformertypes "github.com/konveyor/move2kube-wasm/types/transformer"
	"github.com/sirupsen/logrus"
)

// GradleConfig stores gradle related configuration information
type GradleConfig struct {
	RootProjectName  string              `yaml:"rootProjectName,omitempty" json:"rootProjectName,omitempty"`
	PackagingType    JavaPackaging       `yaml:"packagingType" json:"packagingType"`
	IsGradlewPresent bool                `yaml:"isGradlewPresent" json:"isGradlewPresent"`
	ChildModules     []GradleChildModule `yaml:"childModules,omitempty" json:"childModules,omitempty"`
}

// GradleChildModule represents the data for a Gradle child project in a multi-module project
type GradleChildModule struct {
	// Name is the name/artifact id of the child module
	Name string `yaml:"name" json:"name"`
	// RelBuildScriptPath is the path to the child build.gradle (relative to the root settings.gradle)
	RelBuildScriptPath string `yaml:"buildScriptPath" json:"buildScriptPath"`
}

const (
	// GradleConfigType stores the gradle config
	GradleConfigType transformertypes.ConfigType = "Gradle"
	// GradleSettingsFilePathType is the type for paths to settings.gradle files
	GradleSettingsFilePathType transformertypes.PathType = "GradleSettingsFile"
	// GradleBuildFilePathType stores the Gradle Build File file Path
	GradleBuildFilePathType transformertypes.PathType = "GradleBuildFile"
)

// Merge implements the Config interface allowing artifacts to be merged
func (gc *GradleConfig) Merge(newgcobj interface{}) bool {
	newgcptr, ok := newgcobj.(*GradleConfig)
	if !ok {
		newgc, ok := newgcobj.(GradleConfig)
		if !ok {
			logrus.Error("Unable to cast to GradleConfig for merge")
			return false
		}
		newgcptr = &newgc
	}
	if gc.PackagingType != newgcptr.PackagingType {
		return false
	}
	return true
}
