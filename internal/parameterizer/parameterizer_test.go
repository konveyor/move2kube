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

package parameterize_test

import (
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/google/go-cmp/cmp"

	common "github.com/konveyor/move2kube/internal/common"
	parameterize "github.com/konveyor/move2kube/internal/parameterizer"
	"github.com/konveyor/move2kube/internal/types"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types/output"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

func TestParameterizer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("1.IR with no services, no storage", func(t *testing.T) {

		ir := getIRWithoutServices()
		want := getIRWithoutServices()
		actual, err := parameterize.Parameterize(ir)

		if err != nil {
			t.Fatal("Failed to parameterize the IR properly. Error:", err)
		}

		actual.TargetClusterSpec.Host = ""

		if !cmp.Equal(actual, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to parameterize the IR properly. Difference:\n%s", cmp.Diff(want, actual))
		}

	})

	t.Run("2.IR containing services that have no containers", func(t *testing.T) {

		ir := getIRWithServicesAndWithoutContainers()
		want := getIRWithServicesAndWithoutContainers()
		actual, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize the IR properly. Error:", err)
		}

		want.Values.Services = map[string]output.Service{
			"svcname1": {Containers: map[string]output.Container{}},
			"svcname2": {Containers: map[string]output.Container{}},
		}
		actual.TargetClusterSpec.Host = ""

		if !cmp.Equal(actual, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to parameterize the IR properly. Difference:\n%s", cmp.Diff(want, actual))
		}

	})

	t.Run("3.IR containing services with containers", func(t *testing.T) {

		ir := getIRWithServicesAndContainers()
		want := getIRWithServicesAndContainers()
		actual, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize the IR properly. Error:", err)
		}

		want.TargetClusterSpec.Host = "{{ .Release.Name }}-{{ .Values.ingresshost }}"
		want.Services["svcname1"].PodSpec.Containers[0].Image = `:{{ index .Values.services "svcname1" "containers" "container-1" "imagetag"  }}`
		want.Values.Services = map[string]output.Service{"svcname1": {Containers: map[string]output.Container{"container-1": {TagName: "latest"}}}}

		if !cmp.Equal(actual, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to parameterize the IR properly. Difference:\n%s", cmp.Diff(want, actual))
		}

	})

	t.Run("4.IR with no services , but storage, storage type is not PVCKind ", func(t *testing.T) {

		// as the type is not PVCKind, the scMap will be empty
		// -> no parameterizations -> before = after
		ir := getIRWithStorageNotPVCKind()
		want := getIRWithStorageNotPVCKind()
		actual, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize the IR properly. Error:", err)
		}

		// oracle
		actual.TargetClusterSpec.Host = ""
		if !cmp.Equal(actual, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to parameterize the IR properly. Difference:\n%s", cmp.Diff(want, actual))
		}

	})

	t.Run("5.IR with no services , but storage, storage type is PVCKind ", func(t *testing.T) {

		// scMap should be 1, we can parameterize
		ir := getIRWithStoragePVCKind()
		want := getIRWithStoragePVCKind()
		actual, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize the IR properly. Error:", err)
		}

		paramSC := "{{ .Values.storageclass }}"
		want.Storages[0].PersistentVolumeClaimSpec.StorageClassName = &paramSC
		want.TargetClusterSpec.Host = "{{ .Release.Name }}-{{ .Values.ingresshost }}"
		want.Values.Services = map[string]output.Service{}
		want.Values.StorageClass = "storage-1cn"

		if !cmp.Equal(actual, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to parameterize the IR properly. Difference:\n%s", cmp.Diff(want, actual))
		}

	})

	t.Run("6.IR with no services , check ingress parameterizer ", func(t *testing.T) {

		// scMap should be 1, we can parameterize
		ir := getIRWithoutServices()
		actual, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize the IR properly. Error:", err)
		}

		hostParam := "{{ .Release.Name }}-{{ .Values.ingresshost }}"

		if actual.TargetClusterSpec.Host != hostParam {
			t.Fatal("Failed to parameterize the IR properly")

		}
	})

}

func getIRWithoutServices() types.IR {
	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	return ir
}

func getIRWithServicesAndWithoutContainers() types.IR {
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

func getIRWithServicesAndContainers() types.IR {
	c1 := corev1.Container{Name: "container-1"}
	svcname1 := "svcname1"
	svc1 := types.Service{Name: svcname1, Replicas: 2}
	svc1.Containers = append(svc1.Containers, c1)
	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Services[svcname1] = svc1

	return ir
}

func getIRWithStoragePVCKind() types.IR {

	storage1cn := "storage-1cn" // example class name
	storage1 := irtypes.Storage{
		StorageType: irtypes.PVCKind,
		Name:        "storage-1",
		PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "storage-1",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: common.DefaultPVCSize,
				},
			},
			StorageClassName: &storage1cn,
		}}

	p := plantypes.NewPlan()
	ir := types.NewIR(p)

	ir.Storages = append(ir.Storages, storage1)
	return ir
}

func getIRWithStorageNotPVCKind() types.IR {
	// instead of using a storage type PVCKind, we use SecretKind

	storage1cn := "storage-1cn" // example class name

	storage1 := irtypes.Storage{
		StorageType: irtypes.SecretKind,
		Name:        "storage-1",
		PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
			VolumeName: "storage-1",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: common.DefaultPVCSize,
				},
			},
			StorageClassName: &storage1cn,
		}}

	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	ir.Storages = append(ir.Storages, storage1)
	return ir
}

func getIRWithContainer() types.IR {

	// Setup Containers
	cname1 := "name1"
	cnew1 := true
	cont1 := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue,
		cname1,
		cnew1)

	// Get empty IR
	p := plantypes.NewPlan()
	ir := types.NewIR(p)

	// Add Containers to IR
	ir.Containers = append(ir.Containers, cont1)
	return ir
}
