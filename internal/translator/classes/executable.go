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

/*
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

type Config struct {
	DetectCMD  string    `yaml:"detectCMD"`
	AnalyseCMD string    `yaml:"analyseCMD"`
	LocalEnv   string    `yaml:"localEnv"` //When this environment variable is set, the environment is setup to run the program locally
	Container  Container `yaml:"container,omitempty"`
}


	BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Translator, err error)
	ServiceAugmentDetect(serviceName string, service plantypes.Service) ([]plantypes.Translator, error)
	PlanDetect(plantypes.Plan) ([]plantypes.Translator, error)

	TranslateService(serviceName string, translatorPlan plantypes.Translator, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.Patch, error)
	TranslateIR(ir irtypes.IR, plan plantypes.Plan, tempOutputDir string) ([]translatortypes.PathMapping, error)

*/
