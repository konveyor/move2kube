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

package analysers

import (
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// CNBContainerizer implements Containerizer interface
type CNBContainerizer struct {
	TConfig   transformertypes.Transformer
	CNBConfig CNBContainerizerYamlConfig
	Env       environment.Environment
	CNBEnv    environment.Environment
}

// CNBContainerizerYamlConfig represents the configuration of the CNBBuilder
type CNBContainerizerYamlConfig struct {
	BuilderImageName string `yaml:"CNBBuilder"`
}

// Init Initializes the transformer
func (t *CNBContainerizer) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.TConfig = tc
	t.Env = env
	t.CNBConfig = CNBContainerizerYamlConfig{}
	err = common.GetObjFromInterface(t.TConfig.Spec.Config, &t.CNBConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.TConfig.Spec.Config, t.CNBConfig, err)
		return err
	}

	t.CNBEnv, err = environment.NewEnvironment(tc.Name, t.Env.ProjectName, t.Env.GetEnvironmentSource(), "", "", nil, environmenttypes.Container{
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

// GetConfig returns the transformer config
func (t *CNBContainerizer) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.TConfig, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *CNBContainerizer) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
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
		Mode:              transformertypes.ModeContainer,
		ArtifactTypes:     []string{artifacts.ContainerBuildArtifactType},
		BaseArtifactTypes: []string{artifacts.ContainerBuildArtifactType},
		Paths:             map[string][]string{artifacts.ProjectPathPathType: {dir}},
		Configs: map[string]interface{}{artifacts.CNBMetadataConfigType: artifacts.CNBMetadataConfig{
			CNBBuilder: t.CNBConfig.BuilderImageName,
		}},
	}
	return nil, []plantypes.Transformer{trans}, nil
}

// Transform transforms the artifacts
func (t *CNBContainerizer) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (tPathMappings []transformertypes.PathMapping, tArtifacts []transformertypes.Artifact, err error) {
	tArtifacts = []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		var cConfig artifacts.CNBMetadataConfig
		err = a.GetConfig(artifacts.CNBMetadataConfigType, &cConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", cConfig, err)
			continue
		}
		if cConfig.ImageName == "" {
			cConfig.ImageName = sConfig.ServiceName
		}
		a.Configs[artifacts.CNBMetadataConfigType] = cConfig
		tArtifacts = append(tArtifacts, transformertypes.Artifact{
			Name:     a.Name,
			Artifact: artifacts.CNBMetadataArtifactType,
			Paths:    a.Paths,
			Configs:  a.Configs,
		})
	}
	return nil, tArtifacts, nil
}
