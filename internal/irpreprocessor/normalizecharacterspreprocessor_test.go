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

package irpreprocessor

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
	core "k8s.io/kubernetes/pkg/apis/core"
)

func TestStripQuotation(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("strip matching single quotation marks from input string", func(t *testing.T) {
		// Setup
		inputString := "'testString'"
		want := "testString"

		// Test
		actual := stripQuotation(inputString)
		if actual != want {
			t.Fatalf("Failed to get the expected properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("strip matching double quotation marks from input string", func(t *testing.T) {
		// Setup
		inputString := "\"testString\""
		want := "testString"

		// Test
		actual := stripQuotation(inputString)
		if actual != want {
			t.Fatalf("Failed to get the expected properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("expect unmodified string for input string without any quotation", func(t *testing.T) {
		// Setup
		inputString := "testString"
		want := "testString"

		// Test
		actual := stripQuotation(inputString)
		if actual != want {
			t.Fatalf("Failed to get the expected properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
}

func TestOptimize(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("IR with no services", func(t *testing.T) {
		// Setup
		ir := getIRWithoutServices()
		normalizeCharacterPreprocessor := normalizeCharacterPreprocessor{}
		want := getIRWithoutServices()

		// Test
		actual, err := normalizeCharacterPreprocessor.preprocess(ir)
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
		normalizeCharacterPreprocessor := normalizeCharacterPreprocessor{}
		want := getIRWithServicesAndWithoutContainers()

		// Test
		actual, err := normalizeCharacterPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR containing services and containers but the containers have no environment variables", func(t *testing.T) {
		// Setup
		ir := getIRWithServicesAndContainersWithoutEnv()
		normalizeCharacterPreprocessor := normalizeCharacterPreprocessor{}
		want := getIRWithServicesAndContainersWithoutEnv()

		// Test
		actual, err := normalizeCharacterPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("An IR containing services and containers and all the environment variables are valid", func(t *testing.T) {
		// Setup
		ir := getIRWithServicesAndContainersWithValidEnv()
		normalizeCharacterPreprocessor := normalizeCharacterPreprocessor{}
		want := getIRWithServicesAndContainersWithValidEnv()

		// Test
		actual, err := normalizeCharacterPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("An IR containing services and containers and some of the environment variables are invalid", func(t *testing.T) {
		// Setup
		c1 := core.Container{
			Name: "container-1",
			Env: []core.EnvVar{
				{Name: "NAME\t", Value: "git-resource"},
				{Name: "NO_PROXY", Value: "'no-proxy.git.com'"},
				{Name: "VALID_VARIABLE", Value: "valid-variable"},
			},
		}
		c2 := core.Container{
			Name: "container-2",
			Env: []core.EnvVar{
				{Name: "\nNAME", Value: "git-resource2"},
				{Name: " PROXY", Value: "  proxy.git.com "},
			},
		}
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
		svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
		svc1.Containers = append(svc1.Containers, c1)
		svc2.Containers = append(svc2.Containers, c2)
		ir := irtypes.NewIR("")
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2

		normalizeCharacterPreprocessor := normalizeCharacterPreprocessor{}
		want := getExpectedIR()

		// Test
		actual, err := normalizeCharacterPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR containing services and containers and some of the environment variables are invalid but their names contain the string affinity", func(t *testing.T) {
		// Setup
		c1 := core.Container{
			Name: "container-1",
			Env: []core.EnvVar{
				{Name: "NAME\t", Value: "git-resource"},
				{Name: "affinity", Value: "with-pod-affinity "},
			},
		}
		c2 := core.Container{
			Name: "container-2",
			Env: []core.EnvVar{
				{Name: "\nNAME", Value: "git-resource2"},
				{Name: " PROXY", Value: "  proxy.git.com "},
			},
		}
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
		svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
		svc1.Containers = append(svc1.Containers, c1)
		svc2.Containers = append(svc2.Containers, c2)
		ir := irtypes.NewIR("")
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2

		normalizeCharacterPreprocessor := normalizeCharacterPreprocessor{}
		want := getExpectedIRWithAffinityInContainer()

		// Test
		actual, err := normalizeCharacterPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
}

func getIRWithServicesAndContainersWithValidEnv() irtypes.IR {
	c1 := core.Container{
		Name: "container-1",
		Env: []core.EnvVar{
			{Name: "NAME", Value: "git-resource"},
			{Name: "NO_PROXY", Value: "no-proxy.git.com"},
		},
	}
	c2 := core.Container{
		Name: "container-2",
		Env: []core.EnvVar{
			{Name: "NAME", Value: "git-resource2"},
			{Name: "PROXY", Value: "proxy.git.com"},
		},
	}
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
	svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)

	ir := irtypes.NewIR("")
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2
	return ir
}

func getIRWithServicesAndContainersWithoutEnv() irtypes.IR {
	c1 := core.Container{
		Name: "container-1",
	}
	c2 := core.Container{
		Name: "container-2",
	}
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
	svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)

	ir := irtypes.NewIR("")
	ir.Services[svcname1] = svc1
	ir.Services[svcname1] = svc2
	return ir
}

func getExpectedIR() irtypes.IR {
	c1 := core.Container{
		Name: "container-1",
		Env: []core.EnvVar{
			{Name: "NAME", Value: "git-resource"},
			{Name: "NO_PROXY", Value: "no-proxy.git.com"},
			{Name: "VALID_VARIABLE", Value: "valid-variable"},
		},
	}
	c2 := core.Container{
		Name: "container-2",
		Env: []core.EnvVar{
			{Name: "NAME", Value: "git-resource2"},
			{Name: "PROXY", Value: "proxy.git.com"},
		},
	}
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
	svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)
	ir := irtypes.NewIR("")
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2
	return ir
}

func getExpectedIRWithAffinityInContainer() irtypes.IR {
	c1 := core.Container{
		Name: "container-1",
		Env: []core.EnvVar{
			{Name: "NAME", Value: "git-resource"},
		},
	}
	c2 := core.Container{
		Name: "container-2",
		Env: []core.EnvVar{
			{Name: "NAME", Value: "git-resource2"},
			{Name: "PROXY", Value: "proxy.git.com"},
		},
	}
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
	svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
	svc1.Containers = append(svc1.Containers, c1)
	svc2.Containers = append(svc2.Containers, c2)
	ir := irtypes.NewIR("")
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2
	return ir
}
