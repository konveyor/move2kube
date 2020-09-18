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

package output

const (
	// ParameterRegistryPrefix defines the parameter registry
	ParameterRegistryPrefix string = "{{.Values.registryurl}}/{{.Values.registrynamespace}}/"
	// ServicesTag is the tag name for services
	ServicesTag string = "services"
	// ImageTagTag is the tag name for images
	ImageTagTag string = "imagetag"
	// ContainersTag is the tag name for conatiners
	ContainersTag string = "containers"
)

// HelmValues defines the format of values.yaml
type HelmValues struct {
	RegistryURL       string             `yaml:"registryurl"`
	RegistryNamespace string             `yaml:"registrynamespace"`
	Services          map[string]Service `yaml:"services"`
	StorageClass      string             `yaml:"storageclass,omitempty"`
	GlobalVariables   map[string]string  `yaml:"globalvariables,omitempty"`
}

// Merge helps merge helmvalues
func (h *HelmValues) Merge(newh HelmValues) {
	if newh.RegistryNamespace != "" {
		h.RegistryNamespace = newh.RegistryNamespace
	}
	if newh.RegistryURL != "" {
		h.RegistryURL = newh.RegistryURL
	}
	if newh.StorageClass != "" {
		h.StorageClass = newh.StorageClass
	}
	for gvkey, gvval := range newh.GlobalVariables {
		h.GlobalVariables[gvkey] = gvval
	}
	for serviceName, service := range newh.Services {
		if _, ok := h.Services[serviceName]; !ok {
			h.Services[serviceName] = service
		} else {
			for ncn, nc := range service.Containers {
				if c, ok := h.Services[serviceName].Containers[ncn]; !ok {
					h.Services[serviceName].Containers[ncn] = nc
				} else {
					c.TagName = nc.TagName
					h.Services[serviceName].Containers[ncn] = c
				}
			}
		}
	}
}

// Service stores the metadata about the services and its containers
type Service struct {
	Containers map[string]Container `yaml:"containers"`
}

// Container stores the metadata the container
type Container struct {
	TagName string `yaml:"imagetag"`
}
