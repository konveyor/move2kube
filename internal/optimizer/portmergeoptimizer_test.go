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

package optimize

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

func TestPortMergeOptimizer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("IR with no services", func(t *testing.T) {
		// Setup
		ir := getIRWithoutServices()
		portMergeOptimizer := portMergeOptimizer{}
		want := getIRWithoutServices()

		// Test
		actual, err := portMergeOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR containing services that have no containers", func(t *testing.T) {
		// Setup
		ir := getIRWithServicesAndWithoutContainers()
		portMergeOptimizer := portMergeOptimizer{}
		want := getIRWithServicesAndWithoutContainers()

		// Test
		actual, err := portMergeOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR containing services and containers without image", func(t *testing.T) {
		// Setup
		ir := getIRWithServicesAndContainersWithoutImage()
		portMergeOptimizer := portMergeOptimizer{}
		want := getIRWithServicesAndContainersWithoutImage()

		// Test
		actual, err := portMergeOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR without Containers and Service containing containers with image", func(t *testing.T) {
		// Setup
		ir := getIRWithoutContainersAndServiceContainingContainersWithImage()
		portMergeOptimizer := portMergeOptimizer{}
		want := getIRWithoutContainersAndServiceContainingContainersWithImage()

		// Test
		actual, err := portMergeOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR containing Containers and Service containing containers with image name", func(t *testing.T) {
		// Setup
		c1 := corev1.Container{
			Name: "container-1",
		}
		c1.Image = "image1"
		c2 := corev1.Container{
			Name: "container-2",
		}
		c2.Image = "image2"
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := types.Service{Name: svcname1, Replicas: 2}
		svc2 := types.Service{Name: svcname2, Replicas: 4}
		svc1.Containers = append(svc1.Containers, c1)
		svc2.Containers = append(svc2.Containers, c2)

		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2

		image1 := "image1"
		image2 := "image2"
		cont1 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image1, true)
		cont2 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image2, true)
		cont1.ExposedPorts = append(cont1.ExposedPorts, 8088)
		cont2.ExposedPorts = append(cont2.ExposedPorts, 8000)
		ir.Containers = append(ir.Containers, cont1, cont2)

		portMergeOptimizer := portMergeOptimizer{}
		want := getExpectIRWithServiceContainingContainerPortsGivenImageName()

		// Test
		actual, err := portMergeOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR containing Containers and Services containing containers with image url", func(t *testing.T) {
		// Setup
		c1 := corev1.Container{
			Name: "container-1",
		}
		c1.Image = "registry.com/namespace/image1"
		c2 := corev1.Container{
			Name: "container-2",
		}
		c2.Image = "registry.com/namespace/image2"
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := types.Service{Name: svcname1, Replicas: 2}
		svc2 := types.Service{Name: svcname2, Replicas: 4}
		svc1.Containers = append(svc1.Containers, c1)
		svc2.Containers = append(svc2.Containers, c2)

		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2

		image1 := "image1"
		image2 := "image2"
		registry := "registry.com"
		cont1 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image1, true)
		cont2 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image2, true)
		cont1.ExposedPorts = append(cont1.ExposedPorts, 8088)
		cont2.ExposedPorts = append(cont2.ExposedPorts, 8000, 8080)
		ir.Containers = append(ir.Containers, cont1, cont2)
		ir.Kubernetes.RegistryURL = registry

		portMergeOptimizer := portMergeOptimizer{}
		want := getExpectIRWithServiceContainingContainerPortsGivenImageURL()

		// Test
		actual, err := portMergeOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR containing Containers and Services containing containers with image name and image url", func(t *testing.T) {
		// Setup
		c1 := corev1.Container{
			Name: "container-1",
		}
		c1.Image = "image1"
		c2 := corev1.Container{
			Name: "container-2",
		}
		c2.Image = "registry.com/namespace/image2"
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := types.Service{Name: svcname1, Replicas: 2}
		svc2 := types.Service{Name: svcname2, Replicas: 4}
		svc1.Containers = append(svc1.Containers, c1)
		svc2.Containers = append(svc2.Containers, c2)

		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2

		image1 := "image1"
		image2 := "image2"
		registry := "registry.com"
		cont1 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image1, true)
		cont2 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image2, true)
		cont1.ExposedPorts = append(cont1.ExposedPorts, 8088)
		cont2.ExposedPorts = append(cont2.ExposedPorts, 8000, 8080)
		ir.Containers = append(ir.Containers, cont1, cont2)
		ir.Kubernetes.RegistryURL = registry

		portMergeOptimizer := portMergeOptimizer{}
		want := getExpectIRWithServiceContainingContainerPortsGivenImageNameAndURL()

		// Test
		actual, err := portMergeOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
}

func getIRWithoutContainersAndServiceContainingContainersWithImage() types.IR {
	c1 := corev1.Container{
		Name: "container-1",
	}
	c1.Image = "image1"
	c2 := corev1.Container{
		Name: "container-2",
	}
	c2.Image = "image2"
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc2 := types.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)

	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2
	return ir
}

func getIRWithServicesAndContainersWithoutImage() types.IR {
	c1 := corev1.Container{
		Name: "container-1",
	}
	c2 := corev1.Container{
		Name: "container-2",
	}
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc2 := types.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)

	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2
	return ir
}

func getExpectIRWithServiceContainingContainerPortsGivenImageName() types.IR {
	c1 := corev1.Container{
		Name: "container-1",
	}
	c1.Image = "image1"
	c1.Ports = append(c1.Ports, corev1.ContainerPort{ContainerPort: 8088})
	c2 := corev1.Container{
		Name: "container-2",
	}
	c2.Image = "image2"
	c2.Ports = append(c2.Ports, corev1.ContainerPort{ContainerPort: 8000})
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc2 := types.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)

	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2

	image1 := "image1"
	image2 := "image2"
	cont1 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image1, true)
	cont2 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image2, true)
	cont1.ExposedPorts = append(cont1.ExposedPorts, 8088)
	cont2.ExposedPorts = append(cont2.ExposedPorts, 8000)
	ir.Containers = append(ir.Containers, cont1, cont2)

	return ir
}

func getExpectIRWithServiceContainingContainerPortsGivenImageURL() types.IR {
	c1 := corev1.Container{
		Name: "container-1",
	}
	c1.Image = "registry.com/namespace/image1"
	c1.Ports = append(c1.Ports, corev1.ContainerPort{ContainerPort: 8088})
	c2 := corev1.Container{
		Name: "container-2",
	}
	c2.Image = "registry.com/namespace/image2"
	c2.Ports = append(c2.Ports, corev1.ContainerPort{ContainerPort: 8000}, corev1.ContainerPort{ContainerPort: 8080})
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc2 := types.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)

	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2

	image1 := "image1"
	image2 := "image2"
	registry := "registry.com"
	cont1 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image1, true)
	cont2 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image2, true)
	cont1.ExposedPorts = append(cont1.ExposedPorts, 8088)
	cont2.ExposedPorts = append(cont2.ExposedPorts, 8000, 8080)
	ir.Containers = append(ir.Containers, cont1, cont2)
	ir.Kubernetes.RegistryURL = registry

	return ir
}

func getExpectIRWithServiceContainingContainerPortsGivenImageNameAndURL() types.IR {
	c1 := corev1.Container{
		Name: "container-1",
	}
	c1.Image = "image1"
	c1.Ports = append(c1.Ports, corev1.ContainerPort{ContainerPort: 8088})
	c2 := corev1.Container{
		Name: "container-2",
	}
	c2.Image = "registry.com/namespace/image2"
	c2.Ports = append(c2.Ports, corev1.ContainerPort{ContainerPort: 8000}, corev1.ContainerPort{ContainerPort: 8080})
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc2 := types.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)

	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2

	image1 := "image1"
	image2 := "image2"
	registry := "registry.com"
	cont1 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image1, true)
	cont2 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, image2, true)
	cont1.ExposedPorts = append(cont1.ExposedPorts, 8088)
	cont2.ExposedPorts = append(cont2.ExposedPorts, 8000, 8080)
	ir.Containers = append(ir.Containers, cont1, cont2)
	ir.Kubernetes.RegistryURL = registry

	return ir
}
