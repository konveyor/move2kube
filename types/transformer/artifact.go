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

package transformer

import (
	"fmt"

	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

// ArtifactType is used to store the artifact type
type ArtifactType = string

// ConfigType is used to store the config type
type ConfigType = string

// PathType is used to store the path type
type PathType = string

// PathContext is used to Store Path Context prefix
type PathContext = string

const (
	// Output refers to the Output path context
	Output PathContext = "output:"
	// Source refers to the source path context
	Source PathContext = "source:"
)

// Artifact represents the artifact that can be passed between transformers
type Artifact struct {
	Name     string       `yaml:"name,omitempty" json:"name,omitempty"`
	Artifact ArtifactType `yaml:"artifact,omitempty" json:"artifact,omitempty"`

	Paths   map[PathType][]string      `yaml:"paths,omitempty" json:"paths,omitempty" m2kpath:"normal"`
	Configs map[ConfigType]interface{} `yaml:"configs,omitempty" json:"config,omitempty"` // Could be IR or template config or any custom configuration
}

// GetConfig returns the config that has a particular config name
func (a *Artifact) GetConfig(configName ConfigType, obj interface{}) (err error) {
	cConfig, ok := a.Configs[configName]
	if !ok {
		return fmt.Errorf("unable to find %s config in artifact %+v. Ignoring", configName, a)
	}
	err = common.GetObjFromInterface(cConfig, obj)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", cConfig, obj, err)
		return err
	}
	return nil
}
