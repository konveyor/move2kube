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
	"fmt"
	"reflect"

	"github.com/konveyor/move2kube/internal/translator/gointerfaces"
	"github.com/konveyor/move2kube/internal/translator/gointerfaces/irtranslators"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	gointerfacetypes "github.com/konveyor/move2kube/types/translator/classes/gointerface"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
)

var (
	translatorTypes map[string]reflect.Type = make(map[string]reflect.Type)
)

type Translator interface {
	BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error)
	PlanDetect(plantypes.Plan) ([]plantypes.Translator, error)

	TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error)
	TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.PathMapping, error)
}

type GoInterface struct {
	tc     translatortypes.Translator
	config gointerfacetypes.Config
	impl   Translator
}

func init() {
	translatorObjs := []Translator{new(gointerfaces.Compose), new(irtranslators.Kubernetes), new(irtranslators.Knative), new(irtranslators.Tekton)}
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

func (t *GoInterface) Init(tc translatortypes.Translator) error {
	t.tc = tc
	config := gointerfacetypes.Config{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &config,
		TagName:  "yaml",
	})
	if err := decoder.Decode(tc.Spec.Config); err != nil {
		logrus.Errorf("unable to load config for GoInterface Translator %+v into %T : %s", tc.Spec.Config, gointerfacetypes.Config{}, err)
		return err
	} else {
		logrus.Debugf("GoInterface Translator config is %+v", config)
		t.config = config
	}
	logrus.Debugf("Looking for struct %s", t.config.Class)
	if tt, ok := translatorTypes[t.config.Class]; ok {
		t.impl = reflect.New(tt).Interface().(Translator)
		return nil
	}
	err := fmt.Errorf("unable to locate traslator struct type %s in %+v", t.config.Class, translatorTypes)
	logrus.Errorf("%s", err)
	return err
}

func (t *GoInterface) GetConfig() translatortypes.Translator {
	return t.tc
}

func (t *GoInterface) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	ns, uns, err := t.impl.BaseDirectoryDetect(dir)
	ns, uns = setTranslatorInfo(t.tc, ns, uns)
	return ns, uns, err
}

func (t *GoInterface) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	ns, uns, err := t.impl.DirectoryDetect(dir)
	ns, uns = setTranslatorInfo(t.tc, ns, uns)
	return ns, uns, err
}

func (t *GoInterface) KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	ns, uns, err := t.impl.KnownDirectoryDetect(dir)
	ns, uns = setTranslatorInfo(t.tc, ns, uns)
	return ns, uns, err
}

func (t *GoInterface) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	ts, err := t.impl.ServiceAugmentDetect(serviceName, service)
	_, ts = setTranslatorInfo(t.tc, nil, ts)
	return ts, err
}

func (t *GoInterface) PlanDetect(p plantypes.Plan) ([]plantypes.Translator, error) {
	ts, err := t.impl.PlanDetect(p)
	_, ts = setTranslatorInfo(t.tc, nil, ts)
	return ts, err
}

func (t *GoInterface) TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error) {
	return t.impl.TranslateService(serviceName, translatorPlan, plan, tempOutputDir)
}

func (t *GoInterface) TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.PathMapping, error) {
	return t.impl.TranslateIR(ir, plan, tempOutputDir)
}
