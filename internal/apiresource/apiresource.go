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
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	"github.com/konveyor/move2kube/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	selector = "io." + types.AppName + ".service"
)

// IAPIResource deines the interface to be deined for a new api resource
type IAPIResource interface {
	GetSupportedKinds() []string
	CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object
	ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) // Return nil if not supported
}

// APIResource defines functions that are reusable across the api resources
type APIResource struct {
	IAPIResource
	cluster    collecttypes.ClusterMetadataSpec
	cachedobjs []runtime.Object
}

// SetClusterContext sets the cluster context
func (o *APIResource) SetClusterContext(cluster collecttypes.ClusterMetadataSpec) {
	o.cluster = cluster
}

// LoadResources loads the resources
func (o *APIResource) LoadResources(objs []runtime.Object) []runtime.Object { //Returns resources it could not handle
	ignoredResources := []runtime.Object{}
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		if !o.loadResource(obj, objs) {
			ignoredResources = append(ignoredResources, obj)
		}
	}
	return ignoredResources
}

// GetUpdatedResources converts IR to a runtime object
func (o *APIResource) GetUpdatedResources(ir irtypes.IR) []runtime.Object {
	objs := o.CreateNewResources(ir, o.getClusterSupportedKinds())
	for _, obj := range objs {
		if !o.loadResource(obj, objs) {
			log.Errorf("Object created seems to be of an incompatible type : %+v", obj)
		}
	}
	return o.cachedobjs
}

func (o *APIResource) isSupportedKind(obj runtime.Object) bool {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	return common.IsStringPresent(o.GetSupportedKinds(), kind)
}

func (o *APIResource) loadResource(obj runtime.Object, otherobjs []runtime.Object) bool { //Returns resources it could not handle
	if !o.isSupportedKind(obj) {
		return false
	}
	supportedobjs, done := o.ConvertToClusterSupportedKinds(obj, o.getClusterSupportedKinds(), otherobjs)
	if !done {
		return false
	}
	if o.cachedobjs == nil {
		o.cachedobjs = supportedobjs
		return true
	}
	for _, supportedobj := range supportedobjs {
		objs := []runtime.Object{}
		merged := false
		for _, cachedobj := range o.cachedobjs {
			var mergedobj runtime.Object = nil
			if o.isSameResource(cachedobj, supportedobj) {
				mergedobj = o.merge(cachedobj, supportedobj)
			}
			if mergedobj != nil {
				objs = append(objs, mergedobj)
				merged = true
			} else {
				objs = append(objs, cachedobj)
			}
		}
		if !merged {
			objs = append(objs, supportedobj)
		}
		o.cachedobjs = objs
	}
	return true
}

// Could be different versions, but will still be marked as duplicate
func (o *APIResource) isSameResource(obj1 runtime.Object, obj2 runtime.Object) bool {
	if o.shareSameID(obj1, obj2) && obj1.GetObjectKind().GroupVersionKind().GroupKind().Empty() != obj2.GetObjectKind().GroupVersionKind().GroupKind().Empty() {
		return true
	}
	return false
}

func (o *APIResource) shareSameID(obj1 runtime.Object, obj2 runtime.Object) bool {
	obj1id := o.getObjectID(obj1)
	obj2id := o.getObjectID(obj2)
	if obj1id == "" || obj2id == "" || obj1id != obj2id {
		return false
	}
	return true
}

func getServiceLabels(name string) map[string]string {
	return map[string]string{selector: name}
}

// getAnnotations configures annotations
func getAnnotations(service irtypes.Service) map[string]string {

	annotations := map[string]string{}
	for key, value := range service.Annotations {
		annotations[key] = value
	}

	return annotations
}

func (o *APIResource) merge(obj1 runtime.Object, obj2 runtime.Object) runtime.Object {
	if obj1.GetObjectKind().GroupVersionKind().Kind == obj2.GetObjectKind().GroupVersionKind().Kind {
		reflect.ValueOf(obj2).MethodByName("DeepCopyInto").Call([]reflect.Value{reflect.ValueOf(obj1)})
		return obj1
	}
	return nil
}

func (o *APIResource) getObjectID(obj runtime.Object) string {
	k8sObjValue := reflect.ValueOf(obj).Elem()
	objMeta, ok := k8sObjValue.FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
	if !ok {
		log.Errorf("Failed to retrieve object metadata")
	}
	return objMeta.GetNamespace() + objMeta.GetName()
}

func (o *APIResource) getClusterSupportedKinds() []string {
	kinds := o.IAPIResource.GetSupportedKinds()
	supportedKinds := []string{}
	for _, kind := range kinds {
		if o.cluster.GetSupportedVersions(kind) != nil {
			supportedKinds = append(supportedKinds, kind)
		}
	}
	return supportedKinds
}

func getPodLabels(name string, networks []string) map[string]string {
	labels := getServiceLabels(name)
	networklabels := getNetworkPolicyLabels(networks)
	return common.MergeStringMaps(labels, networklabels)
}
