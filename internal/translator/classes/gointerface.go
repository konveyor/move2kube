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
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	gointerfacetypes "github.com/konveyor/move2kube/types/translator/classes/gointerface"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
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

	Translate(serviceName string) map[string]translatortypes.Patch
}

type GoInterface struct {
	tc     translatortypes.Translator
	config gointerfacetypes.Config
	impl   Translator
}

func init() {
	translatorObjs := []Translator{new(gointerfaces.Compose)}
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

func (t *GoInterface) Init(tc translatortypes.Translator) error {
	t.tc = tc
	config := gointerfacetypes.Config{}
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &config,
		TagName:  "yaml",
	})
	if err := decoder.Decode(tc.Spec.Config); err != nil {
		log.Errorf("unable to load config for GoInterface Translator %+v into %T : %s", tc.Spec.Config, gointerfacetypes.Config{}, err)
		return err
	} else {
		log.Debugf("GoInterface Translator config is %+v", config)
		t.config = config
	}
	log.Debugf("Looking for struct %s", t.config.Class)
	if tt, ok := translatorTypes[t.config.Class]; ok {
		t.impl = reflect.New(tt).Interface().(Translator)
		return nil
	}
	err := fmt.Errorf("unable to locate traslator struct type %s in %+v", t.config.Class, translatorTypes)
	log.Errorf("%s", err)
	return err
}

func (t *GoInterface) GetConfig() translatortypes.Translator {
	return t.tc
}

func (t *GoInterface) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return t.impl.BaseDirectoryDetect(dir)
}

func (t *GoInterface) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return t.impl.DirectoryDetect(dir)
}

func (t *GoInterface) KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	return t.impl.KnownDirectoryDetect(dir)
}

func (t *GoInterface) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	return t.impl.ServiceAugmentDetect(serviceName, service)
}

func (t *GoInterface) PlanDetect(p plantypes.Plan) ([]plantypes.Translator, error) {
	return t.impl.PlanDetect(p)
}

func (t *GoInterface) Translate(serviceName string) map[string]translatortypes.Patch {
	return t.impl.Translate(serviceName)
}