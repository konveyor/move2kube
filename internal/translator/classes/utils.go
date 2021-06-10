/*
Copyright IBM Corporation 2021

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

package classes

import (
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
)

func setTranslatorInfo(translatorConfig translatortypes.Translator, namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator) (map[string]plantypes.Service, []plantypes.Translator) {
	for i, s := range namedServices {
		for j := range s {
			namedServices[i][j].Name = translatorConfig.Name
		}
	}
	for i := range unnamedServices {
		unnamedServices[i].Name = translatorConfig.Name
	}
	return namedServices, unnamedServices
}
