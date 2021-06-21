/*
Copyright IBM Corporation 2021

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package transformer

import (
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
)

type Artifact struct {
	Name     string                 `yaml:"name,omitempty" json:"name,omitempty"`
	Artifact plantypes.ArtifactType `yaml:"artifact,omitempty" json:"artifact,omitempty"`

	Paths   map[plantypes.PathType][]string      `yaml:"paths,omitempty" json:"paths,omitempty" m2kpath:"normal"`
	Configs map[plantypes.ConfigType]interface{} `yaml:"configs,omitempty" json:"config,omitempty"` // Could be IR or template config or any custom configuration
}

func (a *Artifact) GetConfig(configName plantypes.ConfigType, obj interface{}) (err error) {
	cConfig, ok := a.Configs[configName]
	if !ok {
		err = fmt.Errorf("unable to find compose config in artifact %+v. Ignoring", a)
		logrus.Errorf("%s", err)
		return err
	}
	err = common.GetObjFromInterface(cConfig, obj)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", cConfig, obj, err)
		return err
	}
	return nil
}
