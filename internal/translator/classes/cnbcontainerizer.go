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

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerexec"
	"github.com/konveyor/move2kube/internal/containerizer/scripts"
	"github.com/konveyor/move2kube/internal/translator/classes/compose"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

// CNBContainerizer implements Containerizer interface
type CNBContainerizer struct {
	TConfig     translatortypes.Translator
	CNBConfig   CNBContainerizerYamlConfig
	Environment environment.Environment
}

type CNBContainerizerYamlConfig struct {
	BuilderImageName string `yaml:"cnbbuilderimage"`
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

func (t *CNBContainerizer) Init(tc translatortypes.Translator, sourcePath string) (string, error) {
	t.TConfig = tc
	t.CNBConfig = CNBContainerizerYamlConfig{}
	err := common.GetObjFromInterface(t.TConfig.Spec.Config, &t.CNBConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Translator %+v into %T : %s", t.TConfig.Spec.Config, t.CNBConfig, err)
		return err
	}
	container := environment.Container{
		Image: t.CNBConfig.BuilderImageName,
	}
	nSourcePath := filepath.Join(common.TempPath, "translate", "cnb")
	paths := map[string]string{sourcePath: nSourcePath}
	env, err := environment.NewEnvironment(tc.Name, paths, container)
	if err != nil {
		logrus.Errorf("Unable to create environment : %s", err)
		return err
	}
	t.Environment = env
	return nil
}

func (t *CNBContainerizer) GetConfig() translatortypes.Translator {
	return t.TConfig
}

func (t *CNBContainerizer) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *CNBContainerizer) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {

	logrus.Debugf("Running detect on image %s", builder)
	output, _, err := e.RunContainer(builder, "/cnb/lifecycle/detector", p, "/workspace")
	if err != nil {
		logrus.Debugf("Detect failed %s : %s : %s", builder, err, output)
		return false
	}
	logrus.Debug(output)
	return nil, nil, nil
}

func (t *CNBContainerizer) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *CNBContainerizer) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *CNBContainerizer) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error) {
	var config ComposeConfig
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &config,
		TagName:  "yaml",
	})
	if err := decoder.Decode(translatorPlan.Config); err != nil {
		logrus.Errorf("unable to load config for GoInterface Translator %+v into %T : %s", translatorPlan.Config, config, err)
		return nil, err
	} else {
		logrus.Debugf("Compose Translator config is %+v", config)
	}
	ir := irtypes.NewIR(plan.Name)
	for _, path := range translatorPlan.Paths[composeFileSourceArtifactType] {
		logrus.Debugf("File %s being loaded from compose service : %s", path, config.ServiceName)
		// Try v3 first and if it fails try v1v2
		if cir, errV3 := new(compose.V3Loader).ConvertToIR(path, config.ServiceName); errV3 == nil {
			ir.Merge(cir)
			logrus.Debugf("compose v3 translator returned %d services", len(ir.Services))
		} else if cir, errV1V2 := new(compose.V1V2Loader).ConvertToIR(path, config.ServiceName); errV1V2 == nil {
			ir.Merge(cir)
			logrus.Debugf("compose v1v2 translator returned %d services", len(ir.Services))
		} else {
			logrus.Errorf("Unable to parse the docker compose file at path %s Error V3: %q Error V1V2: %q", path, errV3, errV1V2)
		}
	}
	for _, path := range translatorPlan.Paths[imageInfoSourceArtifactType] {
		imgMD := collecttypes.ImageInfo{}
		if err := common.ReadMove2KubeYaml(path, &imgMD); err != nil {
			logrus.Errorf("Failed to read image info yaml at path %s Error: %q", path, err)
			continue
		}
		for _, it := range imgMD.Spec.Tags {
			ir.AddContainer(it, newContainerFromImageInfo(imgMD))
		}
	}
	p := translatortypes.Patch{
		IR: ir,
	}
	return []translatortypes.Patch{p}, nil
}

func (t *CNBContainerizer) TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.PathMapping, error) {
	return nil, nil
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
