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

package containerizer

import (
	"errors"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer/cnb"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// CNBContainerizer implements Containerizer interface
type CNBContainerizer struct {
	builders []string
}

// Cache
var cnbcache = map[string][]string{}

// Init initializes the containerizer
func (d *CNBContainerizer) Init(path string) {
	d.builders = []string{"cloudfoundry/cnb:cflinuxfs3", "gcr.io/buildpacks/builder"}
	//initRunc(d.builders)
	//TODO: Load from CNB Builder name collector
}

// GetTargetOptions gets all possible target options for a path
func (d *CNBContainerizer) GetTargetOptions(plan plantypes.Plan, path string) []string {
	if options, ok := cnbcache[path]; ok {
		return options
	}

	builders := d.builders
	supportedbuilders := []string{}

	for _, builder := range builders {
		if cnb.IsBuilderSupported(path, string(builder)) {
			supportedbuilders = append(supportedbuilders, builder)
		}
	}
	cnbcache[path] = supportedbuilders
	return supportedbuilders
}

// GetContainerBuildStrategy returns the containerization build strategy for the containerizer
func (d *CNBContainerizer) GetContainerBuildStrategy() plantypes.ContainerBuildTypeValue {
	return plantypes.CNBContainerBuildTypeValue
}

// GetContainer returns the container for the service
func (d *CNBContainerizer) GetContainer(plan plantypes.Plan, service plantypes.Service) (irtypes.Container, error) {
	// TODO: Fix exposed ports too
	container := irtypes.NewContainer(d.GetContainerBuildStrategy(), service.Image, true)
	if service.ContainerBuildType == d.GetContainerBuildStrategy() && len(service.ContainerizationTargetOptions) > 0 {
		builder := service.ContainerizationTargetOptions[0]
		cnbbuilderstring, err := common.GetStringFromTemplate(scripts.CNBBuilder_sh, struct {
			ImageName string
			Builder   string
		}{
			ImageName: service.Image,
			Builder:   builder,
		})
		if err != nil {
			log.Warnf("Unable to translate template to string : %s", scripts.CNBBuilder_sh)
		} else {
			container.AddFile(filepath.Join(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0], service.ServiceName+"cnbbuilder.sh"), cnbbuilderstring)
		}
		container.ExposedPorts = []int{8080}
		return container, nil
	}
	return container, errors.New("Unsupported service type for Containerization or insufficient information in service")
}

// GetAllBuildpacks returns all supported buildpacks
func (d *CNBContainerizer) GetAllBuildpacks() map[string][]string {
	return cnb.GetAllBuildpacks(d.builders)
}
