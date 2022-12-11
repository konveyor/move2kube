/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package apiresource

import (
	"github.com/konveyor/move2kube/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// Storage handles all storage objectss
type Storage struct {
}

// getSupportedKinds returns cluster supported kinds
func (s *Storage) getSupportedKinds() []string {
	return []string{string(irtypes.PVCKind), string(irtypes.ConfigMapKind), string(irtypes.SecretKind)}
}

// createNewResources converts IR objects to runtime objects
func (s *Storage) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	for _, stObj := range ir.Storages {
		if stObj.StorageType == irtypes.ConfigMapKind {
			objs = append(objs, s.createConfigMap(stObj))
		}
		if stObj.StorageType == irtypes.SecretKind || stObj.StorageType == irtypes.PullSecretKind {
			objs = append(objs, s.createSecret(stObj))
		}
		if stObj.StorageType == irtypes.PVCKind {
			objs = append(objs, s.createPVC(stObj))
		}
	}
	return objs
}

// convertToClusterSupportedKinds converts kinds to cluster supported kinds
func (s *Storage) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsPresent(s.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}

func (s *Storage) createConfigMap(st irtypes.Storage) *core.ConfigMap {
	data := map[string]string{}
	for k, v := range st.Content {
		data[k] = string(v)
	}

	configMap := &core.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.ConfigMapKind),
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: st.Name,
		},
		Data: data,
	}
	return configMap
}

func (s *Storage) createSecret(st irtypes.Storage) *core.Secret {
	secType := core.SecretTypeOpaque
	if st.SecretType != "" {
		secType = st.SecretType
	} else if st.StorageType == irtypes.PullSecretKind {
		secType = core.SecretTypeDockerConfigJSON
	}
	secret := &core.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.SecretKind),
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        st.Name,
			Annotations: st.Annotations,
		},
		Type: secType,
		Data: st.Content,
	}
	return secret
}

func (s *Storage) createPVC(st irtypes.Storage) *core.PersistentVolumeClaim {
	logrus.Trace("Storage.createPVC start")
	defer logrus.Trace("Storage.createPVC end")
	pvc := &core.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.PVCKind),
			APIVersion: core.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: st.Name,
		},
		Spec: st.PersistentVolumeClaimSpec,
	}

	logrus.Debugf("%+v", st.PersistentVolumeClaimSpec)
	return pvc
}

func convertPVCVolumeToEmptyVolume(vPVC core.Volume) *core.Volume {
	vEmptySrc := &core.VolumeSource{
		EmptyDir: &core.EmptyDirVolumeSource{},
	}

	return &core.Volume{
		Name:         vPVC.Name,
		VolumeSource: *vEmptySrc,
	}
}

func convertCfgMapVolumeToSecretVolume(vCfgMap core.Volume) *core.Volume {
	vSecretVolSrc := core.VolumeSource{
		Secret: &core.SecretVolumeSource{
			SecretName:  vCfgMap.ConfigMap.Name,
			Items:       vCfgMap.ConfigMap.Items,
			DefaultMode: vCfgMap.ConfigMap.DefaultMode,
		},
	}

	v := &core.Volume{
		Name:         vCfgMap.Name,
		VolumeSource: vSecretVolSrc,
	}
	return v
}

func convertSecretVolumeToCfgMapVolume(vs core.Volume) *core.Volume {
	vSrc := &core.ConfigMapVolumeSource{}
	vSrc.Name = vs.Secret.SecretName
	vSrc.Items = vs.Secret.Items
	vSrc.DefaultMode = vs.Secret.DefaultMode

	vCMVolSrc := core.VolumeSource{
		ConfigMap: vSrc,
	}

	v := &core.Volume{
		Name:         vs.Secret.SecretName,
		VolumeSource: vCMVolSrc,
	}

	return v
}

func convertVolumeBySupportedKind(volume core.Volume, cluster collecttypes.ClusterMetadataSpec) (nvolume core.Volume) {

	if volume == (core.Volume{}) {
		return core.Volume{}
	}

	if volume.VolumeSource.ConfigMap != nil {
		if cluster.GetSupportedVersions(string(irtypes.ConfigMapKind)) == nil && cluster.GetSupportedVersions(string(irtypes.SecretKind)) != nil {
			return *convertCfgMapVolumeToSecretVolume(volume)
		}
		return volume
	}
	if volume.VolumeSource.Secret != nil {
		if cluster.GetSupportedVersions(string(irtypes.SecretKind)) == nil && cluster.GetSupportedVersions(string(irtypes.ConfigMapKind)) != nil {
			return *convertSecretVolumeToCfgMapVolume(volume)
		}
		return volume
	}
	if volume.VolumeSource.PersistentVolumeClaim != nil {
		//PVC -> Empty (If PVC not available)
		if cluster.GetSupportedVersions(string(irtypes.PVCKind)) == nil {
			vEmpty := convertPVCVolumeToEmptyVolume(volume)
			logrus.Warnf("PVC not supported in target cluster. Defaulting volume [%s] to emptyDir", volume.Name)
			return *vEmpty
		}
		return volume
	}
	if volume.VolumeSource.HostPath != nil || volume.VolumeSource.EmptyDir != nil {
		return volume
	}
	logrus.Warnf("Unsupported storage type (volume) detected: %#v", volume)

	return core.Volume{}
}
