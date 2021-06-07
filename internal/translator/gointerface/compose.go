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

package gointerface

import (
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/translator/gointerface/compose"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/konveyor/move2kube/types/plan"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	log "github.com/sirupsen/logrus"
)

const (
	Name = "Compose"

	// composeFileSourceArtifactType defines the source artifact type of Docker compose
	composeFileSourceArtifactType = "DockerCompose"
	// imageInfoSourceArtifactType defines the source artifact type of image info
	imageInfoSourceArtifactType = "ImageInfo"
	// dockerfileSourceArtifactType defines the source artifact type of dockerfile
	dockerfileSourceArtifactType = "Dockerfile"
)

// Compose implements Translator interface
type Compose struct {
	Name string
}

type composeConfig struct {
	serviceName string `yaml:"serviceName"`
}

func (t *Compose) init() {
	t.Name = Name
}

func (t *Compose) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	//Load images
	yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
	if err != nil {
		log.Errorf("Unable to fetch yaml files at path %s Error: %q", dir, err)
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

	//Fill data into plan
	services := map[string]plantypes.Service{}
	for _, path := range yamlpaths {
		currServices := t.getServicesFromComposeFile(path, imageMetadataPaths)
		plantypes.MergeServices(services, currServices)
	}

	return services, nil, nil
}

func (t *Compose) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Compose) KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return nil, nil, nil
}

func (t *Compose) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *Compose) PlanDetect(plantypes.Plan) ([]plantypes.Translator, error) {
	return nil, nil
}

func (t *Compose) Translate(serviceName string) map[string]translatortypes.Patch {
	ir := irtypes.NewIR(plan)
	service := plan.Spec.Services[serviceName]
	for _, sa := range service.SourceArtifacts {
		if sa.Type == plantypes.ComposeFileArtifactType {
			for _, path := range sa.Artifacts {
				log.Debugf("File %s being loaded from compose service : %s", path, sa.ID)
				// Try v3 first and if it fails try v1v2
				if cir, errV3 := new(compose.V3Loader).ConvertToIR(path, sa.ID); errV3 == nil {
					ir.Merge(cir)
					log.Debugf("compose v3 translator returned %d services", len(ir.Services))
				} else if cir, errV1V2 := new(compose.V1V2Loader).ConvertToIR(path, sa.ID); errV1V2 == nil {
					ir.Merge(cir)
					log.Debugf("compose v1v2 translator returned %d services", len(ir.Services))
				} else {
					log.Errorf("Unable to parse the docker compose file at path %s Error V3: %q Error V1V2: %q", path, errV3, errV1V2)
				}
			}
		}
	}
	for _, sa := range service.SourceArtifacts {
		if sa.Type != plantypes.ImageInfoArtifactType {
			continue
		}
		for _, path := range sa.Artifacts {
			imgMD := collecttypes.ImageInfo{}
			if err := common.ReadMove2KubeYaml(path, &imgMD); err != nil {
				log.Errorf("Failed to read image info yaml at path %s Error: %q", path, err)
				continue
			}
			ir.AddContainer(irtypes.NewContainerFromImageInfo(imgMD))
		}
	}

	return ir, nil
}

func (t *Compose) getService(composeFilePath string, serviceName string, serviceImage string, relContextPath string, relDockerfilePath string, imageMetadataPaths map[string]string) plantypes.Translator {
	ct := plantypes.Translator{
		Mode:                   plantypes.ModeContainer,
		Name:                   t.Name,
		ArtifactTypes:          []string{plantypes.K8sServiceMetadataTargetArtifactType, plantypes.ContainerBuildTargetArtifactType},
		ExclusiveArtifactTypes: []string{plantypes.K8sServiceMetadataTargetArtifactType, plantypes.ContainerBuildTargetArtifactType},
		Config: composeConfig{
			serviceName: serviceName,
		},
		Paths: map[string][]string{
			composeFileSourceArtifactType: []string{
				composeFilePath,
			},
		},
	}
	if imagepath, ok := imageMetadataPaths[serviceImage]; ok {
		ct.Paths[imageInfoSourceArtifactType] = common.MergeStringSlices(ct.Paths[imageInfoSourceArtifactType], imagepath)
	}
	log.Debugf("Found a docker compose service : %s", serviceName)
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
		ct.Paths[plantypes.ProjectPathSourceArtifact] = common.MergeStringSlices(ct.Paths[dockerfileSourceArtifactType], dockerfilePath)
	}
	return ct
}

func (c *Compose) getServicesFromComposeFile(composeFilePath string, imageMetadataPaths map[string]string) map[string]plantypes.Service {
	services := map[string]plantypes.Service{}
	// Try v3 first and if it fails try v1v2
	if dc, errV3 := compose.ParseV3(composeFilePath); errV3 == nil {
		log.Debugf("Found a docker compose file at path %s", composeFilePath)
		for _, service := range dc.Services {
			services[service.Name] = []plantypes.Translator{c.getService(composeFilePath, service.Name, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)}
		}
	} else if dc, errV1V2 := compose.ParseV2(composeFilePath); errV1V2 == nil {
		log.Debugf("Found a docker compose file at path %s", composeFilePath)
		servicesMap := dc.ServiceConfigs.All()
		for serviceName, service := range servicesMap {
			services[serviceName] = []plantypes.Translator{c.getService(composeFilePath, serviceName, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)}
		}
	} else {
		log.Debugf("Failed to parse file at path %s as a docker compose file. Error V3: %q Error V1V2: %q", composeFilePath, errV3, errV1V2)
	}
	return services
}
