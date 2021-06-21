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

package generators

import (
	"os"
	"path/filepath"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

const (
	ComposeArtifacts = "ComposeYamls"
)

// ComposeGenerator implements Transformer interface
type ComposeGenerator struct {
	Config transformertypes.Transformer
	Env    environment.Environment
}

type composeObj struct {
	Version  string
	Services map[string]composetypes.ServiceConfig   `yaml:",omitempty"`
	Networks map[string]composetypes.NetworkConfig   `yaml:",omitempty"`
	Volumes  map[string]composetypes.VolumeConfig    `yaml:",omitempty"`
	Secrets  map[string]composetypes.SecretConfig    `yaml:",omitempty"`
	Configs  map[string]composetypes.ConfigObjConfig `yaml:",omitempty"`
}

func (t *ComposeGenerator) Init(tc transformertypes.Transformer, env environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

func (t *ComposeGenerator) GetConfig() (transformertypes.Transformer, environment.Environment) {
	return t.Config, t.Env
}

func (t *ComposeGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *ComposeGenerator) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

func (t *ComposeGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) (pathMappings []transformertypes.PathMapping, createdArtifacts []transformertypes.Artifact, err error) {
	pathMappings = []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		if a.Artifact != transformertypes.IRArtifactType {
			continue
		}
		var ir irtypes.IR
		err := a.GetConfig(transformertypes.IRConfigType, &ir)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", ir, err)
			continue
		}
		logrus.Debugf("Starting Compose transform")
		logrus.Debugf("Total services to be transformed : %d", len(ir.Services))
		c := composeObj{
			Version:  "3.5",
			Services: map[string]composetypes.ServiceConfig{},
		}
		var exposedPort uint32 = 8080
		for _, service := range ir.Services {
			for _, container := range service.Containers {
				ports := []composetypes.ServicePortConfig{}
				for _, port := range container.Ports {
					ports = append(ports, composetypes.ServicePortConfig{
						Target:    uint32(port.ContainerPort),
						Published: exposedPort,
					})
					exposedPort++
				}
				env := make(composetypes.MappingWithEquals)
				for _, e := range container.Env {
					env[e.Name] = &e.Value
				}
				serviceConfig := composetypes.ServiceConfig{
					ContainerName: container.Name,
					Image:         container.Image,
					Ports:         ports,
					Environment:   env,
				}
				c.Services[service.Name] = serviceConfig
			}
		}
		logrus.Debugf("Total transformed objects : %d", len(c.Services))
		composePath := filepath.Join(common.DeployDir, "compose")
		absComposePath := filepath.Join(t.Env.TempPath, composePath)
		if err := os.MkdirAll(absComposePath, common.DefaultDirectoryPermission); err != nil {
			logrus.Errorf("Unable to create output directory %s : %s", common.TempPath, err)
		}
		if err := common.WriteYaml(filepath.Join(absComposePath, "docker-compose.yaml"), c); err != nil {
			logrus.Errorf("Unable to write docker compose file %s : %s", absComposePath, err)
		}
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  absComposePath,
			DestPath: composePath,
		})
	}
	return pathMappings, nil, nil
}
