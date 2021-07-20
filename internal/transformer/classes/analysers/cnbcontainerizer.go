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
	"errors"
	"path/filepath"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/environment/container"
	"github.com/konveyor/move2kube/internal/common"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/networking"
)

// CNBContainerizer implements Containerizer interface
type CNBContainerizer struct {
	Config    transformertypes.Transformer
	CNBConfig CNBContainerizerYamlConfig
	Env       *environment.Environment
	CNBEnv    *environment.Environment
}

// CNBContainerizerYamlConfig represents the configuration of the CNBBuilder
type CNBContainerizerYamlConfig struct {
	BuilderImageName string `yaml:"CNBBuilder"`
}

// Init Initializes the transformer
func (t *CNBContainerizer) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	t.CNBConfig = CNBContainerizerYamlConfig{}
	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.CNBConfig)
	if err != nil {
		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.CNBConfig, err)
		return err
	}
	envInfo := environment.EnvInfo{
		Name:          tc.Name,
		ProjectName:   t.Env.GetProjectName(),
		TargetCluster: t.Env.TargetCluster,
		Source:        t.Env.GetEnvironmentSource(),
	}
	t.CNBEnv, err = environment.NewEnvironment(envInfo, nil, environmenttypes.Container{
		Image:      t.CNBConfig.BuilderImageName,
		WorkingDir: filepath.Join(string(filepath.Separator), "tmp"),
	})
	if err != nil {
		if !container.IsDisabled() {
			logrus.Errorf("Unable to create CNB environment : %s", err)
			return err
		}
		return &transformertypes.TransformerDisabledError{Err: err}
	}
	t.Env.AddChild(t.CNBEnv)
	return nil
}

// GetConfig returns the transformer config
func (t *CNBContainerizer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *CNBContainerizer) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *CNBContainerizer) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	path := dir
	cmd := environmenttypes.Command{
		"/cnb/lifecycle/detector", "-app", t.CNBEnv.Encode(path).(string)}
	stdout, stderr, exitcode, err := t.CNBEnv.Exec(cmd)
	if err != nil {
		if errors.Is(err, &environment.EnvironmentNotActiveError{}) {
			logrus.Debugf("%s", err)
			return nil, nil, err
		}
		logrus.Errorf("Detect failed %s : %s : %d : %s", stdout, stderr, exitcode, err)
		return nil, nil, err
	} else if exitcode != 0 {
		logrus.Debugf("Detect did not succeed %s : %s : %d", stdout, stderr, exitcode)
		return nil, nil, nil
	}
	trans := transformertypes.TransformerPlan{
		Mode:              transformertypes.ModeContainer,
		ArtifactTypes:     []string{artifacts.ContainerBuildArtifactType},
		BaseArtifactTypes: []string{artifacts.ContainerBuildArtifactType},
		Paths:             map[string][]string{artifacts.ProjectPathPathType: {dir}},
		Configs: map[string]interface{}{artifacts.CNBMetadataConfigType: artifacts.CNBMetadataConfig{
			CNBBuilder: t.CNBConfig.BuilderImageName,
		}},
	}
	return nil, []transformertypes.TransformerPlan{trans}, nil
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
			cConfig.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		a.Configs[artifacts.CNBMetadataConfigType] = cConfig
		ir := irtypes.NewIR()
		ir.Name = t.Env.GetProjectName()
		container := irtypes.NewContainer()
		container.AddExposedPort(common.DefaultServicePort)
		ir.AddContainer(cConfig.ImageName, container)
		serviceContainer := core.Container{Name: sConfig.ServiceName}
		serviceContainer.Image = cConfig.ImageName
		irService := irtypes.NewServiceWithName(sConfig.ServiceName)
		serviceContainerPorts := []core.ContainerPort{}
		for _, port := range container.ExposedPorts {
			// Add the port to the k8s pod.
			serviceContainerPort := core.ContainerPort{ContainerPort: port}
			serviceContainerPorts = append(serviceContainerPorts, serviceContainerPort)
			// Forward the port on the k8s service to the k8s pod.
			podPort := networking.ServiceBackendPort{Number: port}
			servicePort := podPort
			irService.AddPortForwarding(servicePort, podPort, "")
		}
		serviceContainer.Ports = serviceContainerPorts
		irService.Containers = []core.Container{serviceContainer}
		ir.Services[sConfig.ServiceName] = irService
		tArtifacts = append(tArtifacts, transformertypes.Artifact{
			Name:     a.Name,
			Artifact: artifacts.CNBMetadataArtifactType,
			Paths:    a.Paths,
			Configs:  a.Configs,
		}, transformertypes.Artifact{
			Name:     t.Env.GetProjectName(),
			Artifact: irtypes.IRArtifactType,
			Configs: map[string]interface{}{
				irtypes.IRConfigType: ir,
			},
		})
	}
	return nil, tArtifacts, nil
}
