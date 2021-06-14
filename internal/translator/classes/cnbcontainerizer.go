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
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
)

// CNBContainerizer implements Containerizer interface
type CNBContainerizer struct {
	TConfig   translatortypes.Translator
	CNBConfig CNBContainerizerYamlConfig
	Env       environment.Environment
	CNBEnv    environment.Environment
}

type CNBContainerizerYamlConfig struct {
	BuilderImageName string `yaml:"cnbbuilderimage"`
}

type CNBTemplateConfig struct {
	CNBBuilder string `json:"CNBBuilder"`
	ImageName  string `json:"ImageName"`
}

func (t *CNBContainerizer) Init(tc translatortypes.Translator, env environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	t.CNBConfig = CNBContainerizerYamlConfig{}
	err = common.GetObjFromInterface(t.TConfig.Spec.Config, &t.CNBConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Translator %+v into %T : %s", t.TConfig.Spec.Config, t.CNBConfig, err)
		return err
	}

	t.CNBEnv, err = environment.NewEnvironment(tc.Name, t.Env.GetSourcePath(), "", &environment.Container{
		Image: t.CNBConfig.BuilderImageName,
	})
	if err != nil {
		logrus.Errorf("Unable to create CNB environment : %s", err)
		return err
	}
	return nil
}

func (t *CNBContainerizer) GetConfig() (translatortypes.Translator, environment.Environment) {
	return t.TConfig, t.Env
}

func (t *CNBContainerizer) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *CNBContainerizer) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	stdout, stderr, exitcode, err := t.CNBEnv.Exec("/cnb/lifecycle/detector "+t.CNBEnv.EncodePath(t.CNBEnv.GetSourcePath()), "")
	if err != nil {
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, nil
	}
	trans := plantypes.Translator{
		Mode:                   plantypes.ModeContainer,
		ArtifactTypes:          []string{plantypes.ContainerBuildTargetArtifactType},
		ExclusiveArtifactTypes: []string{plantypes.ContainerBuildTargetArtifactType},
		Paths:                  map[string][]string{plantypes.ProjectPathSourceArtifact: {dir}},
	}
	return nil, []plantypes.Translator{trans}, nil
}

func (t *CNBContainerizer) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *CNBContainerizer) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *CNBContainerizer) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan) ([]translatortypes.Patch, error) {
	return []translatortypes.Patch{{
		PathMappings: []translatortypes.PathMapping{{
			Type:     translatortypes.TemplatePathMappingType,
			SrcPath:  "buildcnb.sh",
			DestPath: translatorPlan.Paths[plantypes.ProjectPathSourceArtifact][0],
		}},
		Config: CNBTemplateConfig{
			ImageName: t.CNBConfig.BuilderImageName,
		},
	}}, nil
}

func (t *CNBContainerizer) TranslateIR(ir irtypes.IR, plan plantypes.Plan) ([]translatortypes.PathMapping, error) {
	return nil, nil
}
