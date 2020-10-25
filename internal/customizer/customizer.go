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

package customizer

import (
	irtypes "github.com/konveyor/move2kube/internal/types"
	log "github.com/sirupsen/logrus"
)

//Customizer paramertizers the configuration
type customizer interface {
	customize(ir *irtypes.IR) error
}

//GetCustomizers gets the customizers registered with it
func getCustomizers() []customizer {
	return []customizer{new(registryCustomizer), new(storageCustomizer), new(ingressCustomizer)}
}

//Customize invokes the customizes based on the customizer options
func Customize(ir irtypes.IR) (irtypes.IR, error) {
	var customizers = getCustomizers()
	log.Infoln("Begin Customization")
	for _, c := range customizers {
		log.Debugf("[%T] Begin Customization", c)
		err := c.customize(&ir)
		if err != nil {
			log.Warnf("[%T] Failed : %s", c, err.Error())
		} else {
			log.Debugf("[%T] Done", c)
		}
	}
	log.Infoln("Customization done")
	return ir, nil
}
