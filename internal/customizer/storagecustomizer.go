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

package customizer

import (
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	irtypes "github.com/konveyor/move2kube/internal/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
)

//storageCustomizer customizes storage
type storageCustomizer struct {
	ir *irtypes.IR
}

const (
	alloption string = "Apply for all"
)

//customize customizes the storage
func (ic *storageCustomizer) customize(ir *irtypes.IR) error {
	ic.ir = ir
	ic.convertHostPathToPVC()

	if len(ic.ir.Storages) == 0 {
		log.Debugf("Empty storage list. Nothing to customize.")
		return nil
	}
	if ic.ir.TargetClusterSpec.StorageClasses == nil || len(ic.ir.TargetClusterSpec.StorageClasses) == 0 {
		s := "No storage classes available in the cluster"
		log.Warnf(s)
		return fmt.Errorf(s)
	}
	claimSvcMap := ic.getPVCs()

	if len(claimSvcMap) == 0 {
		log.Debugf("No service with volumes detected. Storage class configuration not required.")
		return nil
	}

	selectedKeys := []string{}
	for k := range claimSvcMap {
		selectedKeys = append(selectedKeys, k)
	}

	if len(selectedKeys) > 1 {
		if !ic.shouldConfigureSeparately(selectedKeys) {
			storageClass := ic.selectStorageClass(ir.TargetClusterSpec.StorageClasses, alloption, []string{})
			for _, storage := range ic.ir.Storages {
				if storage.StorageType == irtypes.PVCKind {
					storage.PersistentVolumeClaimSpec.StorageClassName = &storageClass
				}
			}
			return nil
		}
	}

	for i, s := range ic.ir.Storages {
		if svs, ok := claimSvcMap[s.Name]; ok {
			storageClassName := ic.selectStorageClass(ic.ir.TargetClusterSpec.StorageClasses, s.Name, svs)
			s.StorageClassName = &storageClassName
			ic.ir.Storages[i] = s
		}
	}

	(*ir) = (*ic.ir)

	return nil
}

func (ic *storageCustomizer) convertHostPathToPVC() {
	hostPathsVisited := map[string]string{}
	for _, service := range ic.ir.Services {
		log.Debugf("Service %s has %d volumes", service.Name, len(service.Volumes))
		for vi, v := range service.Volumes {
			if v.HostPath != nil {
				if name, ok := hostPathsVisited[v.HostPath.Path]; !ok {
					hostPathsVisited[v.HostPath.Path] = ""
					log.Debugf("Detected host path [%+v]", v)
					if !ic.shouldHostPathBeRetained(v.HostPath.Path) {
						hostPathsVisited[v.HostPath.Path] = v.Name
						v.VolumeSource = corev1.VolumeSource{
							PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
								ClaimName: v.Name,
							}}
						service.Volumes[vi] = v
						ic.ir.Services[service.Name] = service
						storageObj := irtypes.Storage{
							StorageType: irtypes.PVCKind,
							Name:        v.Name,
							PersistentVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
								VolumeName: v.Name,
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceStorage: common.DefaultPVCSize,
									},
								},
							}}
						ic.ir.AddStorage(storageObj)
					} else {
						log.Debugf("Host path [%s] is retained", v.HostPath.Path)
					}
				} else {
					v.VolumeSource = corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: name,
						}}
					service.Volumes[vi] = v
					ic.ir.Services[service.Name] = service
				}
			}
		}
	}
}

func (ic storageCustomizer) shouldHostPathBeRetained(hostPath string) bool {
	// if filepath.IsAbs(hostPath) {
	// 	return true
	// }

	problem, err := qatypes.NewConfirmProblem(fmt.Sprintf("Do you want to create PVC for host path [%s]?:", hostPath), []string{"Use PVC for persistent storage wherever applicable"}, false)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	ans, err := problem.GetBoolAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	return !ans
}

func (ic storageCustomizer) shouldConfigureSeparately(claims []string) bool {
	context := make([]string, 2)
	context[0] = "Storage classes have to be configured for below claims:"
	context[1] = fmt.Sprintf("%+v", claims)

	problem, err := qatypes.NewConfirmProblem("Do you want to configure different storage classes for each claim?", context, false)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	ans, err := problem.GetBoolAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	return ans
}

func (ic storageCustomizer) selectStorageClass(storageClasses []string, claimName string, services []string) string {
	var desc string
	if claimName == alloption {
		desc = "Which storage class to use for all persistent volume claims?"
	} else {
		desc = fmt.Sprintf("Which storage class to use for persistent volume claim [%s] used by %+v", claimName, services)
	}
	problem, err := qatypes.NewSelectProblem(desc, []string{"If you have a custom cluster, you can use collect to get storage classes from it."}, storageClasses[0], storageClasses)
	if err != nil {
		log.Fatalf("Unable to create problem : %s", err)
	}
	problem, err = qaengine.FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer : %s", err)
	}
	sc, err := problem.GetStringAnswer()
	if err != nil {
		log.Fatalf("Unable to get answer : %s", err)
	}
	return sc
}

func (ic *storageCustomizer) getPVCs() map[string][]string {
	pvcmap := map[string][]string{}
	for _, s := range ic.ir.Storages {
		if s.StorageType == irtypes.PVCKind {
			svcList := []string{}
			for svcName, svc := range ic.ir.Services {
				for _, v := range svc.Volumes {
					if v.Name == s.Name {
						svcList = append(svcList, svcName)
						break
					}
				}
			}
			pvcmap[s.Name] = svcList
		}
	}
	return pvcmap
}
