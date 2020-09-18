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

package transform

import (
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"

	composetypes "github.com/docker/cli/cli/compose/types"

	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
)

// ComposeTransformer implements Transformer interface
type ComposeTransformer struct {
	Compose    composeConfig
	Containers []irtypes.Container
	Name       string
}

type composeConfig struct {
	Version  string
	Services map[string]composetypes.ServiceConfig   `yaml:",omitempty"`
	Networks map[string]composetypes.NetworkConfig   `yaml:",omitempty"`
	Volumes  map[string]composetypes.VolumeConfig    `yaml:",omitempty"`
	Secrets  map[string]composetypes.SecretConfig    `yaml:",omitempty"`
	Configs  map[string]composetypes.ConfigObjConfig `yaml:",omitempty"`
}

// Transform translates intermediate representation to destination objects
func (kt *ComposeTransformer) Transform(ir irtypes.IR) error {
	log.Debugf("Starting Compose transform")
	log.Debugf("Total services to be transformed : %d", len(ir.Services))

	kt.Name = ir.Name
	kt.Containers = ir.Containers
	kt.Compose = composeConfig{
		Version:  "3.5",
		Services: make(map[string]composetypes.ServiceConfig),
	}

	var exposedPort uint32 = 8080
	for _, service := range ir.Services {
		for _, container := range service.Containers {
			ports := make([]composetypes.ServicePortConfig, 0)
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
			kt.Compose.Services[service.Name] = serviceConfig
			//TODO: Support more than one container
		}
	}
	log.Debugf("Total transformed objects : %d", len(kt.Compose.Services))

	return nil
}

// WriteObjects writes Transformed objects to filesystem
func (kt *ComposeTransformer) WriteObjects(outpath string) error {
	err := os.MkdirAll(outpath, common.DefaultDirectoryPermission)
	if err != nil {
		log.Errorf("Unable to create output directory %s : %s", outpath, err)
	}
	artifactspath := filepath.Join(outpath, "docker-compose.yaml")
	err = common.WriteYaml(artifactspath, kt.Compose)
	if err != nil {
		log.Errorf("Unable to write docker compose file %s : %s", artifactspath, err)
	}
	return nil
}
