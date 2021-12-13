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

// MavenConfig stores maven related configuration information
type MavenConfig struct {
	MavenAppName  string        `yaml:"mavenAppName,omitempty" json:"mavenAppName,omitempty"`
	ArtifactType  JavaPackaging `yaml:"artifactType"`
	MavenProfiles []string      `yaml:"mavenProfiles,omitempty" json:"mavenProfiles,omitempty"`
}

const (
	// MavenConfigType stores the maven config
	MavenConfigType transformertypes.ConfigType = "Maven"
	// MavenPomPathType stores the Maven POM file Path
	MavenPomPathType transformertypes.PathType = "MavenPom"
)

// Merge implements the Config interface allowing artifacts to be merged
func (mc *MavenConfig) Merge(newmcobj interface{}) bool {
	newmcptr, ok := newmcobj.(*MavenConfig)
	if !ok {
		newmc, ok := newmcobj.(MavenConfig)
		if !ok {
			logrus.Error("Unable to cast to MavenConfig for merge")
			return false
		}
		newmcptr = &newmc
	}
	if mc.ArtifactType != newmcptr.ArtifactType || mc.MavenAppName != newmcptr.MavenAppName {
		return false
	}
	mc.MavenProfiles = common.MergeStringSlices(mc.MavenProfiles, newmcptr.MavenProfiles...)
	return true
}
