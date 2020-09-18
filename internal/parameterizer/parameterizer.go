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

package parameterize

import (
	log "github.com/sirupsen/logrus"

	irtypes "github.com/konveyor/move2kube/internal/types"
)

// Parameterizer paramertizers the configuration for helm
type Parameterizer interface {
	parameterize(ir *irtypes.IR) error
}

// getParameterizers returns different supported paramterizers
func getParameterizers() []Parameterizer {
	return []Parameterizer{new(imageNameParameterizer), new(storageClassParameterizer)}
}

// Parameterize parameterizes for usage as a helm chart
func Parameterize(ir irtypes.IR) (irtypes.IR, error) {
	var parameterizers = getParameterizers()
	log.Infoln("Begin Parameterization")
	for _, p := range parameterizers {
		log.Debugf("[%T] Begin Parameterization", p)
		//TODO: Handle conflicting service names and invalid characters in objects
		err := p.parameterize(&ir)
		if err != nil {
			log.Warnf("[%T] Failed : %s", p, err.Error())
		} else {
			log.Debugf("[%T] Done", p)
		}
	}
	log.Infoln("Parameterization done")
	return ir, nil
}
