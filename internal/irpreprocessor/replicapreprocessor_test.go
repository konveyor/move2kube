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
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
)

func TestReplicaPreprocessor(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("IR with no services", func(t *testing.T) {
		// Setup
		ir := getIRWithoutServices()
		replicaPreprocessor := replicaPreprocessor{}
		want := getIRWithoutServices()

		// Test
		actual, err := replicaPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR with services with exact default minimum replicas", func(t *testing.T) {
		// Setup
		ir := getIRWithServicesWithDefaultMinimumReplicas()
		replicaPreprocessor := replicaPreprocessor{}
		want := getIRWithServicesWithDefaultMinimumReplicas()

		// Test
		actual, err := replicaPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR with services with less replicas than default minimum replicas", func(t *testing.T) {
		// Setup
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := irtypes.Service{Name: svcname1, Replicas: 1}
		svc2 := irtypes.Service{Name: svcname2, Replicas: 1}

		p := plantypes.NewPlan()
		ir := irtypes.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2
		replicaPreprocessor := replicaPreprocessor{}
		want := getIRWithServicesWithDefaultMinimumReplicas()

		// Test
		actual, err := replicaPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR with services with more replicas than default minimum replicas", func(t *testing.T) {
		// Setup
		ir := getServicesWithMoreReplicasThanDefaultMinimumReplicas()
		replicaPreprocessor := replicaPreprocessor{}
		want := getServicesWithMoreReplicasThanDefaultMinimumReplicas()

		// Test
		actual, err := replicaPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR with services with less and more replicas respectively than default minimum replicas", func(t *testing.T) {
		// Setup
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := irtypes.Service{Name: svcname1, Replicas: 1}
		svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
		p := plantypes.NewPlan()
		ir := irtypes.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2
		replicaPreprocessor := replicaPreprocessor{}
		want := getExpectedIRWithModifiedReplicas()

		// Test
		actual, err := replicaPreprocessor.preprocess(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
}

func getServicesWithMoreReplicasThanDefaultMinimumReplicas() irtypes.IR {
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := irtypes.Service{Name: svcname1, Replicas: 4}
	svc2 := irtypes.Service{Name: svcname2, Replicas: 3}
	p := plantypes.NewPlan()
	ir := irtypes.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2
	return ir
}

func getIRWithServicesWithDefaultMinimumReplicas() irtypes.IR {
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
	svc2 := irtypes.Service{Name: svcname2, Replicas: 2}
	p := plantypes.NewPlan()
	ir := irtypes.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2
	return ir
}

func getExpectedIRWithModifiedReplicas() irtypes.IR {
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := irtypes.Service{Name: svcname1, Replicas: 2}
	svc2 := irtypes.Service{Name: svcname2, Replicas: 4}
	p := plantypes.NewPlan()
	ir := irtypes.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2

	return ir
}
