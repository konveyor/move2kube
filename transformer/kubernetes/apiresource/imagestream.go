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
	"fmt"

	"github.com/konveyor/move2kube-wasm/common"
	collecttypes "github.com/konveyor/move2kube-wasm/types/collection"
	irtypes "github.com/konveyor/move2kube-wasm/types/ir"
	okdimagev1 "github.com/openshift/api/image/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

//TODO: Add support for ImageStreamImage, ImageStreamImport, ImageStreamLayers, ImageStreamMapping, ImageStreamTag

const (
	// imageStreamKind defines the image stream kind
	imageStreamKind = "ImageStream"
)

// ImageStream handles all objects related to image stream
type ImageStream struct {
}

// getSupportedKinds returns kinds supported by ImageStream
func (*ImageStream) getSupportedKinds() []string {
	return []string{imageStreamKind}
}

// createNewResources converts IR to runtime objects
func (imageStream *ImageStream) createNewResources(ir irtypes.EnhancedIR, supportedKinds []string, targetCluster collecttypes.ClusterMetadata) []runtime.Object {
	objs := []runtime.Object{}
	if !common.IsPresent(supportedKinds, imageStreamKind) {
		logrus.Debugf("Could not find a valid resource type in cluster to create an ImageStream")
		return objs
	}
	// Create an imagestream for each image that we are using
	for in, irContainer := range ir.ContainerImages {
		imageStreamName, imageStreamTag := imageStream.GetImageStreamNameAndTag(in)
		imageStream := imageStream.createImageStream(imageStreamName, imageStreamTag, in, irContainer, ir)
		objs = append(objs, &imageStream)
	}
	return objs
}

// GetImageStreamNameAndTag gives the image stream name and tag given the image name.
func (*ImageStream) GetImageStreamNameAndTag(fullImageName string) (string, string) {
	imageName, tag := common.GetImageNameAndTag(fullImageName)
	imageStreamName := fmt.Sprintf("%s-%s", imageName, tag)
	imageStreamName = common.MakeStringDNSSubdomainNameCompliant(imageStreamName)
	return imageStreamName, tag
}

func (*ImageStream) createImageStream(name, tag string, imageName string, irContainer irtypes.ContainerImage, ir irtypes.EnhancedIR) okdimagev1.ImageStream {
	tags := []okdimagev1.TagReference{
		{
			From: &corev1.ObjectReference{
				Kind: "DockerImage",
				Name: imageName,
			},
			Name: tag,
		},
	}
	imageStream := okdimagev1.ImageStream{
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
	return imageStream
}

// convertToClusterSupportedKinds converts kinds to cluster supported kinds
func (imageStream *ImageStream) convertToClusterSupportedKinds(obj runtime.Object, supportedKinds []string, otherobjs []runtime.Object, _ irtypes.EnhancedIR, targetCluster collecttypes.ClusterMetadata) ([]runtime.Object, bool) {
	if common.IsPresent(imageStream.getSupportedKinds(), obj.GetObjectKind().GroupVersionKind().Kind) {
		return []runtime.Object{obj}, true
	}
	return nil, false
}
