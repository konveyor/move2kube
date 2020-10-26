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
	sourcetypes "github.com/konveyor/move2kube/internal/collector/sourcetypes"
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

// GetTranslatorType returns the translator type
func (c ComposeTranslator) GetTranslatorType() plantypes.TranslationTypeValue {
	return plantypes.Compose2KubeTranslation
}

// GetServiceOptions returns the service options for inputPath
func (c ComposeTranslator) GetServiceOptions(inputPath string, plan plantypes.Plan) ([]plantypes.Service, error) {
	servicesMap := map[string]plantypes.Service{}

	//Load images
	yamlpaths, err := common.GetFilesByExt(inputPath, []string{".yaml", ".yml"})
	if err != nil {
		log.Errorf("Unable to fetch yamls files at path %q Error: %q", inputPath, err)
		return nil, err
	}

	imagemetadatapaths := map[string]string{}
	for _, path := range yamlpaths {
		im := collecttypes.ImageInfo{}
		if err := common.ReadYaml(path, &im); err != nil || im.Kind != string(collecttypes.ImageMetadataKind) {
			continue
		}
		for _, imagetag := range im.Spec.Tags {
			imagemetadatapaths[imagetag] = path
		}
	}

	//Fill data into plan
	for _, path := range yamlpaths {
		dc := sourcetypes.DockerCompose{}
		if err := common.ReadYaml(path, &dc); err != nil {
			continue
		}
		for serviceName, dcservice := range dc.DCServices {
			log.Debugf("Found a Docker compose service : %s", serviceName)
			serviceName = common.NormalizeForServiceName(serviceName)
			if _, ok := servicesMap[serviceName]; !ok {
				servicesMap[serviceName] = c.newService(serviceName)
			}
			service := servicesMap[serviceName]
			service.Image = dcservice.Image
			if service.Image == "" {
				service.Image = serviceName + ":latest"
			}
			service.UpdateContainerBuildPipeline = false
			service.UpdateDeployPipeline = true
			service.AddSourceArtifact(plantypes.ComposeFileArtifactType, path)
			if imagepath, ok := imagemetadatapaths[dcservice.Image]; ok {
				service.AddSourceArtifact(plantypes.ImageInfoArtifactType, imagepath)
			}
			servicesMap[serviceName] = service
		}
	}

	services := []plantypes.Service{}
	for _, service := range servicesMap {
		services = append(services, service)
	}
	return services, nil
}

type composeIRTranslator interface {
	ConvertToIR(filepath string, plan plantypes.Plan, service plantypes.Service) (irtypes.IR, error)
}

// Translate translates the service to IR
func (c ComposeTranslator) Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(p)

	for _, service := range services {
		if service.TranslationType != c.GetTranslatorType() {
			continue
		}
		for _, fullpath := range service.SourceArtifacts[plantypes.ComposeFileArtifactType] {
			log.Debugf("File %s being loaded from compose service : %s", fullpath, service.ServiceName)
			var dcfile sourcetypes.DockerCompose
			err := common.ReadYaml(fullpath, &dcfile)
			if err != nil {
				log.Errorf("Unable to read docker compose yaml %s for version : %s", fullpath, err)
			}
			log.Debugf("Docker Compose version: %s", dcfile.Version)
			var t composeIRTranslator
			switch dcfile.Version {
			case "", "1", "1.0", "2", "2.0", "2.1":
				t = new(compose.V1V2Loader)
			case "3", "3.0", "3.1", "3.2", "3.3", "3.4", "3.5", "3.6", "3.7", "3.8":
				t = new(compose.V3Loader)
			default:
				log.Errorf("Version %s of Docker Compose is not supported (%s). Please use version 1, 2 or 3.", dcfile.Version, fullpath)
			}
			cir, err := t.ConvertToIR(fullpath, p, service)
			if err != nil {
				log.Errorf("Unable to parse docker compose file %s using %T : %s", fullpath, t, err)
			} else {
				ir.Merge(cir)
			}
			log.Debugf("Services returned by compose translator : %d", len(ir.Services))
		}
		for _, path := range service.SourceArtifacts[plantypes.ImageInfoArtifactType] {
			imgMD := collecttypes.ImageInfo{}
			err := common.ReadYaml(path, &imgMD)
			if err != nil {
				log.Errorf("Unable to read image yaml %s : %s", path, err)
			} else {
				ir.AddContainer(irtypes.NewContainerFromImageInfo(imgMD))
			}
		}
	}

	return ir, nil
}

func (c ComposeTranslator) newService(serviceName string) plantypes.Service {
	service := plantypes.NewService(serviceName, c.GetTranslatorType())
	service.AddSourceType(plantypes.ComposeSourceTypeValue)
	service.ContainerBuildType = plantypes.ReuseContainerBuildTypeValue //TODO: Identify when to use enhance
	return service
}
