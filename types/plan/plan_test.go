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
	"path/filepath"
	"reflect"
	"testing"

	"github.com/konveyor/move2kube/internal/common"
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
	t.Run("merge artifact type and ignore supported kinds from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", "333", "444", false}
		out2 := plan.KubernetesOutput{ArtifactType: "type1", IgnoreUnsupportedKinds: true}
		want := out1
		want.ArtifactType = "type1"
		want.IgnoreUnsupportedKinds = true
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge registry url from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", "333", "444", false}
		out2 := plan.KubernetesOutput{ArtifactType: "type1", IgnoreUnsupportedKinds: true, RegistryURL: "url1"}
		want := out1
		want.ArtifactType = "type1"
		want.IgnoreUnsupportedKinds = true
		want.RegistryURL = "url1"
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge registry namespace from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", "333", "444", false}
		out2 := plan.KubernetesOutput{ArtifactType: "type1", IgnoreUnsupportedKinds: true, RegistryNamespace: "namespace1"}
		want := out1
		want.ArtifactType = "type1"
		want.IgnoreUnsupportedKinds = true
		want.RegistryNamespace = "namespace1"
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge image pull secret from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", "333", "444", false}
		out2 := plan.KubernetesOutput{ArtifactType: "type1", IgnoreUnsupportedKinds: true}
		want := out1
		want.ArtifactType = "type1"
		want.IgnoreUnsupportedKinds = true
		out1.Merge(out2)
		if out1 != want {
			t.Fatal("Failed to merge the fields properly. Expected:", want, "Actual:", out1)
		}
	})
	t.Run("merge cluster type from new k8s output into filled k8s output", func(t *testing.T) {
		out1 := plan.KubernetesOutput{"111", "222", "333", "444", false}
		out2 := plan.KubernetesOutput{ArtifactType: "type1", IgnoreUnsupportedKinds: true, ClusterType: "clus_type1"}
		want := out1
		want.ArtifactType = "type1"
		want.IgnoreUnsupportedKinds = true
		want.ClusterType = "clus_type1"
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

func TestGetFullPath(t *testing.T) {
	tempPath := common.TempPath
	assetsDir := common.AssetsDir
	assetsPath := common.AssetsPath
	root := "tests/getfullpath/root"
	if root == assetsPath {
		root += "1234"
	}
	j := filepath.Join
	// Assuming paths don't need to be cleaned since GetFullPath is an internal functions.
	// Paths should be cleaned when first taken as input.
	var testcases = []struct{ in, out string }{
		{j("foo"), j(root, "foo")},
		{j("foo", "bar"), j(root, "foo", "bar")},
		{j("foo", assetsPath, "bar"), j(root, "foo", assetsPath, "bar")},
		{j(tempPath, assetsDir, "foo"), j(root, tempPath, assetsDir, "foo")},
		{j(assetsDir, "foo"), j(tempPath, assetsDir, "foo")},
		{j(assetsDir, "foo", "bar"), j(tempPath, assetsDir, "foo", "bar")},
	}

	p := plan.NewPlan()
	p.Spec.Inputs.RootDir = root
	for _, testcase := range testcases {
		if res := p.GetFullPath(testcase.in); res != testcase.out {
			t.Error("Input:", testcase.in, "Expected:", testcase.out, "Actual:", res)
		}
	}
}

func TestGetRelativePath(t *testing.T) {
	assetsPath := common.AssetsPath
	assetsDir := common.AssetsDir
	root := "tests/getfullpath/root"
	if root == assetsDir {
		root += "1234"
	}
	j := filepath.Join
	/*
		// Since the result is a relative path it will contain ../ in cases like the one shown below:
		root := "foo/bar"
		input := "foo/abc"
		GetRelativePath(input) == "../abc"
	*/
	var testcases = []struct {
		in   string
		out  string
		fail bool
	}{
		{j(root, "foo"), j("foo"), false},
		{j(root, "foo", "bar"), j("foo", "bar"), false},
		{j(root, "foo", assetsDir, "bar"), j("foo", assetsDir, "bar"), false},
		{j(assetsPath, "foo"), j(assetsDir, "foo"), false},
		{j(assetsPath, "foo", "bar"), j(assetsDir, "foo", "bar"), false},
		{j("/", "foo", "bar"), "", true},
	}

	p := plan.NewPlan()
	p.Spec.Inputs.RootDir = root
	for _, testcase := range testcases {
		res, err := p.GetRelativePath(testcase.in)
		if testcase.fail {
			if err == nil {
				t.Error("Input:", testcase.in, "Expected testcase to fail. Actual:", res, err)
			}
		} else {
			if err != nil {
				t.Error("Input:", testcase.in, "Expected testcase to succeed. Actual:", res, err)
			} else if res != testcase.out {
				t.Error("Input:", testcase.in, "Expected:", testcase.out, "Actual:", res, err)
			}
		}
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
		if !reflect.DeepEqual(want, p) {
			t.Fatal("The new services didn't get merged into existing services properly. Expected:", want, "Actual:", p)
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
		if !reflect.DeepEqual(want, p) {
			t.Fatal("The new services didn't get added and merged into existing services properly. Expected:", want, "Actual:", p)
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
		if !reflect.DeepEqual(want, p) {
			t.Fatal("The new services didn't get merged into existing services properly. Expected:", want, "Actual:", p)
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
		if !reflect.DeepEqual(want, p) {
			t.Fatal("The new services didn't get merged into existing services properly. Expected:", want, "Actual:", p)
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
		if !reflect.DeepEqual(want, p) {
			t.Fatal("The new services didn't get merged into existing services properly. Expected:", want, "Actual:", p)
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
		if !reflect.DeepEqual(want, p) {
			t.Fatal("The new services didn't get merged into existing services properly. Expected:", want, "Actual:", p)
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
