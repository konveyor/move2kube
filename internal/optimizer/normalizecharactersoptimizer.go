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
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"

	irtypes "github.com/konveyor/move2kube/internal/types"
)

// normalizeCharacterOptimizer identifies non-allowed characters and replaces them
type normalizeCharacterOptimizer struct {
}

func (po normalizeCharacterOptimizer) optimize(ir irtypes.IR) (irtypes.IR, error) {
	//TODO: Make this generic to ensure all fields have valid names
	for k := range ir.Services {
		scObj := ir.Services[k]
		for _, serviceContainer := range scObj.Containers {
			var tmpEnvArray []corev1.EnvVar
			for _, env := range serviceContainer.Env {
				if !strings.Contains(env.Name, "affinity") {
					env.Name = strings.Trim(env.Name, "\t \n")
					env.Value = strings.Trim(env.Value, "\t \n")
					tmpString, err := stripQuotation(env.Name)
					if err == nil {
						env.Name = tmpString
					}
					tmpString, err = stripQuotation(env.Value)
					if err == nil {
						env.Value = tmpString
					}
					tmpEnvArray = append(tmpEnvArray, env)
				}
			}
			serviceContainer.Env = tmpEnvArray
		}
		ir.Services[k] = scObj
	}
	return ir, nil
}

func stripQuotation(inputString string) (string, error) {
	regex := regexp.MustCompile(`^[',"](.*)[',"]$`)

	return regex.ReplaceAllString(inputString, `$1`), nil
}
