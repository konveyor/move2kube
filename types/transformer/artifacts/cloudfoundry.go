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
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

const (
	// CfManifestPathType defines the source artifact type of cf manifest
	CfManifestPathType transformertypes.PathType = "CfManifest"
	// CfRunningManifestPathType defines the source artifact type of a manifest of a running instance
	CfRunningManifestPathType transformertypes.PathType = "CfRunningManifest"
)

const (
	// CloudFoundryConfigType represents the cloud foundry service configuration
	CloudFoundryConfigType transformertypes.ConfigType = "CloudFoundryService"
)

// CloudFoundryConfig stores the configurations
type CloudFoundryConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
	ImageName   string `yaml:"imageName,omitempty"`
}
