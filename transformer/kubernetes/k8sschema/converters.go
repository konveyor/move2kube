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

package k8sschema

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	core "k8s.io/kubernetes/pkg/apis/core"
	knativev1 "knative.dev/serving/pkg/apis/serving/v1"

	"github.com/konveyor/move2kube-wasm/common"
	collecttypes "github.com/konveyor/move2kube-wasm/types/collection"
	"github.com/sirupsen/logrus"
)

// ConvertToSupportedVersion converts obj to a supported Version
func ConvertToSupportedVersion(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, setDefaultValuesInYamls bool) (runtime.Object, error) {
	newobj, err := convertToSupportedVersion(obj, clusterSpec, setDefaultValuesInYamls)
	if err != nil {
		logrus.Debugf("Unable to transform object to a supported version : %s.", err)
		if obj.GetObjectKind().GroupVersionKind().Version == core.SchemeGroupVersion.Version {
			newobj, err = ConvertToPreferredVersion(obj, clusterSpec, setDefaultValuesInYamls)
			if err != nil {
				logrus.Warnf("Unable to convert (%+v) to preferred version : %s", obj.GetObjectKind(), err)
				newobj = obj
			}
		} else {
			logrus.Debugf("Returning obj in original version : %+v", obj.GetObjectKind())
			newobj = obj
		}
	}
	return newobj, nil
}

// ConvertToSupportedVersion converts obj to a supported Version
func convertToSupportedVersion(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, setDefaultValuesInYamls bool) (newobj runtime.Object, err error) {
	objgvk := obj.GetObjectKind().GroupVersionKind()
	objgv := objgvk.GroupVersion()
	kind := objgvk.Kind
	logrus.Debugf("Converting %s to supported version", kind)
	versions := clusterSpec.GetSupportedVersions(kind)
	if len(versions) == 0 {
		return nil, fmt.Errorf("kind %s unsupported in target cluster : %+v", kind, obj.GetObjectKind())
	}
	logrus.Debugf("Supported Versions : %+v", versions)
	if kind == common.ServiceKind && objgv.Group == knativev1.SchemeGroupVersion.Group {
		return obj, nil
	}
	for _, v := range versions {
		gv, err := schema.ParseGroupVersion(v)
		if err != nil {
			logrus.Debugf("Unable to parse group version %s : %s", v, err)
			continue
		}
		if kind == common.ServiceKind && gv.Group == knativev1.SchemeGroupVersion.Group {
			continue
		}
		newobj, err := ConvertToVersion(obj, gv)
		if err != nil {
			logrus.Debugf("Unable to convert : %s", err)
			continue
		}
		if setDefaultValuesInYamls {
			scheme.Default(newobj)
		}
		return newobj, err
	}
	if setDefaultValuesInYamls {
		scheme.Default(obj)
	}
	return obj, fmt.Errorf("unable to convert to a supported version : %+v", obj.GetObjectKind())
}

// ConvertToPreferredVersion converts obj to a preferred Version
func ConvertToPreferredVersion(obj runtime.Object, clusterSpec collecttypes.ClusterMetadataSpec, setDefaultValuesInYamls bool) (newobj runtime.Object, err error) {
	objgvk := obj.GetObjectKind().GroupVersionKind()
	objgv := objgvk.GroupVersion()
	kind := objgvk.Kind
	logrus.Debugf("Converting %s to preferred version", kind)
	groups := []string{}
	vs := clusterSpec.APIKindVersionMap[kind]
	for _, v := range vs {
		gv, err := schema.ParseGroupVersion(v)
		if err != nil {
			logrus.Debugf("Unable to parse group version %s : %s", v, err)
			continue
		}
		groups = common.AppendIfNotPresent(groups, gv.Group)
	}
	groups = append(groups, objgv.Group)
	for _, g := range groups {
		versions := scheme.PrioritizedVersionsForGroup(g)
		if kind == common.ServiceKind && objgv.Group == knativev1.SchemeGroupVersion.Group {
			return obj, nil
		}
		for _, v := range versions {
			newobj, err := ConvertToVersion(obj, v)
			if err != nil {
				logrus.Debugf("Unable to convert : %s", err)
				continue
			}
			if setDefaultValuesInYamls {
				scheme.Default(newobj)
			}
			return newobj, err
		}
	}
	if setDefaultValuesInYamls {
		scheme.Default(obj)
	}
	return obj, fmt.Errorf("unable to convert to a preferred version : %+v", obj.GetObjectKind())
}

// ConvertToVersion converts objects to a version
func ConvertToVersion(obj runtime.Object, dgv schema.GroupVersion) (newobj runtime.Object, err error) {
	logrus.Debugf("Attempting to convert %s to %s", obj.GetObjectKind().GroupVersionKind(), dgv)
	objvk := obj.GetObjectKind().GroupVersionKind()
	objgv := objvk.GroupVersion()
	kind := objvk.Kind
	newobj, err = checkAndConvertToVersion(obj, dgv)
	if err == nil {
		return newobj, nil
	}
	logrus.Debugf("Unable to do direct transformation : %s", err)
	akt := liasonscheme.AllKnownTypes()
	for kt := range akt {
		if kind != kt.Kind {
			continue
		}
		kobj := obj
		if objgv.Group != dgv.Group {
			igv := schema.GroupVersion{Group: objgv.Group, Version: core.SchemeGroupVersion.Version}
			logrus.Debugf("Attempting conversion of %s obj to %s", objgv, igv)
			iobj, err := checkAndConvertToVersion(obj, igv)
			if err != nil {
				logrus.Debugf("Unable to convert to unversioned object : %s", err)
			} else {
				kobj = iobj
			}
		}
		logrus.Debugf("Attempting conversion of %s obj to %s", obj.GetObjectKind().GroupVersionKind(), kt)
		iobj, err := checkAndConvertToVersion(kobj, kt.GroupVersion())
		if err != nil {
			logrus.Debugf("Unable to convert to unversioned object : %s", err)
			continue
		} else {
			kobj = iobj
		}
		logrus.Debugf("Converted %s obj to %s", objgv, kt)
		newobj, err = checkAndConvertToVersion(kobj, dgv)
		if err == nil {
			return newobj, nil
		}
		logrus.Debugf("Unable to convert through unversioned object : %s", kt)
	}
	err = fmt.Errorf("unable to do convert %s to %s", objgv, dgv)
	logrus.Debugf("%s", err)
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
		iobj := obj
		if obj.GetObjectKind().GroupVersionKind().Group != kt.Group {
			igv := schema.GroupVersion{Group: iobj.GetObjectKind().GroupVersionKind().Group, Version: core.SchemeGroupVersion.Version}
			logrus.Debugf("Attempting conversion of %s obj to %s", iobj.GetObjectKind().GroupVersionKind(), igv)
			eobj, err := checkAndConvertToVersion(iobj, igv)
			if err != nil {
				logrus.Debugf("Unable to convert to unversioned object : %s", err)
			} else {
				iobj = eobj
			}
		}
		logrus.Debugf("Attempting conversion of %s obj to %s", iobj.GetObjectKind().GroupVersionKind(), kt)
		iobj, err := checkAndConvertToVersion(iobj, kt.GroupVersion())
		if err != nil {
			logrus.Debugf("Unable to convert to unversioned object : %s", err)
			continue
		}
		return iobj, nil
	}
	return obj, fmt.Errorf("unable to find liason version for kind %s", kind)
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
		logrus.Debugf("Unable to convert from %T to %T : %s", in, out, err)
	}
	return err
}

// ConvertToV1PodSpec podspec to v1 pod spec
func ConvertToV1PodSpec(podSpec *core.PodSpec) corev1.PodSpec {
	vPodSpec := corev1.PodSpec{}
	err := convertBetweenObjects(podSpec, &vPodSpec)
	if err != nil {
		logrus.Errorf("Unable to convert PodSpec to versioned PodSpec : %s", err)
	}
	return vPodSpec
}

// ConvertToPodSpec podspec to core pod spec
func ConvertToPodSpec(podspec *corev1.PodSpec) core.PodSpec {
	uvPodSpec := core.PodSpec{}
	err := convertBetweenObjects(podspec, &uvPodSpec)
	if err != nil {
		logrus.Errorf("Unable to convert versioned PodSpec to unversioned PodSpec : %s", err)
	}
	return uvPodSpec
}
