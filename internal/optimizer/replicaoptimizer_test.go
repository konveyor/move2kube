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
	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestReplicaOptimizer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("IR with no services", func(t *testing.T) {
		// Setup
		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		replicaOptimizer := replicaOptimizer{}
		want := ir

		// Test
		actual, err := replicaOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR with services with exact default minimum replicas", func(t *testing.T) {
		// Setup
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := types.Service{Name: svcname1, Replicas: 2}
		svc2 := types.Service{Name: svcname2, Replicas: 2}

		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2
		replicaOptimizer := replicaOptimizer{}
		want := ir

		// Test
		actual, err := replicaOptimizer.optimize(ir)
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
		svc1 := types.Service{Name: svcname1, Replicas: 1}
		svc2 := types.Service{Name: svcname2, Replicas: 1}

		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2
		replicaOptimizer := replicaOptimizer{}
		want := getExpectedIRWithDefaultMinimumReplicas()

		// Test
		actual, err := replicaOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})

	t.Run("IR with services with more replicas than default minimum replicas", func(t *testing.T) {
		// Setup
		svcname1 := "svcname1"
		svcname2 := "svcname2"
		svc1 := types.Service{Name: svcname1, Replicas: 4}
		svc2 := types.Service{Name: svcname2, Replicas: 3}

		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2
		replicaOptimizer := replicaOptimizer{}
		want := ir

		// Test
		actual, err := replicaOptimizer.optimize(ir)
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
		svc1 := types.Service{Name: svcname1, Replicas: 1}
		svc2 := types.Service{Name: svcname2, Replicas: 4}

		p := plantypes.NewPlan()
		ir := types.NewIR(p)
		ir.Services[svcname1] = svc1
		ir.Services[svcname2] = svc2
		replicaOptimizer := replicaOptimizer{}
		want := getExpectedIRWithModifiedReplicas()

		// Test
		actual, err := replicaOptimizer.optimize(ir)
		if err != nil {
			t.Fatal("Failed to get the expected. Error:", err)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
}

func getExpectedIRWithDefaultMinimumReplicas() types.IR {
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc2 := types.Service{Name: svcname2, Replicas: 2}
	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2

	return ir
}

func getExpectedIRWithModifiedReplicas() types.IR {
	svcname1 := "svcname1"
	svcname2 := "svcname2"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc2 := types.Service{Name: svcname2, Replicas: 4}
	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1
	ir.Services[svcname2] = svc2

	return ir
}
