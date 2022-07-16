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

// SpringBootConfig stores spring boot related configuration information
type SpringBootConfig struct {
	SpringBootVersion      string             `yaml:"springBootVersion,omitempty" json:"springBootVersion,omitempty"`
	SpringBootAppName      string             `yaml:"springBootAppName,omitempty" json:"springBootAppName,omitempty"`
	SpringBootProfiles     *[]string          `yaml:"springBootProfiles,omitempty" json:"springBootProfiles,omitempty"`
	SpringBootProfilePorts map[string][]int32 `yaml:"springBootProfilePorts,omitempty" json:"springBootProfilePorts,omitempty"`
}

const (
	// SpringBootConfigType stores the springboot config
	SpringBootConfigType transformertypes.ConfigType = "SpringBoot"
)

// Merge implements the Config interface allowing artifacts to be merged
func (sb *SpringBootConfig) Merge(newsbobj interface{}) bool {
	newsbptr, ok := newsbobj.(*SpringBootConfig)
	if !ok {
		newsb, ok := newsbobj.(SpringBootConfig)
		if !ok {
			logrus.Error("Unable to cast to SpringBootConfig for merge")
			return false
		}
		newsbptr = &newsb
	}
	if sb.SpringBootAppName != newsbptr.SpringBootAppName {
		return false
	}
	if sb.SpringBootVersion == "" {
		sb.SpringBootVersion = newsbptr.SpringBootVersion
	}
	if sb.SpringBootVersion != newsbptr.SpringBootVersion {
		logrus.Errorf("Incompatible springboot version found during merge for app %s", sb.SpringBootAppName)
	}
	*sb.SpringBootProfiles = common.MergeSlices(*sb.SpringBootProfiles, *newsbptr.SpringBootProfiles)
	// merge profile ports
	if sb.SpringBootProfilePorts == nil {
		sb.SpringBootProfilePorts = map[string][]int32{}
	}
	for profile, ports := range newsbptr.SpringBootProfilePorts {
		if origPorts, ok := sb.SpringBootProfilePorts[profile]; ok {
			sb.SpringBootProfilePorts[profile] = common.MergeSlices(origPorts, ports)
			continue
		}
		sb.SpringBootProfilePorts[profile] = ports
	}
	return true
}
