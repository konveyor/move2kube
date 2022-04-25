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
	JarConfigType transformertypes.ConfigType = transformertypes.ConfigType(JarArtifactType)
	// JarPathType defines jar path type
	JarPathType transformertypes.PathType = transformertypes.PathType(JarArtifactType)

	// WarArtifactType defines war artifact type
	WarArtifactType transformertypes.ArtifactType = "War"
	// WarConfigType defines the war config type
	WarConfigType transformertypes.ConfigType = transformertypes.ConfigType(WarArtifactType)
	// WarPathType defines the war path type
	WarPathType transformertypes.PathType = transformertypes.PathType(WarArtifactType)

	// EarArtifactType defines the ear artifact type
	EarArtifactType transformertypes.ArtifactType = "Ear"
	// EarConfigType defines the ear config type
	EarConfigType transformertypes.ConfigType = transformertypes.ConfigType(EarArtifactType)
	// EarPathType defines the ear path type
	EarPathType transformertypes.PathType = transformertypes.PathType(EarArtifactType)

	// BuildContainerFileType defines the build container file type
	BuildContainerFileType transformertypes.PathType = "BuildContainerFile"

	// PomPackaging is used by parent pom.xml files in multi-module maven projects.
	// https://maven.apache.org/pom.html#Aggregation
	// https://www.baeldung.com/maven-multi-module
	PomPackaging JavaPackaging = "pom"
	// JarPackaging defines jar packaging
	JarPackaging JavaPackaging = "jar"
	// WarPackaging defines war packaging
	WarPackaging JavaPackaging = "war"
	// EarPackaging defines ear packaging
	EarPackaging JavaPackaging = "ear"
)

// JavaPackaging represents JavaPackaging type
type JavaPackaging string

// JarArtifactConfig defines a JarArtifactConfig struct
type JarArtifactConfig struct {
	Port               int32             `yaml:"port"`
	JavaVersion        string            `yaml:"javaVersion"`
	BuildContainerName string            `yaml:"buildContainerName"`
	DeploymentFilePath string            `yaml:"deploymentFilePath"`
	EnvVariables       map[string]string `yaml:"envVariables"`
}

// WarArtifactConfig defines a WarArtifactConfig struct
type WarArtifactConfig JarArtifactConfig

// EarArtifactConfig defines a EarArtifactConfig struct
type EarArtifactConfig JarArtifactConfig
