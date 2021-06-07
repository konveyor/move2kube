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

package collection

import (
	"github.com/konveyor/move2kube/types"
)

// ImageMetadataKind defines kind for imagemetadata file
const ImageMetadataKind types.Kind = "ImageMetadata"

// ImageInfo stores data about different images
type ImageInfo struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             ImageInfoSpec `yaml:"spec,omitempty"`
}

// ImageInfoSpec defines the data stored about ImageInfo
type ImageInfoSpec struct {
	Tags          []string `yaml:"tags"`
	PortsToExpose []int    `yaml:"ports"`
	AccessedDirs  []string `yaml:"accessedDirs"`
	UserID        int      `yaml:"userID"`
}

// NewImageInfo creates a new imageinfo instance
func NewImageInfo() ImageInfo {
	return ImageInfo{
		TypeMeta: types.TypeMeta{
			Kind:       string(ImageMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}
