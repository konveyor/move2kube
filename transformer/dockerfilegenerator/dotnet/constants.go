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

package dotnet

import (
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

const (
	// CSPROJ_FILE_EXT is the file extension for C# (C Sharp) projects.
	CSPROJ_FILE_EXT = ".csproj"
	// LaunchSettingsJSON is the name of the json containing launch configuration
	LaunchSettingsJSON = "launchSettings.json"
	// DotNetCoreCsprojFilesPathType points to the csproj files path of dotnetcore projects
	DotNetCoreCsprojFilesPathType transformertypes.PathType = "DotNetCoreCsprojPathType"
	// DotNetCoreSolutionFilePathType points to the solution file path of a dot net core project
	DotNetCoreSolutionFilePathType transformertypes.PathType = "DotNetCoreSolutionPathType"
)
