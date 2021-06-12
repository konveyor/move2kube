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

package classes

/*
import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerexec"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
)

var (
	cnbwarnlongwait = false
)

// CNBContainerizer implements Containerizer interface
type CNBContainerizer struct {
	builders []string
}

const (
	orderLabel string = "io.buildpacks.buildpack.order"
)

type order []orderEntry

type orderEntry struct {
	Group []buildpackRef `toml:"group" json:"group"`
}

type buildpackRef struct {
	buildpackInfo
	Optional bool `toml:"optional,omitempty" json:"optional,omitempty"`
}

type buildpackInfo struct {
	ID       string `toml:"id" json:"id,omitempty"`
	Version  string `toml:"version" json:"version,omitempty"`
	Homepage string `toml:"homepage,omitempty" json:"homepage,omitempty"`
}

// Cache
var cnbcache = map[string][]string{}

// Init initializes the containerizer
func (d *CNBContainerizer) Init(path string) {
	d.builders = []string{"cloudfoundry/cnb:cflinuxfs3", "gcr.io/buildpacks/builder"}
	//TODO: Load from CNB Builder name collector
}

func logCNBLongWait() {
	if !cnbwarnlongwait {
		logrus.Warn("This could take a few minutes to complete.")
		cnbwarnlongwait = true
	}
}

// GetTargetOptions gets all possible target options for a path
func (d *CNBContainerizer) GetTargetOptions(plan plantypes.Plan, path string) []string {
	if options, ok := cnbcache[path]; ok {
		return options
	}
	if containerexec.GetEngine() == nil {
		logrus.Warnf("No container execution method valid. Not using CNB.")
		return nil
	}
	supportedbuilders := []string{}
	for _, builder := range d.builders {
		if d.isBuilderSupported(path, string(builder)) {
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
	if len(service.ContainerizationOptions) == 0 {
		err := fmt.Errorf("No Containerization target options found")
		logrus.Errorf("%s", err)
		return container, err
	}
	if service.ContainerizationOptions[0].ContainerBuildType != d.GetContainerBuildStrategy() {
		return container, fmt.Errorf("Service %s has container build type %s . Expected %s", service.ServiceName, service.ContainerizationOptions[0].ContainerBuildType, d.GetContainerBuildStrategy())
	}
	if len(service.ContainerizationOptions) == 0 {
		return container, fmt.Errorf("Service %s has no containerization target options", service.ServiceName)
	}
	builder := service.ContainerizationOptions[0].ID
	cnbbuilderstring, err := common.GetStringFromTemplate(scripts.CNBBuilder_sh, struct {
		ImageName string
		Builder   string
	}{
		ImageName: service.Image,
		Builder:   builder,
	})
	if err != nil {
		logrus.Warnf("Unable to translate template %s to string. Error: %q", scripts.CNBBuilder_sh, err)
		return container, err
	}
	if len(service.SourceArtifacts[plantypes.SourceDirectoryArtifactType]) == 0 {
		err := fmt.Errorf("service %s has no source code directory specified", service.ServiceName)
		return container, err
	}
	sourceCodeDir := service.SourceArtifacts[plantypes.SourceDirectoryArtifactType][0]
	relOutputPath, err := filepath.Rel(plan.Spec.Inputs.RootDir, sourceCodeDir)
	if err != nil {
		logrus.Errorf("Failed to make the source code directory %q relative to the root directory %q Error: %q", sourceCodeDir, plan.Spec.Inputs.RootDir, err)
		return container, err
	}
	container.AddFile(filepath.Join(relOutputPath, service.ServiceName+"-cnb-build.sh"), cnbbuilderstring)
	container.AddExposedPort(common.DefaultServicePort)
	return container, nil
}

// GetAllBuildpacks returns all supported buildpacks
func (d *CNBContainerizer) GetAllBuildpacks() (buildpacks map[string][]string) {
	buildpacks = map[string][]string{}
	e := containerexec.GetEngine()
	if e == nil {
		logrus.Errorf("No container runtime found")
		return buildpacks
	}
	logrus.Debugf("Getting data of all builders %s", d.builders)
	for _, builder := range d.builders {
		inspectOutput, err := e.InspectImage(builder)
		logrus.Debugf("Inspecting image %s", builder)
		if err != nil {
			logrus.Debugf("Unable to inspect image %s : %s, %+v", builder, err, inspectOutput)
			continue
		}
		buildpacks[builder] = d.getBuildersFromLabel(inspectOutput.Config.Labels[orderLabel])
	}
	return buildpacks
}

func (d *CNBContainerizer) getBuildersFromLabel(label string) (buildpacks []string) {
	buildpacks = []string{}
	ogs := order{}
	err := json.Unmarshal([]byte(label), &ogs)
	if err != nil {
		logrus.Warnf("Unable to read order : %s", err)
		return
	}
	logrus.Debugf("Builder data :%s", label)
	for _, og := range ogs {
		for _, buildpackref := range og.Group {
			buildpacks = append(buildpacks, buildpackref.ID)
		}
	}
	return
}

func (d *CNBContainerizer) isBuilderSupported(path string, builder string) bool {
	e := containerexec.GetEngine()
	if e == nil {
		logrus.Errorf("No container runtime available")
		return false
	}
	logCNBLongWait()
	p, err := filepath.Abs(path)
	if err != nil {
		logrus.Warnf("Unable to resolve to absolute path : %s", err)
	}
	logrus.Debugf("Running detect on image %s", builder)
	output, _, err := e.RunContainer(builder, "/cnb/lifecycle/detector", p, "/workspace")
	if err != nil {
		logrus.Debugf("Detect failed %s : %s : %s", builder, err, output)
		return false
	}
	logrus.Debug(output)
	return true
}
*/
