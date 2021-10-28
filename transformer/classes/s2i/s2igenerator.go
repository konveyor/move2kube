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

package s2i

import (
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	irtypes "github.com/konveyor/move2kube/types/ir"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/apis/networking"
)

// S2IGenerator implements Transformer interface
type S2IGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *S2IGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *S2IGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in the base directory
func (t *S2IGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *S2IGenerator) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// Transform transforms the artifacts
func (t *S2IGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	newartifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		s2iConfig := artifacts.S2IMetadataConfig{}
		err := a.GetConfig(artifacts.S2IMetadataConfigType, &s2iConfig)
		if err != nil {
			logrus.Errorf("Unable to read S2I Template config : %s", err)
		}
		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		var sConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}
		if s2iConfig.ImageName == "" {
			s2iConfig.ImageName = common.MakeStringContainerImageNameCompliant(sConfig.ServiceName)
		}
		ir := irtypes.NewIR()
		ir.Name = t.Env.GetProjectName()
		container := irtypes.NewContainer()
		container.AddExposedPort(common.DefaultServicePort)
		ir.AddContainer(s2iConfig.ImageName, container)
		serviceContainer := core.Container{Name: sConfig.ServiceName}
		serviceContainer.Image = s2iConfig.ImageName
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

		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			DestPath: common.DefaultSourceDir,
		}, transformertypes.PathMapping{
			Type:           transformertypes.TemplatePathMappingType,
			SrcPath:        filepath.Join(t.Env.Context, t.Config.Spec.TemplatesDir),
			DestPath:       filepath.Join(common.DefaultSourceDir, relSrcPath),
			TemplateConfig: s2iConfig,
		})
		newartifacts = append(newartifacts, transformertypes.Artifact{
			Name:     t.Env.ProjectName,
			Artifact: artifacts.NewImagesArtifactType,
			Configs: map[string]interface{}{
				artifacts.NewImagesConfigType: artifacts.NewImages{
					ImageNames: []string{s2iConfig.ImageName},
				},
			},
		}, transformertypes.Artifact{
			Name:     s2iConfig.ImageName,
			Artifact: artifacts.ContainerImageBuildScriptArtifactType,
			Paths: map[string][]string{
				artifacts.ContainerImageBuildShScriptPathType:  {filepath.Join(common.DefaultSourceDir, relSrcPath, "s2ibuild.sh")},
				artifacts.ContainerImageBuildBatScriptPathType: {filepath.Join(common.DefaultSourceDir, relSrcPath, "s2ibuild.bat")},
			},
		})
	}
	return pathMappings, newartifacts, nil
}
