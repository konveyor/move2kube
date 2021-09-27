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

import transformertypes "github.com/konveyor/move2kube/types/transformer"

const springbootApplicationFilePath transformertypes.PathType = "SpringbootApplicationFile"

// ConfigurationFromBuildTool defines Configuration properties
type ConfigurationFromBuildTool struct {
	HasModules                    bool                    `yaml:"hasModules,omitempty"`
	IsSpringboot                  bool                    `yaml:"isSpringboot,omitempty"`
	IsTomcatProvided              bool                    `yaml:"isTomcatProvided,omitempty"`
	Packaging                     string                  `yaml:"packaging,omitempty"`
	JavaVersion                   string                  `yaml:"javaVersion,omitempty"`
	TomcatVersion                 string                  `yaml:"tomcatVersion,omitempty"`
	Name                          string                  `yaml:"name,omitempty"`
	ArtifactID                    string                  `yaml:"artifactId,omitempty"`
	Version                       string                  `yaml:"version,omitempty"`
	FileSuffix                    string                  `yaml:"fileSuffix,omitempty"`
	Profiles                      []string                `yaml:"profiles,omitempty"`
	SpringbootConfigFromBuildTool JavaConfigFromBuildTool `yaml:"springbootConfigFromBuildTool,omitempty"`
}

// SpringBootConfig defines SpringBootConfig properties
type SpringBootConfig struct {
	Profiles []string `yaml:"profiles,omitempty"`
}

type JarArtifactConfig struct {
}

type WarArtifactConfig struct {
}
