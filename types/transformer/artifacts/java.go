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
	// JarArtifactType defines the jar artifact type
	JarArtifactType transformertypes.ArtifactType = "Jar"
	// JarConfigType defines the jar config type
	JarConfigType transformertypes.ConfigType = "Jar"
	// JarPathType defines jar path type
	JarPathType transformertypes.PathType = "Jar"

	// WarArtifactType defines war artifact type
	WarArtifactType transformertypes.ArtifactType = "War"
	// WarConfigType defines the war config type
	WarConfigType transformertypes.ConfigType = "War"
	// WarPathType defines the war path type
	WarPathType transformertypes.PathType = "War"

	// EarArtifactType defines the ear artifact type
	EarArtifactType transformertypes.ArtifactType = "Ear"
	// EarConfigType defines the ear config type
	EarConfigType transformertypes.ConfigType = "Ear"
	// EarPathType defines the ear path type
	EarPathType transformertypes.PathType = "Ear"

	// BuildContainerFileType defines the build container file type
	BuildContainerFileType transformertypes.PathType = "BuildContainerFile"
)

// JarArtifactConfig defines a JarArtifactConfig struct
type JarArtifactConfig struct {
	DeploymentFile              string `yaml:"deploymentFile"`
	JavaVersion                 string `yaml:"javaVersion"`
	DeploymentFileDir           string `yaml:"deploymentFileDir"`
	IsDeploymentFileInContainer bool   `yaml:"isDeploymentFileInContainer"`
}

// WarArtifactConfig defines a WarArtifactConfig struct
type WarArtifactConfig struct {
	DeploymentFile                    string `yaml:"DeploymentFile"`
	JavaVersion                       string `yaml:"JavaVersion"`
	DeploymentFileDirInBuildContainer string `yaml:"DeploymentFileDirInBuildContainer"`
}

// EarArtifactConfig defines a EarArtifactConfig struct
type EarArtifactConfig struct {
	DeploymentFile                    string `yaml:"DeploymentFile"`
	JavaVersion                       string `yaml:"JavaVersion"`
	DeploymentFileDirInBuildContainer string `yaml:"DeploymentFileDirInBuildContainer"`
}
