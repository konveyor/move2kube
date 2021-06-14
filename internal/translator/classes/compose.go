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

package classes

import (
	"os"
	"path/filepath"

	composetypes "github.com/docker/cli/cli/compose/types"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/translator/classes/compose"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
)

const (
	// composeFileSourceArtifactType defines the source artifact type of Docker compose
	composeFileSourceArtifactType = "DockerCompose"
	// imageInfoSourceArtifactType defines the source artifact type of image info
	imageInfoSourceArtifactType = "ImageInfo"
	// dockerfileSourceArtifactType defines the source artifact type of dockerfile
	dockerfileSourceArtifactType = "Dockerfile"
)

const (
	ComposeArtifacts = "ComposeYamls"
)

// Compose implements Translator interface
type Compose struct {
	Config      translatortypes.Translator
	Environment environment.Environment
}

type ComposeConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
}

type composeObj struct {
	Version  string
	Services map[string]composetypes.ServiceConfig   `yaml:",omitempty"`
	Networks map[string]composetypes.NetworkConfig   `yaml:",omitempty"`
	Volumes  map[string]composetypes.VolumeConfig    `yaml:",omitempty"`
	Secrets  map[string]composetypes.SecretConfig    `yaml:",omitempty"`
	Configs  map[string]composetypes.ConfigObjConfig `yaml:",omitempty"`
}

func (t *Compose) Init(tc translatortypes.Translator, env environment.Environment) (err error) {
	t.Config = tc
	t.Environment = env
	return nil
}

func (t *Compose) GetConfig() (translatortypes.Translator, environment.Environment) {
	return t.Config, t.Environment
}

func (t *Compose) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
	if err != nil {
		logrus.Errorf("Unable to fetch yaml files at path %s Error: %q", dir, err)
		return nil, nil, err
	}
	imageMetadataPaths := map[string]string{}
	for _, path := range yamlpaths {
		im := collecttypes.ImageInfo{}
		if err := common.ReadMove2KubeYaml(path, &im); err != nil || im.Kind != string(collecttypes.ImageMetadataKind) {
			continue
		}
		for _, imagetag := range im.Spec.Tags {
			imageMetadataPaths[imagetag] = path
		}
	}
	services := map[string]plantypes.Service{}
	for _, path := range yamlpaths {
		currServices := t.getServicesFromComposeFile(path, imageMetadataPaths)
		services = plantypes.MergeServices(services, currServices)
	}
	logrus.Debugf("Docker compose services : %+v", services)
	return services, nil, nil
}

func (t *Compose) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Compose) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *Compose) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	ts := []plantypes.Translator{{
		Mode:          plantypes.ModeContainer,
		ArtifactTypes: []string{ComposeArtifacts},
	}}
	return ts, nil
}

func (t *Compose) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan) ([]translatortypes.Patch, error) {
	var config ComposeConfig
	err := common.GetObjFromInterface(translatorPlan.Config, &config)
	if err != nil {
		logrus.Errorf("unable to load config for Translator %+v into %T : %s", translatorPlan.Config, config, err)
		return nil, err
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

func (t *Compose) TranslateIR(ir irtypes.IR, plan plantypes.Plan) ([]translatortypes.PathMapping, error) {
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

	absComposePath := filepath.Join(t.Environment.TempDir, composePath)
	if err := os.MkdirAll(absComposePath, common.DefaultDirectoryPermission); err != nil {
		logrus.Errorf("Unable to create output directory %s : %s", common.TempPath, err)
	}
	if err := common.WriteYaml(filepath.Join(absComposePath, "docker-compose.yaml"), c); err != nil {
		logrus.Errorf("Unable to write docker compose file %s : %s", absComposePath, err)
	}
	return []translatortypes.PathMapping{{
		Type:     translatortypes.DefaultPathMappingType,
		SrcPath:  absComposePath,
		DestPath: composePath,
	}}, nil
}

func (t *Compose) getService(composeFilePath string, serviceName string, serviceImage string, relContextPath string, relDockerfilePath string, imageMetadataPaths map[string]string) plantypes.Translator {
	ct := plantypes.Translator{
		Mode:                   plantypes.ModeContainer,
		ArtifactTypes:          []string{plantypes.K8sServiceMetadataTargetArtifactType, plantypes.ContainerBuildTargetArtifactType},
		ExclusiveArtifactTypes: []string{plantypes.K8sServiceMetadataTargetArtifactType, plantypes.ContainerBuildTargetArtifactType},
		Config: ComposeConfig{
			ServiceName: serviceName,
		},
		Paths: map[string][]string{
			composeFileSourceArtifactType: {
				composeFilePath,
			},
		},
	}
	if imagepath, ok := imageMetadataPaths[serviceImage]; ok {
		ct.Paths[imageInfoSourceArtifactType] = common.MergeStringSlices(ct.Paths[imageInfoSourceArtifactType], imagepath)
	}
	logrus.Debugf("Found a docker compose service : %s", serviceName)
	if relContextPath != "" {
		composeFileDir := filepath.Dir(composeFilePath)
		contextPath := filepath.Join(composeFileDir, relContextPath)
		if filepath.IsAbs(relContextPath) {
			contextPath = relContextPath // this happens with v1v2 parser
		}
		dockerfilePath := filepath.Join(contextPath, "Dockerfile")
		if relDockerfilePath != "" {
			dockerfilePath = filepath.Join(contextPath, relDockerfilePath)
			if filepath.IsAbs(relDockerfilePath) {
				dockerfilePath = relDockerfilePath // this happens with v1v2 parser
			}
		}
		// Add reuse Dockerfile containerization option
		ct.Paths[dockerfileSourceArtifactType] = common.MergeStringSlices(ct.Paths[dockerfileSourceArtifactType], dockerfilePath)
		ct.Paths[plantypes.ProjectPathSourceArtifact] = common.MergeStringSlices(ct.Paths[plantypes.ProjectPathSourceArtifact], contextPath)
	}
	return ct
}

func (c *Compose) getServicesFromComposeFile(composeFilePath string, imageMetadataPaths map[string]string) map[string]plantypes.Service {
	services := map[string]plantypes.Service{}
	// Try v3 first and if it fails try v1v2
	if dc, errV3 := compose.ParseV3(composeFilePath); errV3 == nil {
		logrus.Debugf("Found a docker compose file at path %s", composeFilePath)
		for _, service := range dc.Services {
			services[service.Name] = []plantypes.Translator{c.getService(composeFilePath, service.Name, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)}
		}
	} else if dc, errV1V2 := compose.ParseV2(composeFilePath); errV1V2 == nil {
		logrus.Debugf("Found a docker compose file at path %s", composeFilePath)
		servicesMap := dc.ServiceConfigs.All()
		for serviceName, service := range servicesMap {
			services[serviceName] = []plantypes.Translator{c.getService(composeFilePath, serviceName, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)}
		}
	} else {
		logrus.Debugf("Failed to parse file at path %s as a docker compose file. Error V3: %q Error V1V2: %q", composeFilePath, errV3, errV1V2)
	}
	return services
}

// newContainerFromImageInfo creates a new container from image info
func newContainerFromImageInfo(i collecttypes.ImageInfo) irtypes.ContainerImage {
	c := irtypes.NewContainer()
	c.ExposedPorts = i.Spec.PortsToExpose
	c.UserID = i.Spec.UserID
	c.AccessedDirs = i.Spec.AccessedDirs
	return c
}
