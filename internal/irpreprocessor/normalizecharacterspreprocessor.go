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

package irpreprocessor

import (
	"regexp"
	"strings"

	irtypes "github.com/konveyor/move2kube/types/ir"
	core "k8s.io/kubernetes/pkg/apis/core"
)

// normalizeCharacterOptimizer identifies non-allowed characters and replaces them
type normalizeCharacterPreprocessor struct {
}

func (po normalizeCharacterPreprocessor) preprocess(ir irtypes.IR) (irtypes.IR, error) {
	//TODO: Make this generic to ensure all fields have valid names
	for i := range ir.Services {
		for j := range ir.Services[i].Containers {
			var tmpEnvArray []core.EnvVar
			for _, env := range ir.Services[i].Containers[j].Env {
				if !strings.Contains(env.Name, "affinity") {
					env.Name = stripQuotation(strings.TrimSpace(env.Name))
					env.Value = stripQuotation(strings.TrimSpace(env.Value))
					tmpEnvArray = append(tmpEnvArray, env)
				}
			}
			ir.Services[i].Containers[j].Env = tmpEnvArray
		}
	}
	return ir, nil
}

func stripQuotation(inputString string) string {
	//TODO: check if regex is correct
	regex := regexp.MustCompile(`^[',"](.*)[',"]$`)

	return regex.ReplaceAllString(inputString, `$1`)
}
