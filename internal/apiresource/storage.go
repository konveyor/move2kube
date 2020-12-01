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

package apiresource

import (
	"github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Storage handles all storage objectss
type Storage struct {
	Cluster collecttypes.ClusterMetadataSpec
}

// GetSupportedKinds returns cluster supported kinds
func (s *Storage) GetSupportedKinds() []string {
	return []string{string(irtypes.PVCKind), string(irtypes.ConfigMapKind), string(irtypes.SecretKind)}
}

// CreateNewResources converts IR objects to runtime objects
func (s *Storage) CreateNewResources(ir irtypes.EnhancedIR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}

	for _, stObj := range ir.Storages {
		if stObj.StorageType == irtypes.ConfigMapKind {
			if !common.IsStringPresent(supportedKinds, string(irtypes.ConfigMapKind)) && common.IsStringPresent(supportedKinds, string(irtypes.SecretKind)) {
				objs = append(objs, s.createSecret(stObj))
			} else {
				objs = append(objs, s.createConfigMap(stObj))
			}
		}

		if stObj.StorageType == irtypes.SecretKind {
			if !common.IsStringPresent(supportedKinds, string(irtypes.SecretKind)) && common.IsStringPresent(supportedKinds, string(irtypes.ConfigMapKind)) {
				objs = append(objs, s.createConfigMap(stObj))
			} else {
				objs = append(objs, s.createSecret(stObj))
			}
		}

		if stObj.StorageType == irtypes.PullSecretKind {
			objs = append(objs, s.createSecret(stObj))
		}

		if stObj.StorageType == irtypes.PVCKind {
			objs = append(objs, s.createPVC(stObj))
		}
	}

	return objs
}

// ConvertToClusterSupportedKinds converts kinds to cluster supported kinds
func (s *Storage) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR) ([]runtime.Object, bool) {
	if cfgMap, ok := obj.(*corev1.ConfigMap); ok {
		if !common.IsStringPresent(supportedKinds, string(irtypes.ConfigMapKind)) && common.IsStringPresent(supportedKinds, string(irtypes.SecretKind)) {
			return []runtime.Object{convertCfgMapToSecret(*cfgMap)}, true
		}
		return []runtime.Object{cfgMap}, true
	}

	if secret, ok := obj.(*corev1.Secret); ok {
		if !common.IsStringPresent(supportedKinds, string(irtypes.SecretKind)) && common.IsStringPresent(supportedKinds, string(irtypes.ConfigMapKind)) {
			return []runtime.Object{convertSecretToCfgMap(*secret)}, true
		}
		return []runtime.Object{secret}, true
	}

	if pvc, ok := obj.(*corev1.PersistentVolumeClaim); ok {
		if !common.IsStringPresent(supportedKinds, string(irtypes.PVCKind)) {
			log.Warnf("PVC not supported in target cluster. [%s]", pvc.Name)
		}
		return []runtime.Object{pvc}, true
	}
	return nil, false
}

func (s *Storage) createConfigMap(st irtypes.Storage) *corev1.ConfigMap {
	cmName := common.MakeFileNameCompliant(st.Name)

	data := map[string]string{}
	for k, v := range st.Content {
		data[k] = string(v)
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.ConfigMapKind),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cmName,
		},
		Data: data,
	}
	return configMap
}

func (s *Storage) createSecret(st irtypes.Storage) *corev1.Secret {
	secretName := common.MakeFileNameCompliant(st.Name) // TODO: probably remove this. Names should be manipulated at a higher level.
	secType := corev1.SecretTypeOpaque
	if st.SecretType != "" {
		secType = st.SecretType
	} else if st.StorageType == irtypes.PullSecretKind {
		secType = corev1.SecretTypeDockerConfigJson
	}
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.SecretKind),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        secretName,
			Annotations: st.Annotations,
		},
		Type:       secType,
		StringData: st.StringData,
		Data:       st.Content,
	}
	return secret
}

func (s *Storage) createPVC(st irtypes.Storage) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.PVCKind),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: st.Name,
		},
		Spec: st.PersistentVolumeClaimSpec,
	}

	log.Debugf("%+v", st.PersistentVolumeClaimSpec)
	return pvc
}

func convertCfgMapToSecret(cfgMap corev1.ConfigMap) *corev1.Secret {

	secretDataMap := stringMapToByteMap(cfgMap.Data)

	s := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.SecretKind),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   cfgMap.Name,
			Labels: cfgMap.Labels,
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretDataMap,
	}

	return s
}

func convertSecretToCfgMap(s corev1.Secret) *corev1.ConfigMap {
	cmDataMap := byteMapToStringMap(s.Data)

	cm := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(irtypes.ConfigMapKind),
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   s.Name,
			Labels: s.Labels,
		},
		Data: cmDataMap,
	}

	return cm
}

func convertPVCVolumeToEmptyVolume(vPVC corev1.Volume) *corev1.Volume {
	vEmptySrc := &corev1.VolumeSource{
		EmptyDir: &corev1.EmptyDirVolumeSource{},
	}

	return &corev1.Volume{
		Name:         vPVC.Name,
		VolumeSource: *vEmptySrc,
	}
}

func convertCfgMapVolumeToSecretVolume(vCfgMap corev1.Volume) *corev1.Volume {
	vSecretVolSrc := corev1.VolumeSource{
		Secret: &corev1.SecretVolumeSource{
			SecretName:  vCfgMap.ConfigMap.Name,
			Items:       vCfgMap.ConfigMap.Items,
			DefaultMode: vCfgMap.ConfigMap.DefaultMode,
		},
	}

	v := &corev1.Volume{
		Name:         vCfgMap.Name,
		VolumeSource: vSecretVolSrc,
	}
	return v
}

func convertSecretVolumeToCfgMapVolume(vs corev1.Volume) *corev1.Volume {
	vSrc := &corev1.ConfigMapVolumeSource{}
	vSrc.Name = vs.Secret.SecretName
	vSrc.Items = vs.Secret.Items
	vSrc.DefaultMode = vs.Secret.DefaultMode

	vCMVolSrc := corev1.VolumeSource{
		ConfigMap: vSrc,
	}

	v := &corev1.Volume{
		Name:         vs.Secret.SecretName,
		VolumeSource: vCMVolSrc,
	}

	return v
}

func convertVolumeBySupportedKind(volume corev1.Volume, cluster collecttypes.ClusterMetadataSpec) (nvolume corev1.Volume) {

	if volume == (corev1.Volume{}) {
		return corev1.Volume{}
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
			log.Warnf("PVC not supported in target cluster. Defaulting volume [%s] to emptyDir", volume.Name)
			return *vEmpty

		}
		return volume
	}
	if volume.VolumeSource.HostPath != nil || volume.VolumeSource.EmptyDir != nil {
		return volume
	}
	log.Warnf("Unsupported storage type (volume) detected")

	return corev1.Volume{}
}

func stringMapToByteMap(sm map[string]string) map[string][]byte {
	bm := map[string][]byte{}

	for k, v := range sm {
		bm[k] = []byte(v)
	}

	return bm
}

func byteMapToStringMap(bm map[string][]byte) map[string]string {
	sm := map[string]string{}

	for k, v := range bm {
		sm[k] = string(v)
	}

	return sm
}
