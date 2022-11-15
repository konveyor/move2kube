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

package transformer

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
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
	Config              transformertypes.Transformer
	BuilderImageNameCfg *CNBContainerizerConfig
	Env                 *environment.Environment
	CNBEnv              *environment.Environment
}

// CNBContainerizerConfig contains the configuration options for the CNB containerizer transformer.
type CNBContainerizerConfig struct {
	Container environmenttypes.Container
}

const (
	// LinuxFileSeperator is used to join paths for linux container file system
	LinuxFileSeperator = "/"
)

// Init Initializes the transformer
func (t *CNBContainerizer) Init(tc transformertypes.Transformer, env *environment.Environment) error {
	t.Config = tc
	t.Env = env
	t.BuilderImageNameCfg = &CNBContainerizerConfig{}
	if err := common.GetObjFromInterface(t.Config.Spec.Config, t.BuilderImageNameCfg); err != nil {
		return fmt.Errorf("failed to load the config for Transformer %+v into %T . Error: %w", t.Config.Spec.Config, t.BuilderImageNameCfg, err)
	}
	t.BuilderImageNameCfg.Container.WorkingDir = filepath.Join(LinuxFileSeperator, "tmp")

	envInfo := environment.EnvInfo{
		Name:        tc.Name,
		ProjectName: t.Env.GetProjectName(),
		Source:      t.Env.GetEnvironmentSource(),
		EnvPlatformConfig: environmenttypes.EnvPlatformConfig{
			Container: t.BuilderImageNameCfg.Container,
			Platforms: []string{runtime.GOOS},
		},
		SpawnContainers: env.SpawnContainers,
	}
	var err error
	t.CNBEnv, err = environment.NewEnvironment(envInfo, nil)
	if err != nil {
		return fmt.Errorf("failed to create the CNB environment. Error: %w", err)
	}
	t.Env.AddChild(t.CNBEnv)
	return nil
}

// GetConfig returns the transformer config
func (t *CNBContainerizer) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *CNBContainerizer) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	path := dir
	cmd := environmenttypes.Command{"/cnb/lifecycle/detector", "-app", t.CNBEnv.Encode(path).(string)}
	stdout, stderr, exitcode, err := t.CNBEnv.Exec(cmd, nil)
	if err != nil {
		logrus.Debugf("CNB detector failed. exit code: %d error: %q\nstdout: %s\nstderr: %s", exitcode, err, stdout, stderr)
		return nil, fmt.Errorf("CNB detector failed with exitcode %d . Error: %q", exitcode, err)
	}
	if exitcode != 0 {
		logrus.Debugf("CNB detector gave a non-zero exit code. exit code: %d\nstdout: %s\nstderr: %s", exitcode, stdout, stderr)
		return nil, nil
	}
	detectedServices := []transformertypes.Artifact{{
		Paths:   map[transformertypes.PathType][]string{artifacts.ServiceDirPathType: {dir}},
		Configs: map[string]interface{}{artifacts.ImageNameConfigType: t.BuilderImageNameCfg},
	}}
	return map[string][]transformertypes.Artifact{"": detectedServices}, nil
}

// Transform transforms the artifacts
func (t *CNBContainerizer) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	createdArtifacts := []transformertypes.Artifact{}
	for _, newArtifact := range newArtifacts {
		var sConfig artifacts.ServiceConfig
		if err := newArtifact.GetConfig(artifacts.ServiceConfigType, &sConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		var iConfig artifacts.ImageName
		if err := newArtifact.GetConfig(artifacts.ImageNameConfigType, &iConfig); err != nil {
			logrus.Errorf("unable to load config for Transformer into %T . Error: %q", iConfig, err)
			continue
		}
		ir := irtypes.NewIR()
		ir.Name = t.Env.GetProjectName()
		if prevIR, ok := newArtifact.Configs[irtypes.IRConfigType].(irtypes.IR); ok {
			ir = prevIR
		}
		// Update an existing service with default port
		if service, ok := ir.Services[sConfig.ServiceName]; ok {
			if len(service.Containers) > 0 && len(service.Containers[0].Ports) == 0 {
				// Add the port to the k8s pod.
				service.Containers[0].Ports = []core.ContainerPort{{ContainerPort: common.DefaultServicePort}}
				// Forward the port on the k8s service to the k8s pod.
				port := networking.ServiceBackendPort{Number: common.DefaultServicePort}
				service.AddPortForwarding(port, port, "")
				ir.Services[sConfig.ServiceName] = service
			}
		} else {
			// Create a new container in a new service in the IR (new or existing)
			container := irtypes.NewContainer()
			container.AddExposedPort(common.DefaultServicePort)
			ir.AddContainer(iConfig.ImageName, container)
			irService := irtypes.NewServiceWithName(sConfig.ServiceName)
			serviceContainerPorts := []core.ContainerPort{}
			for _, eport := range container.ExposedPorts {
				// Add the port to the k8s pod.
				serviceContainerPorts = append(serviceContainerPorts, core.ContainerPort{ContainerPort: eport})
				// Forward the port on the k8s service to the k8s pod.
				port := networking.ServiceBackendPort{Number: eport}
				irService.AddPortForwarding(port, port, "")
			}
			serviceContainer := core.Container{Name: sConfig.ServiceName,
				Image: iConfig.ImageName,
				Ports: serviceContainerPorts}
			irService.Containers = []core.Container{serviceContainer}
			ir.Services[sConfig.ServiceName] = irService
		}
		newArtifact.Configs[irtypes.IRConfigType] = ir
		createdArtifacts = append(createdArtifacts, transformertypes.Artifact{
			Name:    newArtifact.Name,
			Type:    artifacts.CNBDetectedServiceArtifactType,
			Paths:   newArtifact.Paths,
			Configs: newArtifact.Configs,
		})
	}
	return nil, createdArtifacts, nil
}
