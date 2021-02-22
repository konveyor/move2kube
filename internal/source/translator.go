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

	"github.com/konveyor/move2kube/internal/containerizer"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

// Translator interface defines translator that translates files and converts it to ir representation
type Translator interface {
	GetServiceOptions(p plantypes.Plan) (map[string]plantypes.Service, error)
	Translate(serviceName string, plan plantypes.Plan) (irtypes.IR, error)
}

// GetTranslators returns translator for given format
func GetTranslators() []Translator {
	var l = []Translator{new(DockerfileTranslator), new(ComposeTranslator), new(CfManifestTranslator), new(Any2KubeTranslator)} //Any2Kube should be the last option
	return l
}

// UpdateServiceOptions looks identify source artifacts and generation options
func UpdateServiceOptions(p plantypes.Plan) {
	log.Infoln("Planning Translation")
	for _, l := range GetTranslators() {
		log.Infof("[%T] Planning translation", l)
		services, err := l.GetServiceOptions(p)
		if err != nil {
			log.Warnf("[%T] Failed : %s", l, err)
		} else {
			p.AddServicesToPlan(services)
			log.Infof("[%T] Done", l)
		}
	}
}

// Translate loads all sources
func Translate(p plantypes.Plan) (irtypes.IR, error) {
	ir := irtypes.NewIR(p)
	ts := GetTranslators()
	log.Infoln("Begin Translation")
	for sn, s := range p.Spec.Services {
		log.Debugf("Translating service %s", sn)
		if _, ok := ir.Services[sn]; !ok {
			ir.Services[sn] = irtypes.NewServiceWithName(sn)
		}
		log.Debugf("Loading containers")
		for _, l := range ts {
			log.Debugf("Using translator %T for service %s", l, sn)
			currir, err := l.Translate(sn, p)
			if err != nil {
				log.Warnf("[%T] Failed : %s", l, err.Error())
				continue
			}
			log.Infof("[%T] Done for service %s", l, sn)
			ir.Merge(currir)
			log.Debugf("Total Services after translation : %d", len(ir.Services))
			log.Debugf("Total Containers after translation : %d", len(ir.Containers))
		}
		if len(s.GenerationOptions) > 0 {
			containerizer.GetContainer()
		}
		log.Debugf("Translated service %s", sn)
	}
	log.Infoln("Translation done")
	return ir, nil
}
