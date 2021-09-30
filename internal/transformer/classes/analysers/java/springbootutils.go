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
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/source/springboot"
	"github.com/magiconair/properties"
	"github.com/sirupsen/logrus"
)

const (
	springbootAppNameConfig = "spring.application.name"
)

func getSpringBootInfo(dir string) (appName string, bootstrapFiles []string, bootstrapYamlFiles []string, appPropFiles []string, appYamlPropFiles []string) {
	bootstrapFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), []string{"bootstrap.properties"}, nil)
	if len(bootstrapFiles) != 0 {
		props, err := properties.LoadFiles(bootstrapFiles, properties.UTF8, false)
		if err != nil {
			logrus.Errorf("Unable to read bootstrap properties : %s", err)
		} else {
			appName = props.GetString(springbootAppNameConfig, "")
		}
	}
	bootstrapYamlFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), nil, []string{"bootstrap.ya?ml"})
	if len(bootstrapFiles) != 0 {
		if appName != "" {
			for _, bootstrapYamlFile := range bootstrapYamlFiles {
				bootstrapYaml := springboot.SpringApplicationYaml{}
				err := common.ReadYaml(bootstrapYamlFile, &bootstrapYaml)
				if err != nil {
					logrus.Errorf("Unable to read bootstrap yaml properties : %s", err)
				} else {
					if bootstrapYaml.Spring.SpringApplication.Name != "" {
						appName = bootstrapYaml.Spring.SpringApplication.Name
						break
					}
				}
			}
		}
	}

	appPropFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), nil, []string{"application(-.+)?.properties"})
	if len(appPropFiles) != 0 {
		if appName == "" {
			props, err := properties.LoadFiles(appPropFiles, properties.UTF8, false)
			if err != nil {
				logrus.Errorf("Unable to read springboot application properties : %s", err)
			} else {
				appName = props.GetString(springbootAppNameConfig, "")
			}
		}
	}
	appYamlPropFiles, _ = common.GetFilesByName(filepath.Join(dir, defaultResourcesPath), nil, []string{"application(-.+)?.ya?ml"})
	if len(appYamlPropFiles) != 0 {
		if appName != "" {
			for _, appYamlFile := range appYamlPropFiles {
				appYaml := springboot.SpringApplicationYaml{}
				err := common.ReadYaml(appYamlFile, &appYaml)
				if err != nil {
					logrus.Errorf("Unable to read bootstrap yaml properties : %s", err)
				} else {
					if appYaml.Spring.SpringApplication.Name != "" {
						appName = appYaml.Spring.SpringApplication.Name
						break
					}
				}
			}
		}
	}
	return
}
