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

package optimize

import (
	log "github.com/sirupsen/logrus"

	irtypes "github.com/konveyor/move2kube/internal/types"
)

// optimizer optimizes the configuration
type optimizer interface {
	optimize(sourceir irtypes.IR) (irtypes.IR, error)
}

// getOptimizers returns loader for given format
func getOptimizers() []optimizer {
	var l = []optimizer{new(ingressOptimizer), new(normalizeCharacterOptimizer), new(replicaOptimizer), new(imagePullPolicyOptimizer), new(portMergeOptimizer)}
	return l
}

// Optimize optimizes the application artifacts
func Optimize(ir irtypes.IR) (irtypes.IR, error) {
	optimizers := getOptimizers()
	log.Infoln("Begin Optimization")
	for _, o := range optimizers {
		log.Debugf("[%T] Begin Optimization", o)
		var err error
		ir, err = o.optimize(ir)
		if err != nil {
			log.Warnf("[%T] Failed : %s", o, err.Error())
		} else {
			log.Debugf("[%T] Done", o)
		}
	}
	log.Infoln("Optimization done")
	return ir, nil
}
