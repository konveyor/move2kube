/*
 *  Copyright IBM Corporation 2020, 2021
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

package ir

import (
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/common"
	"gopkg.in/yaml.v3"
	core "k8s.io/kubernetes/pkg/apis/core"
	networking "k8s.io/kubernetes/pkg/apis/networking"
)

func TestIR(t *testing.T) {
	t.Run("test for MarshalYAML for podspec", func(t *testing.T) {
		podspec := PodSpec{
			Containers: []core.Container{
				{
					Name:  "nginx",
					Image: "nginx",
				},
			},
		}
		// take a yaml and then do marshal and then check here
		podspecyaml, err := podspec.MarshalYAML()
		if err != nil {
			t.Errorf("failed to marshal podspec yaml : %v. Error : %v", podspecyaml, err)
		}
	})

	t.Run("test for MarshalYAML for Service", func(t *testing.T) {
		s := Service{
			PodSpec: PodSpec{
				Containers: []core.Container{
					{
						Name:  "nginx",
						Image: "nginx",
					},
				},
			},
			Name:               "my-service",
			BackendServiceName: "backend-service",
			Annotations: map[string]string{
				"foo": "bar",
			},
			Labels: map[string]string{
				"app": "my-app",
			},
			Replicas: 2,
			Networks: []string{"frontend", "backend"},
		}
		serviceyaml, err := s.MarshalYAML()
		if err != nil {
			t.Errorf("failed to marshal service yaml : %v. Error : %v", serviceyaml, err)
		}
	})

	t.Run("test for new IR, service with name, and container", func(t *testing.T) {
		ir := NewIR()
		if reflect.TypeOf(ir) != reflect.TypeOf(IR{}) {
			t.Errorf("failed to get new IR struct.")
		}

		serviceName := "sampleService"

		service := NewServiceWithName(serviceName)
		if reflect.TypeOf(service) != reflect.TypeOf(Service{}) {
			t.Error("failed to get new Service struct.")
		}
		if service.Name != serviceName {
			t.Errorf("failed to get service with the given service name. need %s, got %s", serviceName, service.Name)
		}

		container := NewContainer()
		if reflect.TypeOf(container) != reflect.TypeOf(ContainerImage{}) {
			t.Error("failed to get new container struct.")
		}

		enhancedIR := NewEnhancedIRFromIR(ir)
		if reflect.TypeOf(enhancedIR) != reflect.TypeOf(EnhancedIR{}) {
			t.Error("failed to get new enhanced IR.")
		}
		if !reflect.DeepEqual(ir, enhancedIR.IR) {
			t.Error("failed to get new enhanced IR with the provided IR.")
		}
	})

	t.Run("test for adding service to IR", func(t *testing.T) {
		serviceName := "sampleService"
		service := NewServiceWithName(serviceName)
		ir := NewIR()
		ir.AddService(service)

		_, ok := ir.Services[service.Name]
		if !ok {
			t.Error("failed to add service to the IR")
		}
	})

	t.Run("test for adding container to IR", func(t *testing.T) {
		container := NewContainer()
		imageName := "testImage"
		ir := NewIR()
		ir.AddContainer(imageName, container)

		_, ok := ir.ContainerImages[imageName]
		if !ok {
			t.Error("failed to add container to the IR")
		}
	})

	t.Run("test for adding storage to IR", func(t *testing.T) {
		storage := Storage{Name: "testStorage", StorageType: "pvc"}
		ir := NewIR()
		ir.AddStorage(storage)
		found := false
		for i := range ir.Storages {
			if reflect.DeepEqual(ir.Storages[i], storage) {
				found = true
			}
		}
		if !found {
			t.Errorf("failed to add storage to IR")
		}
	})

	t.Run("test for adding volume to service", func(t *testing.T) {
		serviceName := "sampleSevice"
		volumeName := "sampleVolume"
		service := NewServiceWithName(serviceName)
		volume := core.Volume{Name: volumeName}
		service.AddVolume(volume)
		found := false
		for i := range service.Volumes {
			if reflect.DeepEqual(service.Volumes[i], volume) {
				found = true
			}
		}
		if !found {
			t.Error("failed to add volume to service")
		}
	})

	t.Run("test for adding port forwardings to service", func(t *testing.T) {
		serviceName := "sampleSevice"
		servicePort := networking.ServiceBackendPort{Name: "servicePort", Number: 8080}
		podPort := networking.ServiceBackendPort{Name: "podPort", Number: 8080}
		serviceRelPath := "service"
		serviceToPodPortForwarding := ServiceToPodPortForwarding{ServicePort: servicePort, PodPort: podPort, ServiceRelPath: serviceRelPath}
		service := NewServiceWithName(serviceName)
		service.AddPortForwarding(servicePort, podPort, serviceRelPath)
		found := false
		for i := range service.ServiceToPodPortForwardings {
			if reflect.DeepEqual(service.ServiceToPodPortForwardings[i], serviceToPodPortForwarding) {
				found = true
			}
		}
		if !found {
			t.Error("failed to add port forwarding to service")
		}
	})

	t.Run("test for adding exposed port to container", func(t *testing.T) {
		container := NewContainer()
		var exposedPort int32 = 8080
		container.AddExposedPort(exposedPort)
		found := false

		for i := range container.ExposedPorts {
			if container.ExposedPorts[i] == exposedPort {
				found = true
			}
		}
		if !found {
			t.Error("failed to add exposed port to container")
		}
	})

	t.Run("test for adding accessed dirs to container", func(t *testing.T) {
		container := NewContainer()
		var accessedDir string = "dir1"
		container.AddAccessedDirs(accessedDir)
		found := false

		for i := range container.AccessedDirs {
			if container.AccessedDirs[i] == accessedDir {
				found = true
			}
		}
		if !found {
			t.Error("failed to add accessed dir to container")
		}
	})

	t.Run("test for having valid annotation", func(t *testing.T) {
		service := NewServiceWithName("sampleService")
		service.Annotations = map[string]string{common.WindowsAnnotation: common.AnnotationLabelValue}
		if !service.HasValidAnnotation(common.WindowsAnnotation) {
			t.Error("failed to validate annotations of the service")
		}
	})

	t.Run("test for getting all service ports", func(t *testing.T) {
		ir := NewIR()
		var port int32 = 8080
		service := Service{Name: "sampleService", ServiceToPodPortForwardings: []ServiceToPodPortForwarding{{PodPort: networking.ServiceBackendPort{Name: "somePort", Number: port}}}}
		ir.AddService(service)
		ports := ir.GetAllServicePorts()
		found := false
		for i := range ports {
			if ports[i] == port {
				found = true
			}
		}
		if !found {
			t.Error("failed to get service ports")
		}
	})

	t.Run("test for merging storages", func(t *testing.T) {
		oldStorage := Storage{Name: "sampleStorage", StorageType: "pvc"}
		newStorage := Storage{Name: "sampleStorage", StorageType: "cfmap"}
		oldStorage.Merge(newStorage)
		if oldStorage.Name != newStorage.Name || oldStorage.StorageType != newStorage.StorageType {
			t.Errorf("failed to merge new storage into old storage. Old Storage : %+v, New Storage: %+v", oldStorage, newStorage)
		}
	})

	t.Run("test for merging IRs", func(t *testing.T) {

		iryaml, err := os.ReadFile("./testdata/ir.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var ir IR
		err = yaml.Unmarshal(iryaml, &ir)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		newiryaml, err := os.ReadFile("./testdata/newir.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var newir IR
		err = yaml.Unmarshal(newiryaml, &newir)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		mergediryaml, err := os.ReadFile("./testdata/mergedir.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var mergedir IR
		err = yaml.Unmarshal(mergediryaml, &mergedir)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		ir.Merge(newir)

		v, ok := mergedir.Services[""]
		if ok {
			// Modifying the desired field in the struct value
			v.SecurityContext = &core.PodSecurityContext{}

			// Assigning the modified struct value back to the map
			mergedir.Services[""] = v
		}

		if diff := cmp.Diff(mergedir, ir); diff != "" {
			t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
			t.Logf("\nMergedIr= %+v", mergedir.Services[""])
		}

	})

	t.Run("test for merging containers images", func(t *testing.T) {
		conimageyaml, err := os.ReadFile("./testdata/conimage.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var conimage ContainerImage
		err = yaml.Unmarshal(conimageyaml, &conimage)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		newconimageyaml, err := os.ReadFile("./testdata/newconimage.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var newconimage ContainerImage
		err = yaml.Unmarshal(newconimageyaml, &newconimage)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		conimage.Merge(newconimage)

		mergedconimageyaml, err := os.ReadFile("./testdata/mergedconimage.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var mergedconimage ContainerImage
		err = yaml.Unmarshal(mergedconimageyaml, &mergedconimage)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		if diff := cmp.Diff(mergedconimage, conimage); diff != "" {
			t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
		}

	})

	t.Run("test for merging containerbuilds", func(t *testing.T) {
		conbuildyaml, err := os.ReadFile("./testdata/conbuild.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var conbuild ContainerBuild
		err = yaml.Unmarshal(conbuildyaml, &conbuild)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		newconbuildyaml, err := os.ReadFile("./testdata/newconbuild.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var newconbuild ContainerBuild
		err = yaml.Unmarshal(newconbuildyaml, &newconbuild)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		conbuild.Merge(newconbuild)

		mergedconbuildyaml, err := os.ReadFile("./testdata/mergedconbuild.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		// Unmarshal YAML into struct
		var mergedconbuild ContainerBuild
		err = yaml.Unmarshal(mergedconbuildyaml, &mergedconbuild)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		if diff := cmp.Diff(mergedconbuild, conbuild); diff != "" {
			t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
		}

	})

	t.Run("test for merging services", func(t *testing.T) {
		serviceyaml, err := os.ReadFile("./testdata/service.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		var service Service
		err = yaml.Unmarshal(serviceyaml, &service)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}

		newserviceyaml, err := os.ReadFile("./testdata/nservice.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		var newservice Service
		err = yaml.Unmarshal(newserviceyaml, &newservice)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}
		service.merge(newservice)

		mergedserviceyaml, err := os.ReadFile("./testdata/merged_service.yaml")
		if err != nil {
			fmt.Println("Failed to read YAML file:", err)
			return
		}

		var mergedservice Service
		err = yaml.Unmarshal(mergedserviceyaml, &mergedservice)
		if err != nil {
			fmt.Println("Failed to unmarshal YAML:", err)
			return
		}
		mergedservice.SecurityContext = &core.PodSecurityContext{}

		if diff := cmp.Diff(mergedservice, service); diff != "" {
			t.Errorf("MakeGatewayInfo() mismatch (-want +got):\n%s", diff)
		}

	})

}
