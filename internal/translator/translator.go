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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"

	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/translator/classes"
	"github.com/konveyor/move2kube/qaengine"
	irtypes "github.com/konveyor/move2kube/types/ir"
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

	TranslateService(serviceName string, translatorPlan plantypes.Translator, tempOutputDir string) ([]translatortypes.Patch, error)
	PathForIR(patch translatortypes.Patch, planTranslator plantypes.Translator) string
	TranslateIR(ir irtypes.IR, destDir string, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.PathMapping, error)
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

func InitTranslators(translatorToInit map[string]string) error {
	for tn, tfilepath := range translatorToInit {
		tc, err := getTranslatorConfig(tfilepath)
		if err != nil {
			logrus.Errorf("Unable to load %s as Translator config", tfilepath, err)
			continue
		}
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

func GetPlanTranslators(plan plantypes.Plan) (suitableTranslators []plantypes.Translator, err error) {
	logrus.Infoln("Planning plan translators")
	for _, t := range translators {
		logrus.Infof("[%s] Planning translation", t.GetConfig().Name)
		ts, err := t.PlanDetect(plan)
		if err != nil {
			logrus.Warnf("[%s] Failed : %s", t.GetConfig().Name, err)
		} else {
			suitableTranslators = append(suitableTranslators, ts...)
			logrus.Infof("[%s] Done", t.GetConfig().Name)
		}
	}
	logrus.Infoln("Plan translator planning - done")
	return suitableTranslators, nil
}

func Translate(plan plantypes.Plan, outputPath string) (err error) {
	tempOutputPath := filepath.Join(outputPath, common.TempOutputRelPath)
	if err := os.MkdirAll(tempOutputPath, common.DefaultDirectoryPermission); err != nil {
		logrus.Fatalf("Failed to create the output directory at path %s Error: %q", tempOutputPath, err)
	}
	tempInputPath, err := ioutil.TempDir(common.TempPath, "translate")
	if err != nil {
		logrus.Fatalf("Failed to create temporary input directory at path %s Error: %q", tempInputPath, err)
	}
	logrus.Debugf("Temp translate Dir : %s", tempInputPath)
	//defer os.RemoveAll(tempInputPath) // TOFIX: Uncomment
	patches := []translatortypes.Patch{}
	pathMappings := []translatortypes.PathMapping{}
	for serviceName, service := range plan.Spec.Services {
		tempServiceOutputPath := filepath.Join(tempOutputPath, serviceName)
		if err := os.MkdirAll(tempServiceOutputPath, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to create the output directory at path %s Error: %q", tempServiceOutputPath, err)
		}
		for _, translator := range service {
			var ttempdirName string
			if ttempdirName, err = ioutil.TempDir(tempServiceOutputPath, translator.Name+".*"); err != nil {
				logrus.Fatalf("Failed to create the temp translator output directory at path %s Error: %q", tempServiceOutputPath, err)
			}
			tempServiceTranslatorOutputPath := filepath.Join(tempServiceOutputPath, ttempdirName)
			tempTranslator := plantypes.Translator{}
			err := plantypes.CopyObj(translator, &tempTranslator)
			if err != nil {
				logrus.Errorf("Unable to make copy of translator obj %+v. Ignoring : %s", translator, err)
				continue
			}
			_, err = plantypes.ConvertPathsEncode(&tempTranslator, plan.Spec.RootDir)
			if err != nil {
				logrus.Errorf("Unable to encode of translator obj %+v. Ignoring : %s", translator, err)
				continue
			}
			_, err = plantypes.ConvertPathsDecode(&tempTranslator, tempInputPath)
			if err != nil {
				logrus.Errorf("Unable to decode of translator obj %+v. Ignoring : %s", translator, err)
				continue
			}
			if err := filesystem.Replicate(plan.Spec.RootDir, tempInputPath); err != nil {
				logrus.Errorf("Unable to copy contents to temp directory %s: %s", tempInputPath, err)
			}
			ps, err := translators[translator.Name].TranslateService(serviceName, tempTranslator, tempServiceTranslatorOutputPath)
			if err != nil {
				logrus.Errorf("Unable to translate service %s using %s : %s", serviceName, translator.Name, err)
				continue
			}
			for _, p := range ps {
				p.ServiceName = serviceName
				p.Translator = translator
				pathMappings = append(pathMappings, convertPatchToPathMappings(p, plan.Spec.RootDir, tempInputPath, tempOutputPath, filepath.Dir(translators[translator.Name].GetConfig().Spec.FilePath))...)
				patches = append(patches, p)
			}
		}
	}
	logrus.Infof("Got %d patches from services", len(patches))
	irs := map[string]map[string]irtypes.IR{} // [translatorName][path]
	logrus.Infof("Processing patches for IR")
	for _, pt := range plan.Spec.PlanTranslators {
		for _, p := range patches {
			path := translators[pt.Name].PathForIR(p, pt)
			if irs[pt.Name] == nil {
				irs[pt.Name] = map[string]irtypes.IR{}
			}
			if ir, ok := irs[pt.Name][path]; !ok {
				irs[pt.Name][path] = p.IR
			} else {
				ir.Merge(p.IR)
				irs[pt.Name][path] = ir
			}
		}
	}
	logrus.Infof("Done Processing patches for IR")
	logrus.Debugf("IRs : %+v", irs)

	logrus.Infof("Starting IR Translations")
	for ptName, tIRs := range irs {
		logrus.Infof("Starting %s IR Translations with %d IRs", ptName, len(tIRs))
		for path, ir := range tIRs {
			irTempdir, err := ioutil.TempDir(tempOutputPath, "ir.*")
			if err != nil {
				logrus.Fatalf("Failed to create the temp translator output directory at path %s Error: %q", tempOutputPath, err)
			}
			pm, err := translators[ptName].TranslateIR(ir, path, plan, irTempdir)
			if err != nil {
				logrus.Errorf("Unable to translate IR using %s", ptName)
				continue
			}
			pathMappings = append(pathMappings, pm...)
		}
		logrus.Infof("%s IR Translation done", ptName)
	}
	logrus.Infof("IR Translations Done")
	if err := createOutput(pathMappings, plan.Spec.RootDir, outputPath); err != nil {
		logrus.Errorf("Unable to create output from pathmappings : %s", err)
		return err
	}
	if err := os.RemoveAll(tempOutputPath); err != nil {
		logrus.Errorf("Unable to delete temp directory %s : %s", tempOutputPath, err)
	}
	return nil
}
