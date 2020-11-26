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

package source

import (
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/source/compose"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

// ComposeTranslator implements Translator interface
type ComposeTranslator struct {
}

func (c *ComposeTranslator) newService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, c.GetTranslatorType())
	service.AddSourceType(plantypes.ComposeSourceTypeValue)
	service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue //TODO: Identify when to use enhance
	return service
}

func (c *ComposeTranslator) getReuseService(composeFilePath string, serviceName string, serviceImage string, imageMetadataPaths map[string]string) plantypes.Service {
	service := c.newService(serviceName)
	service.Image = serviceImage
	if service.Image == "" {
		service.Image = serviceName + ":latest"
	}
	service.UpdateContainerBuildPipeline = false
	service.UpdateDeployPipeline = true
	service.AddSourceArtifact(plantypes.ComposeFileArtifactType, composeFilePath)
	if imagepath, ok := imageMetadataPaths[serviceImage]; ok {
		service.AddSourceArtifact(plantypes.ImageInfoArtifactType, imagepath)
	}
	return service
}

func (c *ComposeTranslator) getReuseAndReuseDockerfileServices(composeFilePath string, serviceName string, serviceImage string, relContextPath string, relDockerfilePath string, imageMetadataPaths map[string]string) []plantypes.Service {
	services := []plantypes.Service{}
	serviceName = common.NormalizeForServiceName(serviceName)
	log.Debugf("Found a docker compose service : %s", serviceName)
	if relContextPath != "" {
		// Add reuse Dockerfile containerization option
		reuseDockerfileService := c.getReuseService(composeFilePath, serviceName, serviceImage, imageMetadataPaths)

		reuseDockerfileService.ContainerBuildType = plantypes.ReuseDockerFileContainerBuildTypeValue
		reuseDockerfileService.UpdateContainerBuildPipeline = true
		reuseDockerfileService.UpdateDeployPipeline = true

		composeFileDir := filepath.Dir(composeFilePath)
		contextPath := filepath.Join(composeFileDir, relContextPath)
		if filepath.IsAbs(relContextPath) {
			contextPath = relContextPath // this happens with v1v2 parser
		}
		reuseDockerfileService.AddSourceType(plantypes.DirectorySourceTypeValue)
		reuseDockerfileService.AddBuildArtifact(plantypes.SourceDirectoryBuildArtifactType, contextPath)

		dockerfilePath := filepath.Join(contextPath, "Dockerfile")
		if relDockerfilePath != "" {
			dockerfilePath = filepath.Join(contextPath, relDockerfilePath)
			if filepath.IsAbs(relDockerfilePath) {
				dockerfilePath = relDockerfilePath // this happens with v1v2 parser
			}
		}
		reuseDockerfileService.AddSourceArtifact(plantypes.DockerfileArtifactType, dockerfilePath)
		reuseDockerfileService.ContainerizationTargetOptions = append(reuseDockerfileService.ContainerizationTargetOptions, dockerfilePath)

		services = append(services, reuseDockerfileService)
	}
	// Add reuse containerization
	reuseService := c.getReuseService(composeFilePath, serviceName, serviceImage, imageMetadataPaths)
	services = append(services, reuseService)
	return services
}

func (c *ComposeTranslator) getServicesFromComposeFile(composeFilePath string, imageMetadataPaths map[string]string) []plantypes.Service {
	services := []plantypes.Service{}
	// Try v3 first and if it fails try v1v2
	if dc, errV3 := compose.ParseV3(composeFilePath); errV3 == nil {
		log.Debugf("Found a docker compose file at path %s", composeFilePath)
		for _, service := range dc.Services {
			currServices := c.getReuseAndReuseDockerfileServices(composeFilePath, service.Name, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)
			services = append(services, currServices...)
		}
	} else if dc, errV1V2 := compose.ParseV2(composeFilePath); errV1V2 == nil {
		log.Debugf("Found a docker compose file at path %s", composeFilePath)
		servicesMap := dc.ServiceConfigs.All()
		for serviceName, service := range servicesMap {
			currServices := c.getReuseAndReuseDockerfileServices(composeFilePath, serviceName, service.Image, service.Build.Context, service.Build.Dockerfile, imageMetadataPaths)
			services = append(services, currServices...)
		}
	} else {
		log.Debugf("Failed to parse file at path %s as a docker compose file. Error V3: %q Error V1V2: %q", composeFilePath, errV3, errV1V2)
	}
	return services
}

// GetTranslatorType returns the translator type
func (c *ComposeTranslator) GetTranslatorType() plantypes.TranslationTypeValue {
	return plantypes.Compose2KubeTranslation
}

// GetServiceOptions returns the service options for inputPath
func (c *ComposeTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	//Load images
	yamlpaths, err := common.GetFilesByExt(inputPath, []string{".yaml", ".yml"})
	if err != nil {
		log.Errorf("Unable to fetch yaml files at path %s Error: %q", inputPath, err)
		return nil, err
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
	services := []plantypes.Service{}
	for _, path := range yamlpaths {
		currServices := c.getServicesFromComposeFile(path, imageMetadataPaths)
		services = append(services, currServices...)
	}

	return services, nil
}

// Translate translates the service to IR
func (c *ComposeTranslator) Translate(services []plantypes.Service, plan plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(plan)

	for _, service := range services {
		if service.TranslationType != c.GetTranslatorType() {
			log.Debugf("Expected service to have compose2kube translation type. Got %s . Skipping.", service.TranslationType)
			continue
		}
		for _, path := range service.SourceArtifacts[plantypes.ComposeFileArtifactType] {
			log.Debugf("File %s being loaded from compose service : %s", path, service.ServiceName)
			// Try v3 first and if it fails try v1v2
			if cir, errV3 := new(compose.V3Loader).ConvertToIR(path, plan, service); errV3 == nil {
				ir.Merge(cir)
				log.Debugf("compose v3 translator returned %d services", len(ir.Services))
			} else if cir, errV1V2 := new(compose.V1V2Loader).ConvertToIR(path, plan, service); errV1V2 == nil {
				ir.Merge(cir)
				log.Debugf("compose v1v2 translator returned %d services", len(ir.Services))
			} else {
				log.Errorf("Unable to parse the docker compose file at path %s Error V3: %q Error V1V2: %q", path, errV3, errV1V2)
			}
		}
		for _, path := range service.SourceArtifacts[plantypes.ImageInfoArtifactType] {
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
