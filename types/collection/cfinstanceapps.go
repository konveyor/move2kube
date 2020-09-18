/*
Copyright IBM Corporation 2020

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

package collection

import (
	"github.com/konveyor/move2kube/types"
)

// CfInstanceAppsMetadataKind defines kind of cf runtime instance apps file
const CfInstanceAppsMetadataKind types.Kind = "CfInstanceApps"

// CfInstanceApps defines definition of cf runtime instance apps file
type CfInstanceApps struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             CfInstanceAppsSpec `yaml:"spec,omitempty"`
}

// CfInstanceAppsSpec stores the data
type CfInstanceAppsSpec struct {
	CfApplications []CfApplication `yaml:"applications"`
}

// CfApplication defines the structure of a cf runtime application
type CfApplication struct {
	Name              string            `yaml:"name"`
	Buildpack         string            `yaml:"buildpack,omitempty"`
	DetectedBuildpack string            `yaml:"detectedBuildpack,omitempty"`
	Memory            int64             `yaml:"memory"`
	Instances         int               `yaml:"instances"`
	DockerImage       string            `yaml:"dockerImage,omitempty"`
	Ports             []int32           `yaml:"ports"`
	Env               map[string]string `yaml:"env,omitempty"`
}

// NewCfInstanceApps creates a new instance of CfInstanceApps
func NewCfInstanceApps() CfInstanceApps {
	var cfInstanceApps = CfInstanceApps{
		TypeMeta: types.TypeMeta{
			Kind:       string(CfInstanceAppsMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
	return cfInstanceApps
}
