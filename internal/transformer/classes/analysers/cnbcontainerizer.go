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

package analysers

import (
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

// CNBContainerizer implements Containerizer interface
type CNBContainerizer struct {
	TConfig   transformertypes.Transformer
	CNBConfig CNBContainerizerYamlConfig
	Env       environment.Environment
	CNBEnv    environment.Environment
}

type CNBContainerizerYamlConfig struct {
	BuilderImageName string `yaml:"cnbbuilderimage"`
}

type CNBTemplateConfig struct {
	CNBBuilder string `json:"CNBBuilder"`
	ImageName  string `json:"ImageName,omitempty"`
}

func (t *CNBContainerizer) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	t.CNBConfig = CNBContainerizerYamlConfig{}
	err = common.GetObjFromInterface(t.TConfig.Spec.Config, &t.CNBConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.TConfig.Spec.Config, t.CNBConfig, err)
		return err
	}

	t.CNBEnv, err = environment.NewEnvironment(tc.Name, t.Env.GetWorkspaceSource(), "", environmenttypes.Container{
		Image:      t.CNBConfig.BuilderImageName,
		WorkingDir: filepath.Join(string(filepath.Separator), "tmp"),
	})
	if err != nil {
		logrus.Errorf("Unable to create CNB environment : %s", err)
		return err
	}
	t.Env.AddChild(t.CNBEnv)
	return nil
}

func (t *CNBContainerizer) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

func (t *CNBContainerizer) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *CNBContainerizer) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	path := dir
	cmd := environmenttypes.Command{
		"/cnb/lifecycle/detector", "-app", t.CNBEnv.Encode(path).(string)}
	stdout, stderr, exitcode, err := t.CNBEnv.Exec(cmd)
	if err != nil {
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, nil
	}
	trans := plantypes.Transformer{
		Mode:                   plantypes.ModeContainer,
		ArtifactTypes:          []string{transformertypes.ContainerBuildArtifactType},
		ExclusiveArtifactTypes: []string{transformertypes.ContainerBuildArtifactType},
		Paths:                  map[string][]string{plantypes.ProjectPathPathType: {dir}},
		Configs: map[string]interface{}{transformertypes.TemplateConfigType: CNBTemplateConfig{
			CNBBuilder: t.CNBConfig.BuilderImageName,
		}},
	}
	return nil, []plantypes.Transformer{trans}, nil
}

func (t *CNBContainerizer) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	artifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		artifacts = append(artifacts, transformertypes.Artifact{
			Name:     a.Name,
			Artifact: transformertypes.CNBMetadataArtifactType,
			Paths:    a.Paths,
			Configs:  a.Configs,
		})
	}
	return nil, artifacts, nil
}
