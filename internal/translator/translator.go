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

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/translator/classes"
	"github.com/konveyor/move2kube/qaengine"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	"github.com/sirupsen/logrus"
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

	TranslateService(serviceName string, translatorPlan plantypes.Translator, artifactsToGenerate []string) map[string]translatortypes.Patch
}

func init() {
	translatorObjs := []Translator{new(classes.GoInterface)}
	for _, tt := range translatorObjs {
		t := reflect.TypeOf(tt).Elem()
		tn := t.Name()
		if ot, ok := translatorTypes[tn]; ok {
			logrus.Errorf("Two translator classes have the same name %s : %T, %T; Ignoring %T", tn, ot, t, t)
			continue
		}
		translatorTypes[tn] = t
	}
}

func Init(assetsPath string) error {
	filePaths, err := common.GetFilesByExt(assetsPath, []string{".yml", ".yaml"})
	if err != nil {
		logrus.Warnf("Unable to fetch yaml files and recognize cf manifest yamls at path %q Error: %q", assetsPath, err)
		return err
	}
	translatorConfigs := make(map[string]translatortypes.Translator)
	for _, filePath := range filePaths {
		tc, err := getTranslatorConfig(filePath)
		if err != nil {
			logrus.Debugf("Unable to load %s as Translator config", filePath, err)
			continue
		}
		if ot, ok := translatorConfigs[tc.Name]; ok {
			logrus.Errorf("Found two conflicting translator Names %s : %s, %s. Ignoring %s.", tc.Name, ot.Spec.FilePath, tc.Spec.FilePath)
			continue
		}
		if _, ok := translatorTypes[tc.Spec.Class]; ok {
			translatorConfigs[tc.Name] = tc
			continue
		}
		logrus.Errorf("Unable to find suitable translator class (%s) for translator config at %s", tc.Spec.Class, filePath)
	}
	tns := make([]string, 0)
	for tn := range translatorConfigs {
		tns = append(tns, tn)
	}
	translatorNames := qaengine.FetchMultiSelectAnswer(common.ConfigTranslatorTypesKey, "Select all translator types that you are interested in:", []string{"Services that don't support any of the translator types you are interested in will be ignored."}, tns, tns)
	for _, tn := range translatorNames {
		tc := translatorConfigs[tn]
		if c, ok := translatorTypes[tc.Spec.Class]; !ok {
			logrus.Errorf("Unable to find Translator class %s in %+v", tc.Spec.Class, translatorTypes)
		} else {
			t := reflect.New(c).Interface().(Translator)
			if err := t.Init(tc); err != nil {
				logrus.Errorf("Unable to initialize translator %s : %s", tc.Name, err)
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
	logrus.Infoln("Planning Translation - Base Directory")
	logrus.Debugf("Translators : %+v", translators)
	for _, t := range translators {
		tn := t.GetConfig().Name
		logrus.Infof("[%s] Planning translation", tn)
		nservices, nunservices, err := t.BaseDirectoryDetect(dir)
		if err != nil {
			logrus.Errorf("[%s] Failed : %s", tn, err)
		} else {
			services = plantypes.MergeServices(services, nservices)
			unservices = append(unservices, nunservices...)
			logrus.Infof("Identified %d namedservices and %d unnamedservices", len(nservices), len(nunservices))
			logrus.Infof("[%s] Done", tn)
		}
	}
	logrus.Infof("[Base Directory] Identified %d namedservices and %d unnamedservices", len(services), len(unservices))
	logrus.Infoln("Translation planning - Base Directory done")
	logrus.Infoln("Planning Translation - Directory Walk")
	nservices, nunservices, err := walkForServices(dir, translators, services)
	if err != nil {
		logrus.Errorf("Translation planning - Directory Walk failed : %s", err)
	} else {
		services = nservices
		unservices = append(unservices, nunservices...)
		logrus.Infoln("Translation planning - Directory Walk done")
	}
	logrus.Infof("[Directory Walk] Identified %d namedservices and %d unnamedservices", len(services), len(unservices))
	services = nameServices(prjName, services, unservices)
	logrus.Infof("[Named Services] Identified %d namedservices", len(services))
	logrus.Infoln("Planning Service Augmentors")
	for _, t := range translators {
		logrus.Debugf("[%T] Planning translation", t)
		for sn, s := range services {
			sts, err := t.ServiceAugmentDetect(sn, s)
			if err != nil {
				logrus.Errorf("[%T] Failed for service %s : %s", t, sn, err)
			} else {
				services[sn] = append(s, sts...)
			}
		}
		logrus.Debugf("[%T] Done", t)
	}
	logrus.Infoln("Service Augmentors planning - done")
	return
}

func GetIRTranslators(plan plantypes.Plan) (suitableTranslators []plantypes.Translator, err error) {
	logrus.Infoln("Planning plan translators")
	for _, t := range translators {
		logrus.Infof("[%T] Planning translation", t)
		ts, err := t.PlanDetect(plan)
		if err != nil {
			logrus.Warnf("[%T] Failed : %s", t, err)
		} else {
			suitableTranslators = append(suitableTranslators, ts...)
			logrus.Infof("[%T] Done", t)
		}
	}
	logrus.Infoln("Plan translator planning - done")
	return suitableTranslators, nil
}

func Translate(plan plantypes.Plan, outputPath string) (err error) {

}
