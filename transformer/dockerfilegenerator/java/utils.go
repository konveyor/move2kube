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

package java

import (
	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

const (
	defaultAppPathInContainer = "/app/"
	defaultJavaVersion        = "17"
	defaultJavaPackage        = "java-17-openjdk-devel"
)

func getJavaPackage(mappingFile string, version string) (pkg string, err error) {
	var javaPackageNamesMapping JavaPackageNamesMapping
	if err := common.ReadMove2KubeYaml(mappingFile, &javaPackageNamesMapping); err != nil {
		logrus.Debugf("Could not load mapping at %s", mappingFile)
		return "", err
	}
	v, ok := javaPackageNamesMapping.Spec.PackageVersions[version]
	if !ok {
		logrus.Infof("Matching java package not found for java version : %s. Going with default.", version)
		return defaultJavaPackage, nil
	}
	return v, nil
}
