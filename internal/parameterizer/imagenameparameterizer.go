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

package parameterize

import (
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	outputtypes "github.com/konveyor/move2kube/types/output"
)

// imageNameParameterizer parameterizes the image names
type imageNameParameterizer struct {
}

func (it imageNameParameterizer) parameterize(ir *irtypes.IR) error {
	newimages := []string{}
	for _, container := range ir.Containers {
		if container.New {
			for _, img := range container.ImageNames {
				newimages = append(newimages, ir.RegistryURL+"/"+ir.RegistryNamespace+"/"+img)
			}
		}
	}

	ir.Values.Services = map[string]outputtypes.Service{}
	for _, service := range ir.Services {
		ir.Values.Services[service.Name] = outputtypes.Service{
			Containers: map[string]outputtypes.Container{},
		}
		for ci, serviceContainer := range service.Containers {
			nImageName := ""
			parts := strings.Split(serviceContainer.Image, "/")
			if len(parts) == 3 {
				nImageName += parts[0] + "/"
			}
			if len(parts) > 1 {
				nImageName += parts[1] + "/"
			}
			if common.IsStringPresent(newimages, serviceContainer.Image) {
				nImageName = outputtypes.ParameterRegistryPrefix
			}
			imageName := parts[len(parts)-1]
			im, tag := common.GetImageNameAndTag(imageName)
			ir.Values.Services[service.Name].Containers[serviceContainer.Name] = outputtypes.Container{TagName: tag}
			newTag := "{{ index .Values." + outputtypes.ServicesTag + " \"" + service.Name + "\" \"" + outputtypes.ContainersTag + "\" \"" + serviceContainer.Name + "\" \"" + outputtypes.ImageTagTag + "\"  }}"
			nImageName += im + ":" + newTag
			serviceContainer.Image = nImageName
			service.Containers[ci] = serviceContainer
		}
		ir.Services[service.Name] = service
	}
	return nil
}
