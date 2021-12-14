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

package collection

import (
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/types"
	"github.com/sirupsen/logrus"
)

// ClusterMetadataKind defines the kind of cluster metadata file
const ClusterMetadataKind types.Kind = "ClusterMetadata"

// ClusterMetadata for collect output
type ClusterMetadata struct {
	types.TypeMeta   `yaml:",inline"`
	types.ObjectMeta `yaml:"metadata,omitempty"`
	Spec             ClusterMetadataSpec `yaml:"spec,omitempty"`
}

// Merge merges two ClusterMetadatas
func (c *ClusterMetadata) Merge(nc interface{}) bool {
	newcptr, ok := nc.(*ClusterMetadata)
	if !ok {
		newc, ok := nc.(ClusterMetadata)
		if !ok {
			logrus.Error("Unable to cast to ClusterMetadata for merge")
			return false
		}
		newcptr = &newc
	}
	if newcptr.isEmpty() {
		return true
	}
	if c.isEmpty() {
		c.Kind = newcptr.Kind
		c.Name = newcptr.Name
	} else if c.Kind != newcptr.Kind {
		// If neither metadata is empty then their kinds should match
		return false
	}
	if newcptr.Name != "" {
		c.Name = newcptr.Name
	}
	if newcptr.ObjectMeta.Name != c.ObjectMeta.Name {
		return false
	}
	c.ObjectMeta.Labels = common.MergeStringMaps(c.ObjectMeta.Labels, newcptr.ObjectMeta.Labels)
	return c.Spec.Merge(newcptr.Spec)
}

func (c *ClusterMetadata) isEmpty() bool {
	return c.Kind == ""
}

// ClusterMetadataSpec stores the data
type ClusterMetadataSpec struct {
	StorageClasses    []string            `yaml:"storageClasses"`
	APIKindVersionMap map[string][]string `yaml:"apiKindVersionMap"` //[kubernetes kind]["gv1", "gv2",...,"gvn"] prioritized group-version
	Host              string              `yaml:"host,omitempty"`    // Optional field, either collected with move2kube collect or by asking the user.
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
	if len(c.StorageClasses) == 0 {
		c.StorageClasses = []string{"default"}
	}
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

// GetSupportedVersions returns all the group version supported for the kind in this cluster
func (c *ClusterMetadataSpec) GetSupportedVersions(kind string) []string {
	if gvList, ok := c.APIKindVersionMap[kind]; ok {
		if len(gvList) > 0 {
			return gvList
		}
	}
	return nil
}

// NewClusterMetadata creates a new cluster metadata instance
func NewClusterMetadata(contextName string) ClusterMetadata {
	return ClusterMetadata{
		TypeMeta: types.TypeMeta{
			Kind:       string(ClusterMetadataKind),
			APIVersion: types.SchemeGroupVersion.String(),
		},
		ObjectMeta: types.ObjectMeta{
			Name: contextName,
		},
		Spec: ClusterMetadataSpec{
			StorageClasses:    []string{},
			APIKindVersionMap: map[string][]string{},
		},
	}
}
