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

	"github.com/konveyor/move2kube/internal/translator/gointerface"
	plantypes "github.com/konveyor/move2kube/types/plan"
	translatortypes "github.com/konveyor/move2kube/types/translator"
	gointerfacetypes "github.com/konveyor/move2kube/types/translator/classes/gointerface"
	log "github.com/sirupsen/logrus"
)

var (
	translatorTypes map[string]reflect.Type = make(map[string]reflect.Type)
	translators     map[string]Translator   = make(map[string]Translator)
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
	translatorObjs := []Translator{gointerface.Compose{}}
	for _, tt := range translatorObjs {
		t := reflect.TypeOf(tt)
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
	var ok bool
	if t.config, ok = tc.Spec.Config.(gointerfacetypes.Config); !ok {
		err := fmt.Errorf("unable to load config %+v into %T", tc.Spec.Config, gointerfacetypes.Config{})
		log.Errorf("%s", err)
		return err
	}
	t.impl = reflect.New(translatorTypes[tc.Spec.Class]).Interface().(Translator)
	return nil
}

func (t *GoInterface) GetConfig() translatortypes.Translator {
	return t.tc
}

func (t *GoInterface) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	panic("not implemented") // TODO: Implement
}

func (t *GoInterface) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	panic("not implemented") // TODO: Implement
}

func (t *GoInterface) KnownDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error) {
	panic("not implemented") // TODO: Implement
}

func (t *GoInterface) ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error) {
	panic("not implemented") // TODO: Implement
}

func (t *GoInterface) PlanDetect(_ plantypes.Plan) ([]plantypes.Translator, error) {
	panic("not implemented") // TODO: Implement
}

func (t *GoInterface) Translate(serviceName string) map[string]translatortypes.Patch {
	panic("not implemented") // TODO: Implement
}
