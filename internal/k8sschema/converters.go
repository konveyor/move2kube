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

package k8sschema

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core "k8s.io/kubernetes/pkg/apis/core"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/konveyor/move2kube/internal/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
)

// ConvertToSupportedVersion converts obj to a supported Version
func ConvertToSupportedVersion(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, ignoreUnsupportedKinds bool) (runtime.Object, error) {
	newobj, err := convertToSupportedVersion(obj, clusterSpec)
	if err != nil {
		log.Debugf("Unable to translate object to a supported version : %s.", err)
		if ignoreUnsupportedKinds {
			log.Warnf("Ignoring object : %+v", obj.GetObjectKind())
			return newobj, err
		}
		log.Debugf("Attempting to move to the preferred version")
		if obj.GetObjectKind().GroupVersionKind().Version == core.SchemeGroupVersion.Version {
			newobj, err = convertToPreferredVersion(obj)
			if err != nil {
				log.Warnf("Unable to support to preferred version : %+v", obj.GetObjectKind())
				newobj = obj
			}
		} else {
			log.Debugf("Returning obj in original version : %+v", obj.GetObjectKind())
			newobj = obj
		}
	}
	return newobj, nil
}

// ConvertToSupportedVersion converts obj to a supported Version
func convertToSupportedVersion(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec) (newobj runtime.Object, err error) {
	objvk := obj.GetObjectKind().GroupVersionKind()
	objgv := objvk.GroupVersion()
	kind := objvk.Kind
	log.Debugf("Converting %s to supported version", kind)
	versions := clusterSpec.GetSupportedVersions(kind)
	if versions == nil || len(versions) == 0 {
		return nil, fmt.Errorf("Kind %s unsupported in target cluster : %+v", kind, obj.GetObjectKind())
	}
	log.Debugf("Supported Versions : %+v", versions)
	if kind == common.ServiceKind && objgv.Group == knativev1.SchemeGroupVersion.Group {
		return obj, nil
	}
	for _, v := range versions {
		gv, err := schema.ParseGroupVersion(v)
		if err != nil {
			log.Errorf("Unable to parse group version %s : %s", v, err)
			continue
		}
		if kind == common.ServiceKind && gv.Group == knativev1.SchemeGroupVersion.Group {
			continue
		}
		newobj, err := ConvertToVersion(obj, gv)
		if err != nil {
			log.Errorf("Unable to convert : %s", err)
			continue
		}
		scheme.Default(newobj)
		return newobj, err
	}
	scheme.Default(obj)
	return obj, fmt.Errorf("Unable to convert to a supported version : %+v", obj.GetObjectKind())
}

// ConvertToPreferredVersion converts obj to a preferred Version
func convertToPreferredVersion(obj runtime.Object) (newobj runtime.Object, err error) {
	objvk := obj.GetObjectKind().GroupVersionKind()
	objgv := objvk.GroupVersion()
	kind := objvk.Kind
	log.Debugf("Converting %s to preferred version", kind)
	versions := scheme.PrioritizedVersionsForGroup(objgv.Group)
	if kind == common.ServiceKind && objgv.Group == knativev1.SchemeGroupVersion.Group {
		return obj, nil
	}
	for _, v := range versions {
		newobj, err := ConvertToVersion(obj, v)
		if err != nil {
			log.Errorf("Unable to convert : %s", err)
			continue
		}
		scheme.Default(newobj)
		return newobj, err
	}
	scheme.Default(obj)
	return obj, fmt.Errorf("Unable to convert to a preferred version : %+v", obj.GetObjectKind())
}

// ConvertToVersion converts objects to a version
func ConvertToVersion(obj runtime.Object, dgv schema.GroupVersion) (newobj runtime.Object, err error) {
	log.Debugf("Attempting to convert %s to %s", obj.GetObjectKind().GroupVersionKind(), dgv)
	objvk := obj.GetObjectKind().GroupVersionKind()
	objgv := objvk.GroupVersion()
	kind := objvk.Kind
	newobj, err = checkAndConvertToVersion(obj, dgv)
	if err == nil {
		return newobj, nil
	}
	log.Debugf("Unable to do direct translation : %s", err)
	akt := liasonscheme.AllKnownTypes()
	for kt := range akt {
		if kind != kt.Kind {
			continue
		}
		log.Debugf("Attempting conversion of %s obj to %s", objgv, kt)
		uvobj, err := checkAndConvertToVersion(obj, kt.GroupVersion())
		if err != nil {
			log.Errorf("Unable to convert to unversioned object : %s", err)
			continue
		}
		log.Debugf("Converted %s obj to %s", objgv, kt)
		newobj, err = checkAndConvertToVersion(uvobj, dgv)
		if err == nil {
			return newobj, nil
		}
		log.Errorf("Unable to convert through unversioned object : %s", kt)
	}
	err = fmt.Errorf("Unable to do convert %s to %s", objgv, dgv)
	log.Errorf("%s", err)
	return obj, err
}

// ConvertToLiasonScheme converts objects to liason type
func ConvertToLiasonScheme(obj runtime.Object) (newobj runtime.Object, err error) {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	akt := liasonscheme.AllKnownTypes()
	for kt := range akt {
		if kind != kt.Kind {
			continue
		}
		log.Debugf("Attempting conversion of %s obj to %s", obj.GetObjectKind().GroupVersionKind(), kt)
		uvobj, err := checkAndConvertToVersion(obj, kt.GroupVersion())
		if err != nil {
			log.Errorf("Unable to convert to unversioned object : %s", err)
			continue
		}
		return uvobj, nil
	}
	return obj, fmt.Errorf("Unable to find liason version for kind %s", kind)
}

func checkAndConvertToVersion(obj runtime.Object, dgv schema.GroupVersion) (newobj runtime.Object, err error) {
	if obj.GetObjectKind().GroupVersionKind().GroupVersion() == dgv {
		return obj, nil
	}
	newobj, err = scheme.ConvertToVersion(obj, dgv)
	if err != nil {
		return
	}
	gvk := obj.GetObjectKind().GroupVersionKind()
	gvk.Group = dgv.Group
	gvk.Version = dgv.Version
	newobj.GetObjectKind().SetGroupVersionKind(gvk)
	return newobj, nil
}

func convertBetweenObjects(in interface{}, out interface{}) error {
	err := scheme.Convert(in, out, nil)
	if err != nil {
		log.Errorf("Unable to convert from %T to %T : %s", in, out, err)
	}
	return err
}

// ConvertToV1PodSpec podspec to v1 pod spec
func ConvertToV1PodSpec(podSpec *core.PodSpec) corev1.PodSpec {
	vPodSpec := corev1.PodSpec{}
	err := convertBetweenObjects(podSpec, &vPodSpec)
	if err != nil {
		log.Errorf("Unable to convert PodSpec to versioned PodSpec : %s", err)
	}
	return vPodSpec
}

// ConvertToPodSpec podspec to core pod spec
func ConvertToPodSpec(podspec *corev1.PodSpec) core.PodSpec {
	uvPodSpec := core.PodSpec{}
	err := convertBetweenObjects(podspec, &uvPodSpec)
	if err != nil {
		log.Errorf("Unable to convert versioned PodSpec to unversioned PodSpec : %s", err)
	}
	return uvPodSpec
}
