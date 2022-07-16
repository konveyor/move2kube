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

package artifacts

import (
	"github.com/konveyor/move2kube/common"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/sirupsen/logrus"
)

// NewImagesArtifactType represents New Image Artifact Type
const NewImagesArtifactType transformertypes.ArtifactType = "NewImages"

// NewImagesConfigType represents New Image Config type
const NewImagesConfigType transformertypes.ConfigType = "NewImages"

// NewImages represents the strut having configuration about new images
type NewImages struct {
	ImageNames []string `yaml:"imageNames" json:"imageNames"`
}

// Merge implements the Config interface allowing artifacts to be merged
func (ni *NewImages) Merge(newniobj interface{}) bool {
	newniptr, ok := newniobj.(*NewImages)
	if !ok {
		newni, ok := newniobj.(NewImages)
		if !ok {
			logrus.Error("Unable to cast to NewImages for merge")
			return false
		}
		newniptr = &newni
	}
	ni.ImageNames = common.MergeSlices(ni.ImageNames, newniptr.ImageNames)
	return true
}
