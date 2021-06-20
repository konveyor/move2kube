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
	"github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

const (
	ServiceArtifactType               plantypes.ArtifactType = "Service"
	ContainerBuildArtifactType        plantypes.ArtifactType = "ContainerBuild"
	IRArtifactType                    plantypes.ArtifactType = "IR"
	ContainerBuildSupportArtifactType plantypes.ArtifactType = "ContainerBuildSupport"
	CNBMetadataArtifactType           plantypes.ArtifactType = "CNBMetadata"
)

const (
	ServiceConfigType  plantypes.ConfigType = "Service"
	PlanConfigType     plantypes.ConfigType = "Plan"
	TemplateConfigType plantypes.ConfigType = "Template"
	IRConfigType       plantypes.ConfigType = "IR"
)

type ServiceConfig struct {
	ServiceName string `yaml:"serviceName"`
}

type PlanConfig struct {
	PlanName      string                     `yaml:"planName"`
	TargetCluster collection.ClusterMetadata `yaml:"cluster"`
}
