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

package apiresource_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/apiresource"
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetSupportedKinds(t *testing.T) {
	netPolicy := apiresource.NetworkPolicy{}
	supKinds := netPolicy.GetSupportedKinds()
	if supKinds == nil || len(supKinds) == 0 {
		t.Fatal("The supported kinds is nil/empty.")
	}
}

func TestCreateNewResources(t *testing.T) {
	t.Run("empty IR and empty supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		plan := plantypes.NewPlan()
		ir := irtypes.NewIR(plan)
		supKinds := []string{}
		// Test
		actual := netPolicy.CreateNewResources(ir, supKinds)
		if actual != nil {
			t.Fatal("Should not have created any objects since IR is empty. Actual:", actual)
		}
	})
	t.Run("empty IR and some supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		plan := plantypes.NewPlan()
		ir := irtypes.NewIR(plan)
		supKinds := []string{"NetworkPolicy"}
		want := []runtime.Object{}
		// Test
		actual := netPolicy.CreateNewResources(ir, supKinds)
		if !cmp.Equal(actual, want) {
			t.Fatalf("Should not have created any objects since IR is empty. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
	t.Run("IR with some services and empty supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		plan := plantypes.NewPlan()
		ir := irtypes.NewIR(plan)
		svc1Name := "svc1"
		svc2Name := "svc2"
		ir.Services = map[string]irtypes.Service{
			svc1Name: irtypes.NewServiceWithName(svc1Name),
			svc2Name: irtypes.NewServiceWithName(svc2Name),
		}
		supKinds := []string{}
		// Test
		actual := netPolicy.CreateNewResources(ir, supKinds)
		if actual != nil {
			t.Fatal("Should not have created any object since the supported kinds is empty. Actual:", actual)
		}
	})
	t.Run("IR with some services and but no acceptable supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		plan := plantypes.NewPlan()
		ir := irtypes.NewIR(plan)
		svc1Name := "svc1"
		svc2Name := "svc2"
		ir.Services = map[string]irtypes.Service{
			svc1Name: irtypes.NewServiceWithName(svc1Name),
			svc2Name: irtypes.NewServiceWithName(svc2Name),
		}
		supKinds := []string{"Pod", "Secret"}
		// Test
		actual := netPolicy.CreateNewResources(ir, supKinds)
		if actual != nil {
			t.Fatal("Should not have created any object since the supported kinds are valid for NetworkPolicy. Actual:", actual)
		}
	})
	t.Run("IR with some services and no networks and some supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		plan := plantypes.NewPlan()
		ir := irtypes.NewIR(plan)
		svc1Name := "svc1"
		svc2Name := "svc2"
		ir.Services = map[string]irtypes.Service{
			svc1Name: irtypes.NewServiceWithName(svc1Name),
			svc2Name: irtypes.NewServiceWithName(svc2Name),
		}
		supKinds := []string{"NetworkPolicy"}
		want := []runtime.Object{}
		// Test
		actual := netPolicy.CreateNewResources(ir, supKinds)
		if !cmp.Equal(actual, want) {
			t.Fatalf("Should not have created any objects since the services don't have networks. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
	t.Run("IR with some services and some networks and some supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		plan := plantypes.NewPlan()
		ir := irtypes.NewIR(plan)
		svc1Name := "svc1"
		svc2Name := "svc2"
		net1 := "net1"
		net2 := "net2"

		ir.Services = map[string]irtypes.Service{
			svc1Name: irtypes.NewServiceWithName(svc1Name),
			svc2Name: irtypes.NewServiceWithName(svc2Name),
		}
		tmpS := ir.Services[svc1Name]
		tmpS.Networks = []string{net1}
		ir.Services[svc1Name] = tmpS

		tmpS = ir.Services[svc2Name]
		tmpS.Networks = []string{net2}
		ir.Services[svc2Name] = tmpS

		supKinds := []string{"NetworkPolicy"}

		testDataPath := "testdata/networkpolicy/create-new-resources.yaml"
		wantNetPols := []networkingv1.NetworkPolicy{}
		if err := common.ReadYaml(testDataPath, &wantNetPols); err != nil {
			t.Fatal("Failed to read the test data. Error:", err)
		}
		want := []runtime.Object{}
		for i := range wantNetPols {
			want = append(want, &wantNetPols[i])
		}
		// Test
		actual := netPolicy.CreateNewResources(ir, supKinds)
		if len(actual) != len(want) {
			t.Fatalf("Expected %d resources to be created. Actual no. of resources %d. Actual list %v", len(want), len(actual), actual)
		}
		for _, wantres := range want {
			matched := false
			for _, actualres := range actual {
				if cmp.Equal(actualres, wantres) {
					if matched {
						t.Fatalf("The expected network policy %v was found more than once in the returned list. Actual: %v", wantres, actual)
					} else {
						matched = true
					}
				}
			}
			if !matched {
				t.Fatalf("Didn't find the expected network policy %v in the returned list. Actual: %v", wantres, actual)
			}
		}
	})
}

func TestConvertToClusterSupportedKinds(t *testing.T) {
	t.Run("empty object and empty supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		obj := &networkingv1.NetworkPolicy{}
		otherObjs := []runtime.Object{}
		supKinds := []string{}
		// Test
		_, ok := netPolicy.ConvertToClusterSupportedKinds(obj, supKinds, otherObjs)
		if ok {
			t.Fatal("Should have failed since supported kinds is empty.")
		}
	})
	t.Run("some object and empty supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		obj := helperCreateNetworkPolicy("net1")
		otherObjs := []runtime.Object{}
		supKinds := []string{}
		// Test
		_, ok := netPolicy.ConvertToClusterSupportedKinds(obj, supKinds, otherObjs)
		if ok {
			t.Fatal("Should have failed since supported kinds is empty.")
		}
	})
	t.Run("invalid object and correct supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		obj := helperCreateSecret("sec1", map[string][]byte{"key1": []byte("val1")})
		otherObjs := []runtime.Object{}
		supKinds := []string{"Pod", "NetworkPolicy", "Secret"}
		// Test
		_, ok := netPolicy.ConvertToClusterSupportedKinds(obj, supKinds, otherObjs)
		if ok {
			t.Fatal("Should have failed since the object is not a valid network policy.")
		}
	})
	t.Run("some object and correct supported kinds", func(t *testing.T) {
		// Setup
		netPolicy := apiresource.NetworkPolicy{}
		obj := helperCreateNetworkPolicy("net1")
		otherObjs := []runtime.Object{}
		supKinds := []string{"Pod", "NetworkPolicy", "Secret"}
		want := []runtime.Object{helperCreateNetworkPolicy("net1")}
		// Test
		actual, ok := netPolicy.ConvertToClusterSupportedKinds(obj, supKinds, otherObjs)
		if !ok {
			t.Fatal("Failed to convert to cluster supported kind, Function returned false. Actual:", actual)
		}
		if !cmp.Equal(actual, want) {
			t.Fatalf("Failed to convert the network policy properly. Differences:\n%s", cmp.Diff(want, actual))
		}
	})
}

func helperCreateNetworkPolicy(name string) *networkingv1.NetworkPolicy {
	return &networkingv1.NetworkPolicy{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NetworkPolicy",
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"foo": "bar"},
			},
		},
	}
}

func helperCreateSecret(name string, secretData map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}
}
