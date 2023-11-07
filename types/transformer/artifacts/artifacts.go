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
)

// OriginalNameConfig stores the original name of the app
type OriginalNameConfig struct {
	OriginalName string `yaml:"originalName,omitempty" json:"originalName,omitempty"`
}

const (
	// ServiceDirPathType points to the service context directory.
	ServiceDirPathType transformertypes.PathType = "ServiceDirectories"
	// ServiceRootDirPathType points to the directory of the root project for a service.
	ServiceRootDirPathType transformertypes.PathType = "ServiceRootDirectory"
	// OriginalNameConfigType stores the original name of the app
	OriginalNameConfigType transformertypes.ConfigType = "OriginalName"
)
