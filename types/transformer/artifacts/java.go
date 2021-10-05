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
	JarArtifactType transformertypes.ArtifactType = "Jar"
	JarConfigType   transformertypes.ConfigType   = "Jar"

	WarArtifactType transformertypes.ArtifactType = "War"
	WarConfigType   transformertypes.ConfigType   = "War"

	EarArtifactType transformertypes.ArtifactType = "Ear"
	EarConfigType   transformertypes.ConfigType   = "Ear"

	SpringBootConfigType transformertypes.ConfigType = "SpringBoot"

	BuildContainerFileType transformertypes.PathType = "BuildContainerFile"
)

type JarArtifactConfig struct {
	DeploymentFile                    string `yaml:"DeploymentFile"`
	JavaVersion                       string `yaml:"JavaVersion"`
	DeploymentFileDirInBuildContainer string `yaml:"DeploymentFileDirInBuildContainer"`
}

type WarArtifactConfig struct {
	DeploymentFile                    string `yaml:"DeploymentFile"`
	JavaVersion                       string `yaml:"JavaVersion"`
	DeploymentFileDirInBuildContainer string `yaml:"DeploymentFileDirInBuildContainer"`
}

type EarArtifactConfig struct {
	DeploymentFile                    string `yaml:"DeploymentFile"`
	JavaVersion                       string `yaml:"JavaVersion"`
	DeploymentFileDirInBuildContainer string `yaml:"DeploymentFileDirInBuildContainer"`
}

type SpringBootConfig struct {
	SpringBootVersion string `yaml:"SpringBootVersion"`
}
