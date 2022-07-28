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

package dockerfilegenerator

import (
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// getNodeVersion returns the Node version to be used for the service
func getNodeVersion(versionConstraint, defaultNodejsVersion string, supportedVersions []string) string {
	v1, err := version.NewVersion(versionConstraint)
	if err == nil {
		logrus.Debugf("the constraint is a Node version: %#v", v1)
		return "v" + v1.String()
	}
	constraints, err := version.NewConstraint(versionConstraint)
	if err != nil {
		logrus.Errorf("failed to parse the Node version constraint string. Error: %q Actual: %s", err, versionConstraint)
		return defaultNodejsVersion
	}
	logrus.Debugf("Node version constraints len = %d; constraints =  %#v", constraints.Len(), constraints.String())
	for _, supportedVersion := range supportedVersions {
		ver, _ := version.NewVersion(supportedVersion)
		if constraints.Check(ver) {
			logrus.Debugf("%#v satisfies constraints %#v\n", ver, constraints)
			return supportedVersion
		}
	}
	logrus.Infof("no supported Node version detected in package.json. Selecting default Node version- %s", defaultNodejsVersion)
	return defaultNodejsVersion
}
