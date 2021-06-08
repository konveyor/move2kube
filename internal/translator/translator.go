/*
Copyright IBM Corporation 2020, 2021

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

package translator

import (
	"reflect"

	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/translator/classes"
	"github.com/konveyor/move2kube/qaengine"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
)

var (
	translatorTypes map[string]reflect.Type = make(map[string]reflect.Type)
	translators     map[string]Translator   = make(map[string]Translator)
)

// Translator interface defines translator that translates files and converts it to ir representation
type Translator interface {
	Init(tc translatortypes.Translator) error
	GetConfig() translatortypes.Translator

	BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error)
	PlanDetect(plantypes.Plan) ([]plantypes.Translator, error)

	Translate(serviceName string) map[string]translatortypes.Patch
}

func init() {
	translatorObjs := []Translator{new(classes.GoInterface)}
	for _, tt := range translatorObjs {
		t := reflect.TypeOf(tt).Elem()
		tn := t.Name()
		if ot, ok := translatorTypes[tn]; ok {
			log.Errorf("Two translator classes have the same name %s : %T, %T; Ignoring %T", tn, ot, t, t)
			continue
		}
		translatorTypes[tn] = t
	}
}

func Init(assetsPath string) error {
	filePaths, err := common.GetFilesByExt(assetsPath, []string{".yml", ".yaml"})
	if err != nil {
		log.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", assetsPath, err)
		return err
	}
	translatorConfigs := make(map[string]translatortypes.Translator)
	for _, filePath := range filePaths {
		tc, err := getTranslatorConfig(filePath)
		if err != nil {
			log.Debugf("Unable to load %s as Translator config", filePath, err)
			continue
		}
		if ot, ok := translatorConfigs[tc.Name]; ok {
			log.Errorf("Found two conflicting translator Names %s : %s, %s. Ignoring %s.", tc.Name, ot.Spec.FilePath, tc.Spec.FilePath)
			continue
		}
		if _, ok := translatorTypes[tc.Spec.Class]; ok {
			translatorConfigs[tc.Name] = tc
			continue
		}
		log.Errorf("Unable to find suitable translator class (%s) for translator config at %s", tc.Spec.Class, filePath)
	}
	tns := make([]string, 0)
	for tn := range translatorConfigs {
		tns = append(tns, tn)
	}
	translatorNames := qaengine.FetchMultiSelectAnswer(common.ConfigTranslatorTypesKey, "Select all translator types that you are interested in:", []string{"Services that don't support any of the translator types you are interested in will be ignored."}, tns, tns)
	for _, tn := range translatorNames {
		tc := translatorConfigs[tn]
		if c, ok := translatorTypes[tc.Spec.Class]; !ok {
			log.Errorf("Unable to find Translator class %s in %+v", tc.Spec.Class, translatorTypes)
		} else {
			t := reflect.New(c).Interface().(Translator)
			if err := t.Init(tc); err != nil {
				log.Errorf("Unable to initialize translator %s : %s", tc.Name, err)
			} else {
				translators[tn] = t
			}
		}
	}
	return nil
}

func GetTranslators() map[string]Translator {
	return translators
}

func GetServices(prjName string, dir string) (services map[string]plantypes.Service) {
	services = make(map[string]plantypes.Service)
	unservices := make([]plantypes.Translator, 0)
	log.Infoln("Planning Translation - Base Directory")
	log.Debugf("Translators : %+v", translators)
	for _, t := range translators {
		tn := t.GetConfig().Name
		log.Infof("[%s] Planning translation", tn)
		nservices, nunservices, err := t.BaseDirectoryDetect(dir)
		if err != nil {
			log.Errorf("[%s] Failed : %s", tn, err)
		} else {
			services = plantypes.MergeServices(services, nservices)
			unservices = append(unservices, nunservices...)
			log.Infof("Identified %d namedservices and %d unnamedservices", len(nservices), len(nunservices))
			log.Infof("[%s] Done", tn)
		}
	}
	log.Infof("[Base Directory] Identified %d namedservices and %d unnamedservices", len(services), len(unservices))
	log.Infoln("Translation planning - Base Directory done")
	log.Infoln("Planning Translation - Directory Walk")
	nservices, nunservices, err := walkForServices(dir, translators, services)
	if err != nil {
		log.Errorf("Translation planning - Directory Walk failed : %s", err)
	} else {
		services = nservices
		unservices = append(unservices, nunservices...)
		log.Infoln("Translation planning - Directory Walk done")
	}
	log.Infof("[Directory Walk] Identified %d namedservices and %d unnamedservices", len(services), len(unservices))
	services = nameServices(prjName, services, unservices)
	log.Infof("[Named Services] Identified %d namedservices", len(services))
	log.Infoln("Planning Service Augmentors")
	for _, t := range translators {
		log.Debugf("[%T] Planning translation", t)
		for sn, s := range services {
			sts, err := t.ServiceAugmentDetect(sn, s)
			if err != nil {
				log.Errorf("[%T] Failed for service %s : %s", t, sn, err)
			} else {
				services[sn] = append(s, sts...)
			}
		}
		log.Debugf("[%T] Done", t)
	}
	log.Infoln("Service Augmentors planning - done")
	return
}

func GetIRTranslators(plan plantypes.Plan) (suitableTranslators []plantypes.Translator, err error) {
	log.Infoln("Planning plan translators")
	for _, t := range translators {
		log.Infof("[%T] Planning translation", t)
		ts, err := t.PlanDetect(plan)
		if err != nil {
			log.Warnf("[%T] Failed : %s", t, err)
		} else {
			suitableTranslators = append(suitableTranslators, ts...)
			log.Infof("[%T] Done", t)
		}
	}
	log.Infoln("Plan translator planning - done")
	return suitableTranslators, nil
}

func TranslateServices(plan plantypes.Plan) (ir irtypes.IR, err error) {

}
