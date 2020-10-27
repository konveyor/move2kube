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
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/google/go-cmp/cmp"

	common "github.com/konveyor/move2kube/internal/common"
	parameterize "github.com/konveyor/move2kube/internal/parameterizer"
	"github.com/konveyor/move2kube/internal/types"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

func TestParameterizer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("1.IR with no services, no storage", func(t *testing.T) {

		ir := getIRWithoutServices()
		irAfter, err := parameterize.Parameterize(ir)

		if err != nil {
			t.Fatal("Failed to parameterize. Error:", err)
		}

		irAfter.TargetClusterSpec.Host = ""

		if !cmp.Equal(ir, irAfter, cmpopts.EquateEmpty()) {
			t.Fatalf("TODO explanation. Difference:\n%s", cmp.Diff(ir, irAfter))
		}

	})

	/*
		t.Run("2.IR containing services that have no containers", func(t *testing.T) {

			ir := getIRWithServicesAndWithoutContainers()
			irAfter, err := parameterize.Parameterize(ir)
			if err != nil {
				t.Fatal("Failed to parameterize. Error:", err)
			}

			// oracle
			// if the service has no containers, then no changes will be done
			// ir = irAfter
			if !cmp.Equal(ir, irAfter, cmpopts.EquateEmpty()) {
				t.Fatalf("TODO explanation. Difference:\n%s", cmp.Diff(ir, irAfter))
			}

		})
	*/

	t.Run("3.IR containing services with containers", func(t *testing.T) {

		ir := getIRWithServicesAndContainers()
		irAfter, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize. Error:", err)
		}
		fmt.Printf("%T", irAfter)
		// oracle

	})

	t.Run("4.IR with no services , but storage, storage type is not PVCKind ", func(t *testing.T) {

		// as the type is not PVCKind, the scMap will be empty
		// -> no parameterizations -> before = after
		ir := getIRWithStorageNotPVCKind()
		irAfter, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize. Error:", err)
		}

		// oracle

		irAfter.TargetClusterSpec.Host = ""
		if !cmp.Equal(ir, irAfter, cmpopts.EquateEmpty()) {
			t.Fatalf("TODO explanation. Difference:\n%s", cmp.Diff(ir, irAfter))
		}

	})

	t.Run("5.IR with no services , but storage, storage type is PVCKind ", func(t *testing.T) {

		// scMap should be 1, we can parameterize
		ir := getIRWithStoragePVCKind()
		irAfter, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize. Error:", err)
		}

		// oracle:
		paramSC := "{{ .Values.storageclass }}"
		for _, j := range irAfter.Storages {
			x := *j.PersistentVolumeClaimSpec.StorageClassName
			if x != paramSC {
				t.Fatal("Failed to parameterize. Error:", err)
			}
		}
	})

	t.Run("6.IR with no services , check ingress parameterizer ", func(t *testing.T) {

		// scMap should be 1, we can parameterize
		ir := getIRWithoutServices()
		irAfter, err := parameterize.Parameterize(ir)
		if err != nil {
			t.Fatal("Failed to parameterize. Error:", err)
		}

		hostParam := "{{ .Release.Name }}-"

		if irAfter.TargetClusterSpec.Host != hostParam {
			t.Fatal("Failed to parameterize")

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
	c1 := corev1.Container{
		Name: "container-1",
	}
	c2 := corev1.Container{
		Name: "container-2",
	}
	//c1.ImagePullPolicy = v1.PullAlways // not needed for the moment
	//c2.ImagePullPolicy = v1.PullAlways // not needed for the moment
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
