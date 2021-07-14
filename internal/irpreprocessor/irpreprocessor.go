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

package irpreprocessor

import (
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/sirupsen/logrus"
)

// irpreprocessor optimizes the configuration
type irpreprocessor interface {
	preprocess(sourceir irtypes.IR) (irtypes.IR, error)
}

// getIRPreprocessors returns optimizers
func getIRPreprocessors() []irpreprocessor {
	var l = []irpreprocessor{new(mergePreprocessor), new(normalizeCharacterPreprocessor), new(ingressPreprocessor), new(replicaPreprocessor), new(imagePullPolicyPreprocessor), new(registryPreProcessor)}
	return l
}

// Preprocess preprocesses IR before application artifacts are generated
func Preprocess(ir irtypes.IR) (irtypes.IR, error) {
	optimizers := getIRPreprocessors()
	logrus.Infoln("Begin Optimization")
	for _, o := range optimizers {
		logrus.Debugf("[%T] Begin Optimization", o)
		var err error
		ir, err = o.preprocess(ir)
		if err != nil {
			logrus.Warnf("[%T] Failed : %s", o, err.Error())
		} else {
			logrus.Debugf("[%T] Done", o)
		}
	}
	logrus.Infoln("Optimization done")
	return ir, nil
}
