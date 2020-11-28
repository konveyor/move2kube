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
	okdimagev1 "github.com/openshift/api/image/v1"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/konveyor/move2kube/internal/common"
	irtypes "github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
)

//TODO: Add support for Build, BuildConfig, ImageStreamImage, ImageStreamImport, ImageStreamLayers, ImageStreamMapping, ImageStreamTag, tekton

const (
	// imageStreamKind defines the image stream kind
	imageStreamKind = "ImageStream"
)

// ImageStream handles all objects related to image stream
type ImageStream struct {
	Cluster collecttypes.ClusterMetadataSpec
}

// GetSupportedKinds returns kinds supported by ImageStream
func (d *ImageStream) GetSupportedKinds() []string {
	return []string{imageStreamKind}
}

// CreateNewResources converts IR to runtime objects
func (d *ImageStream) CreateNewResources(ir irtypes.IR, supportedKinds []string) []runtime.Object {
	objs := []runtime.Object{}

	for _, service := range ir.Services {
		if common.IsStringPresent(supportedKinds, imageStreamKind) {
			newobjs := d.createImageStream(service.Name, service)
			objs = append(objs, newobjs...)
		} else {
			log.Debugf("Could not find a valid resource type in cluster to create a ImageStream")
		}
	}
	return objs
}

// ConvertToClusterSupportedKinds converts kinds to cluster supported kinds
func (d *ImageStream) ConvertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object) ([]runtime.Object, bool) {
	if common.IsStringPresent(supportedKinds, imageStreamKind) {
		if _, ok := obj.(*okdimagev1.ImageStream); ok {
			return []runtime.Object{obj}, true
		}
	}
	return nil, false
}

// createImageStream creates a ImageStream object
func (d *ImageStream) createImageStream(name string, service irtypes.Service) []runtime.Object {
	imageStreams := []runtime.Object{}
	for _, serviceContainer := range service.Containers {
		if serviceContainer.Image == "" {
			serviceContainer.Image = name
		}

		var tags []okdimagev1.TagReference
		_, tag := common.GetImageNameAndTag(serviceContainer.Image)
		tags = []okdimagev1.TagReference{
			{
				From: &corev1.ObjectReference{
					Kind: "DockerImage",
					Name: serviceContainer.Image,
				},
				Name: tag,
			},
		}

		is := &okdimagev1.ImageStream{
			TypeMeta: metav1.TypeMeta{
				Kind:       imageStreamKind,
				APIVersion: okdimagev1.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: getServiceLabels(name),
			},
			Spec: okdimagev1.ImageStreamSpec{Tags: tags},
		}
		imageStreams = append(imageStreams, is)
	}
	return imageStreams
}
