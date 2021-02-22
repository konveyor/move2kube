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

package generator

import (
	"reflect"

	"github.com/konveyor/move2kube/internal/common"
	generatortypes "github.com/konveyor/move2kube/types/generator"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

var (
	generators        = []Generator{}
	generatorRegistry = map[string]reflect.Type{}
)

func init() {
	generatorClasses := []Generator{&GenericGenerator{}}
	for gc := range generatorClasses {
		generatorRegistry[reflect.TypeOf(gc).Name()] = reflect.TypeOf(gc)
	}
}

// Generator can be used to generate applications given path to the source code
type Generator interface {
	Init(gc generatortypes.Generator, path string)
	Name() string
	Mode() string
	Detect(plan plantypes.Plan, path string) []plantypes.GenerationOption
	Generate(serviceName string, option plantypes.GenerationOption, ir irtypes.IR) (irtypes.IR, error)
}

// Init initializes the generators
func Init(path string, selectGenerators bool) {
	generatorConfigs, err := findAllGeneratorConfigs(path)
	if err != nil {
		log.Errorf("Unable to load Generator configs from %s", path)
		return
	}
	ngc, err := findAllGeneratorConfigs(common.AssetsPath)
	if err != nil {
		log.Errorf("Unable to load Generator configs from %s", common.AssetsPath)
		return
	}
	generatorConfigs = append(generatorConfigs, ngc...)
	//TODO: Allow selecting generators
	for _, gc := range generatorConfigs {
		gt, ok := generatorRegistry[gc.Spec.Class]
		if !ok {
			log.Errorf("Unknown Generator Class, Ignoring %s (%s)", gc.Spec.FilePath, gc.Spec.Class)
			continue
		}
		generator, ok := reflect.New(gt).Interface().(Generator)
		if !ok {
			log.Errorf("Unable to instantiate Generator, Ignoring %s (%s)", gc.Spec.FilePath, gc.Spec.Class)
			continue
		}
		generator.Init(gc)
		generators = append(generators, generator)
	}
}

func findAllGeneratorConfigs(path string) ([]generatortypes.Generator, error) {
	yamlpaths, err := common.GetFilesByExt(path, []string{".yaml", ".yml"})
	if err != nil {
		log.Errorf("Unable to fetch yaml files at path %s Error: %q", path, err)
		return nil, err
	}
	generators := []generatortypes.Generator{}
	for _, path := range yamlpaths {
		g := generatortypes.Generator{
			Spec: generatortypes.GeneratorSpec{
				Class:    reflect.TypeOf(GenericGenerator{}).Name(),
				FilePath: path,
			},
		}
		if err := common.ReadMove2KubeYaml(path, &g); err != nil || g.Kind != string(generatortypes.GeneratorKind) {
			continue
		}
		generators = append(generators, g)
	}
	return generators, nil
}

// GetAllContainerBuildStrategies returns all translator types
func GetAllContainerBuildStrategies() []string {
	cbs := []string{}
	for _, c := range getAllContainerizers() {
		cbs = append(cbs, string(c.GetContainerBuildStrategy()))
	}
	cbs = append(cbs, string((new(ReuseDockerfileContainerizer)).GetContainerBuildStrategy()), string((new(ManualContainerizer)).GetContainerBuildStrategy()))
	return cbs
}

// GetGenerationOptions returns GenerationOptions for given sourcepath
func GetGenerationOptions(plan plantypes.Plan, sourcepath string) []plantypes.GenerationOption {
	cops := []plantypes.ContainerizationOption{}
	for _, containerizer := range containerizers {
		for _, option := range containerizer.GetTargetOptions(plan, sourcepath) {
			cops = append(cops, plantypes.ContainerizationOption{
				BuildType:   containerizer.GetContainerBuildStrategy(),
				ID:          option,
				ContextPath: sourcepath,
			})
		}
	}
	return cops
}

// GetContainer get the container for a service
func GetContainer(serviceName string, containerizationOption plantypes.ContainerizationOption, plan plantypes.Plan) (irtypes.Container, error) {
	log.Debugf("Containerizing %s using %s", serviceName, containerizationOption.BuildType)
	return containerizers[containerizationOption.BuildType].GetContainer(serviceName, containerizationOption, plan)
}
