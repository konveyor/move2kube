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

package plan_test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/types/plan"
)

func TestMerge(t *testing.T) {
	t.Run("merge new empty k8s output into empty k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{}
		out2 := plan.KubernetesOutput{}
		want := plan.KubernetesOutput{}
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("The output should not have changed. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge ignore supported kinds from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", plan.TargetClusterType{Type: "444"}, false}
		out2 := plan.KubernetesOutput{IgnoreUnsupportedKinds: true}
		want := out1
		want.IgnoreUnsupportedKinds = true
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge registry url from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", plan.TargetClusterType{Type: "444"}, false}
		out2 := plan.KubernetesOutput{IgnoreUnsupportedKinds: true, RegistryURL: "url1"}
		want := out1
		want.IgnoreUnsupportedKinds = true
		want.RegistryURL = "url1"
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge registry namespace from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", plan.TargetClusterType{Type: "444"}, false}
		out2 := plan.KubernetesOutput{IgnoreUnsupportedKinds: true, RegistryNamespace: "namespace1"}
		want := out1
		want.IgnoreUnsupportedKinds = true
		want.RegistryNamespace = "namespace1"
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge image pull secret from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", plan.TargetClusterType{Type: "444"}, false}
		out2 := plan.KubernetesOutput{IgnoreUnsupportedKinds: true}
		want := out1
		want.IgnoreUnsupportedKinds = true
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge cluster type from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", plan.TargetClusterType{Type: "444"}, false}
		out2 := plan.KubernetesOutput{IgnoreUnsupportedKinds: true, TargetCluster: plan.TargetClusterType{Type: "clus_type1"}}
		want := out1
		want.IgnoreUnsupportedKinds = true
		want.TargetCluster = plan.TargetClusterType{Type: "clus_type1"}
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
}

func TestAddSourceArtifact(t *testing.T) {
	// Setup
	s := plan.NewService("foo", "bar")
	var key1 plan.SourceArtifactTypeValue = "key1"
	val1 := "val1"
	val2 := "val2"
	// Test
	t.Run("add source artifact to empty service", func(t *testing.T) {
		s.AddSourceArtifact(key1, val1)
		if arr, ok := s.SourceArtifacts[key1]; !ok || len(arr) != 1 || arr[0] != val1 {
			t.Fatal("Failed to add source artifact type properly. Expected:", []string{val1}, "Actual:", arr)
		}
	})
	// Setup
	s = plan.NewService("foo", "bar")
	// Test
	t.Run("add source artifact to filled service", func(t *testing.T) {
		s.AddSourceArtifact(key1, val1)
		s.AddSourceArtifact(key1, val2)
		if arr, ok := s.SourceArtifacts[key1]; !ok || len(arr) != 2 || arr[1] != val2 {
			t.Fatal("Failed to add source artifact type properly when array is not empty. Expected:", []string{val1, val2}, "Actual:", arr)
		}
	})
}

func TestAddBuildArtifact(t *testing.T) {
	// Setup
	s := plan.NewService("foo", "bar")
	var key1 plan.BuildArtifactTypeValue = "key1"
	val1 := "val1"
	val2 := "val2"
	// Test
	t.Run("add build artifact to empty service", func(t *testing.T) {
		s.AddBuildArtifact(key1, val1)
		if arr, ok := s.BuildArtifacts[key1]; !ok || len(arr) != 1 || arr[0] != val1 {
			t.Fatal("Failed to add build artifact type properly. Expected:", []string{val1}, "Actual:", arr)
		}
	})
	// Setup
	s = plan.NewService("foo", "bar")
	// Test
	t.Run("add build artifact to filled service", func(t *testing.T) {
		s.AddBuildArtifact(key1, val1)
		s.AddBuildArtifact(key1, val2)
		if arr, ok := s.BuildArtifacts[key1]; !ok || len(arr) != 2 || arr[1] != val2 {
			t.Fatal("Failed to add build artifact type properly when array is not empty. Expected:", []string{val1, val2}, "Actual:", arr)
		}
	})
}

func TestAddSourceType(t *testing.T) {
	// Setup
	var src1 plan.SourceTypeValue = "src1"
	var src2 plan.SourceTypeValue = "src2"
	svc0 := plan.NewService("foo", "bar")
	svc1 := plan.NewService("foo", "bar")
	svc1.AddSourceType(src1)
	svc2 := plan.NewService("foo", "bar")
	svc2.AddSourceType(src2)
	// Test
	tcs := []struct {
		name           string
		svc            plan.Service
		src            plan.SourceTypeValue
		shouldIncrease bool
	}{
		{name: "add source to empty service", svc: svc0, src: src1, shouldIncrease: true},
		{name: "skip adding source to filled service", svc: svc1, src: src1, shouldIncrease: false},
		{name: "add source to filled service", svc: svc2, src: src1, shouldIncrease: true},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			prevLen := len(tc.svc.SourceTypes)
			tc.svc.AddSourceType(tc.src)
			if tc.shouldIncrease {
				if len(tc.svc.SourceTypes) != prevLen+1 {
					t.Fatal("Expected len of source types:", prevLen+1, ". Actual length:", len(tc.svc.SourceTypes))
				}
			} else {
				if len(tc.svc.SourceTypes) != prevLen {
					t.Fatal("Expected service to skip adding the source", tc.src, ". Actual:", tc.svc)
				}
			}
		})
	}
}

func TestAddServicesToPlan(t *testing.T) {
	t.Run("add all services to empty plan", func(t *testing.T) {
		// Setup
		p := plan.NewPlan()
		services := []plan.Service{
			plan.NewService("111", "111"),
			plan.NewService("222", "222"),
			plan.NewService("333", "333"),
		}
		// Test
		p.AddServicesToPlan(services)
		for _, s := range services {
			if _, ok := p.Spec.Inputs.Services[s.ServiceName]; !ok {
				t.Error("Failed to add the service", s, "Actual:", p)
			} else {
				es := p.Spec.Inputs.Services[s.ServiceName]
				if len(es) != 1 || !reflect.DeepEqual(es[0], s) {
					t.Error("Failed to merge the service", s, "correctly. Actual:", p)
				}
			}
		}
	})

	t.Run("merge all services to filled plan", func(t *testing.T) {
		// Setup
		p := plan.NewPlan()
		p.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}
		p.Spec.Inputs.Services["222"] = []plan.Service{plan.NewService("222", "222")}
		p.Spec.Inputs.Services["333"] = []plan.Service{plan.NewService("333", "333"), plan.NewService("333", "444")}
		services := []plan.Service{
			plan.NewService("111", "111"),
			plan.NewService("222", "222"),
			plan.NewService("333", "333"),
			plan.NewService("333", "444"),
		}
		want := plan.NewPlan()
		want.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}
		want.Spec.Inputs.Services["222"] = []plan.Service{plan.NewService("222", "222")}
		want.Spec.Inputs.Services["333"] = []plan.Service{plan.NewService("333", "333"), plan.NewService("333", "444")}
		// Test
		p.AddServicesToPlan(services)
		if !cmp.Equal(want, p) {
			t.Fatalf("The new services didn't get merged into existing services properly. Difference:\n%s:", cmp.Diff(want, p))
		}
	})

	t.Run("merge some services and add some services to filled plan", func(t *testing.T) {
		// Setup
		p := plan.NewPlan()
		p.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}
		p.Spec.Inputs.Services["222"] = []plan.Service{plan.NewService("222", "222")}
		p.Spec.Inputs.Services["333"] = []plan.Service{plan.NewService("333", "333"), plan.NewService("333", "444")}
		svc1 := plan.NewService("444", "444")
		svc1.BuildArtifacts[plan.SourceDirectoryBuildArtifactType] = []string{"src1"}
		svc2 := plan.NewService("444", "444")
		svc2.BuildArtifacts[plan.SourceDirectoryBuildArtifactType] = []string{"src2"}
		services := []plan.Service{
			plan.NewService("111", "111"),
			plan.NewService("222", "222"),
			plan.NewService("333", "333"),
			svc1,
			svc2,
		}
		want := plan.NewPlan()
		want.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}
		want.Spec.Inputs.Services["222"] = []plan.Service{plan.NewService("222", "222")}
		want.Spec.Inputs.Services["333"] = []plan.Service{plan.NewService("333", "333"), plan.NewService("333", "444")}
		want.Spec.Inputs.Services[svc1.ServiceName] = []plan.Service{svc1, svc2}
		// Test
		p.AddServicesToPlan(services)
		if !cmp.Equal(want, p) {
			t.Fatalf("The new services didn't get added and merged into existing services properly. Difference:\n%s:", cmp.Diff(want, p))
		}
	})

	t.Run("merge all services having target options to filled plan", func(t *testing.T) {
		// Setup
		p := plan.NewPlan()
		p.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}

		svc1 := plan.NewService("111", "111")
		svc1.ContainerizationTargetOptions = []string{"opt1"}
		services := []plan.Service{svc1}

		want := plan.NewPlan()
		svc2 := plan.NewService("111", "111")
		svc2.ContainerizationTargetOptions = []string{"opt1"}
		want.Spec.Inputs.Services["111"] = []plan.Service{svc2}

		// Test
		p.AddServicesToPlan(services)
		p.AddServicesToPlan(services)
		if !cmp.Equal(want, p) {
			t.Fatalf("The new services didn't get merged into existing services properly. Difference:\n%s:", cmp.Diff(want, p))
		}
	})

	t.Run("merge all services having source types to filled plan", func(t *testing.T) {
		// Setup
		p := plan.NewPlan()
		p.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}

		svc1 := plan.NewService("111", "111")
		svc1.SourceTypes = []plan.SourceTypeValue{"type1"}
		services := []plan.Service{svc1}

		want := plan.NewPlan()
		svc2 := plan.NewService("111", "111")
		svc2.SourceTypes = []plan.SourceTypeValue{"type1"}
		want.Spec.Inputs.Services["111"] = []plan.Service{svc2}

		// Test
		p.AddServicesToPlan(services)
		p.AddServicesToPlan(services)
		if !cmp.Equal(want, p) {
			t.Fatalf("The new services didn't get merged into existing services properly. Difference:\n%s:", cmp.Diff(want, p))
		}
	})

	t.Run("merge all services having build artifacts to filled plan", func(t *testing.T) {
		// Setup
		p := plan.NewPlan()
		p.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}

		var key1 plan.BuildArtifactTypeValue = "111"

		svc1 := plan.NewService("111", "111")
		svc1.BuildArtifacts[key1] = []string{"art1"}
		services := []plan.Service{svc1}

		want := plan.NewPlan()
		svc2 := plan.NewService("111", "111")
		svc2.BuildArtifacts[key1] = []string{"art1"}
		want.Spec.Inputs.Services["111"] = []plan.Service{svc2}

		// Test
		p.AddServicesToPlan(services)
		p.AddServicesToPlan(services)
		if !cmp.Equal(want, p) {
			t.Fatalf("The new services didn't get merged into existing services properly. Difference:\n%s:", cmp.Diff(want, p))
		}
	})

	t.Run("merge all services having source artifacts to filled plan", func(t *testing.T) {
		// Setup
		p := plan.NewPlan()
		p.Spec.Inputs.Services["111"] = []plan.Service{plan.NewService("111", "111")}

		var key1 plan.SourceArtifactTypeValue = "111"

		svc1 := plan.NewService("111", "111")
		svc1.SourceArtifacts[key1] = []string{"art1"}
		services := []plan.Service{svc1}

		want := plan.NewPlan()
		svc2 := plan.NewService("111", "111")
		svc2.SourceArtifacts[key1] = []string{"art1"}
		want.Spec.Inputs.Services["111"] = []plan.Service{svc2}

		// Test
		p.AddServicesToPlan(services)
		p.AddServicesToPlan(services)
		if !cmp.Equal(want, p) {
			t.Fatalf("The new services didn't get merged into existing services properly. Difference:\n%s:", cmp.Diff(want, p))
		}
	})
}

func TestNewPlan(t *testing.T) {
	p := plan.NewPlan()
	if p.Spec.Inputs.Services == nil || p.Spec.Inputs.TargetInfoArtifacts == nil {
		t.Error("Failed to instantiate the plan fields properly. Actual:", p)
	}
}

func TestNewService(t *testing.T) {
	svcName := "foo"
	var transType plan.TranslationTypeValue = "bar"
	s := plan.NewService(svcName, transType)
	if s.ServiceName != svcName || s.TranslationType != transType {
		t.Error("Service name and translation type have not been set correctly. Expected:", svcName, transType, "Actual:", s.ServiceName, s.TranslationType)
	}
	if s.SourceTypes == nil || s.BuildArtifacts == nil || s.SourceArtifacts == nil {
		t.Error("Failed to instantiate the service fields properly. Actual:", s)
	}
}
