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
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterMetadataKind defines the kind of cluster metadata file
const ClusterMetadataKind types.Kind = "ClusterMetadata"

// ClusterMetadata for collect output
type ClusterMetadata struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty"`
	Spec              ClusterMetadataSpec `yaml:"spec,omitempty"`
}

// ClusterMetadataSpec stores the data
type ClusterMetadataSpec struct {
	Inbuilt           bool                `yaml:"-"`
	StorageClasses    []string            `yaml:"storageClasses"`
	APIKindVersionMap map[string][]string `yaml:"apiKindVersionMap"` //[kubernetes kind]["gv1", "gv2",...,"gvn"] prioritized group-version
	Host              string              `yaml:"host,omitempty"`    // Optional field, either collected with move2kube collect or by asking the user.
}

// Merge helps merge clustermetadata
func (c *ClusterMetadata) Merge(newc ClusterMetadata) bool {
	if newc.isEmpty() {
		return true
	}
	if c.isEmpty() {
		c.Kind = newc.Kind
		c.Name = newc.Name
	} else if c.Kind != newc.Kind {
		// If neither metadata is empty then their kinds should match
		return false
	}
	if newc.Name != "" {
		c.Name = newc.Name
	}

	// Allow only intersection of storage classes
	newslice := []string{}
	for _, sc := range c.Spec.StorageClasses {
		if common.IsStringPresent(newc.Spec.StorageClasses, sc) {
			newslice = append(newslice, sc)
		}
	}
	c.Spec.StorageClasses = newslice
	if len(c.Spec.StorageClasses) == 0 {
		c.Spec.StorageClasses = []string{"default"}
	}
	//TODO: Do Intelligent merge of version
	apiversionkindmap := map[string][]string{}
	for kindname, gvList := range newc.Spec.APIKindVersionMap {
		if _, ok := c.Spec.APIKindVersionMap[kindname]; ok {
			apiversionkindmap[kindname] = gvList
		}
	}
	c.Spec.APIKindVersionMap = apiversionkindmap
	c.Spec.Host = newc.Spec.Host
	return true
}

// Merge helps merge clustermetadata
func (c *ClusterMetadataSpec) Merge(newc ClusterMetadataSpec) bool {
	// Allow only intersection of storage classes
	newslice := []string{}
	for _, sc := range c.StorageClasses {
		if common.IsStringPresent(newc.StorageClasses, sc) {
			newslice = append(newslice, sc)
		}
	}
	c.StorageClasses = newslice
	//TODO: Do Intelligent merge of version
	apiversionkindmap := map[string][]string{}
	for kindname, gvList := range newc.APIKindVersionMap {
		if _, ok := c.APIKindVersionMap[kindname]; ok {
			apiversionkindmap[kindname] = gvList
		}
	}
	c.APIKindVersionMap = apiversionkindmap
	c.Host = newc.Host
	return true
}

func (c *ClusterMetadata) isEmpty() bool {
	return c.Kind == ""
}

// GetSupportedVersions returns all the group version supported for the kind in this cluster
func (c *ClusterMetadataSpec) GetSupportedVersions(kind string) []string {
	if gvList, ok := c.APIKindVersionMap[kind]; ok {
		if len(gvList) > 0 {
			return gvList
		}
	}
	return nil
}

// IsTektonInstalled returns true if Tekton has been installed on this cluster
func (c *ClusterMetadataSpec) IsTektonInstalled() bool {
	return len(c.GetSupportedVersions("Task")) > 0
}

// IsBuildConfigSupported returns true if the cluster is Openshift and has build configs
func (c *ClusterMetadataSpec) IsBuildConfigSupported() bool {
	return len(c.GetSupportedVersions("BuildConfig")) > 0
}

// NewClusterMetadata creates a new cluster metadata instance
func NewClusterMetadata(contextName string) ClusterMetadata {
	return ClusterMetadata{
		TypeMeta: metav1.TypeMeta{
			Kind:       string(ClusterMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: contextName,
		},
		Spec: ClusterMetadataSpec{
			StorageClasses:    []string{},
			APIKindVersionMap: map[string][]string{},
		},
	}
}
