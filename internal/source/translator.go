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

package source

import (
	log "github.com/sirupsen/logrus"

	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// Translator interface defines translator that translates files and converts it to ir representation
type Translator interface {
	GetTranslatorType() plantypes.TranslationTypeValue
	GetServiceOptions(inputPath string, p plantypes.Plan) ([]plantypes.Service, error)
	Translate(services []plantypes.Service, p plantypes.Plan) (irtypes.IR, error)
	newService(serviceName string) plantypes.Service
}

// GetTranslators returns translator for given format
func GetTranslators() []Translator {
	var l = []Translator{new(DockerfileTranslator), new(ComposeTranslator), new(CfManifestTranslator), new(KnativeTranslator), new(KubeTranslator), new(Any2KubeTranslator)} //Any2Kube should be the last option
	return l
}

// GetAllTranslatorTypes returns all translator types
func GetAllTranslatorTypes() []string {
	translationTypes := []string{}
	for _, tp := range GetTranslators() {
		translationTypes = append(translationTypes, (string)(tp.GetTranslatorType()))
	}
	return translationTypes
}

// Translate loads all sources
func Translate(p plantypes.Plan) (irtypes.IR, error) {
	ts := GetTranslators()
	ir := irtypes.NewIR(p)
	log.Infoln("Begin Translation")
	for _, l := range ts {
		log.Infof("[%T] Begin translation", l)
		validservices := []plantypes.Service{}
		for _, services := range p.Spec.Inputs.Services {
			//Choose the first service even if there are multiple options
			service := services[0]
			if service.TranslationType == l.GetTranslatorType() {
				validservices = append(validservices, service)
			}
		}
		log.Debugf("Services to translate : %d", len(validservices))
		currir, err := l.Translate(validservices, p)
		log.Debugf("Services translated : %d", len(currir.Services))
		log.Debugf("Containers translated : %d", len(currir.Containers))
		if err != nil {
			log.Warnf("[%T] Failed : %s", l, err.Error())
			continue
		}
		log.Infof("[%T] Done", l)
		ir.Merge(currir)
		log.Debugf("Total Services after translation : %d", len(ir.Services))
		log.Debugf("Total Containers after translation : %d", len(ir.Containers))
	}
	log.Infoln("Translation done")

	return ir, nil
}
