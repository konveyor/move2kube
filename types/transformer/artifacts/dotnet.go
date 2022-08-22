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

package artifacts

import (
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

// DotNetConfig stores dot net related configuration information
type DotNetConfig struct {
	IsDotNetCore          bool                 `yaml:"isDotNetCore" json:"isDotNetCore"`
	DotNetAppName         string               `yaml:"dotNetAppName,omitempty" json:"dotNetAppName,omitempty"`
	IsSolutionFilePresent bool                 `yaml:"isSolutionFilePresent" json:"isSolutionFilePresent"`
	ChildProjects         []DotNetChildProject `yaml:"childProjects,omitempty" json:"childProjects,omitempty"`
}

// DotNetChildProject represents the data for a child project in a multi-project dot net app
type DotNetChildProject struct {
	// Name is the name of the child project
	Name string `yaml:"name" json:"name"`
	// OriginalName is the name of the child project before normalization
	OriginalName string `yaml:"originalName" json:"originalName"`
	// RelCSProjPath is the path to the child .csproj (relative to the parent .sln file)
	RelCSProjPath string `yaml:"csProjPath" json:"csProjPath"`
	// TargetFramework contains the target dot net core or dot net framework name and version
	TargetFramework string `yaml:"targetFramework" json:"targetFramework"`
}

const (
	// DotNetConfigType stores the dot net config
	DotNetConfigType transformertypes.ConfigType = "DotNet"
)
