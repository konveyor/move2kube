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

package generator

import (
	"fmt"

	generatortypes "github.com/konveyor/move2kube/types/generator"
	irtypes "github.com/konveyor/move2kube/types/ir"
	plantypes "github.com/konveyor/move2kube/types/plan"
)

type GenericGenerator struct {
	name     string
	mode     string
	filepath string
	config   generatortypes.GenericConfig
}

func (g *GenericGenerator) Init(gc generatortypes.Generator) error {
	g.name = gc.Name
	g.mode = string(gc.Spec.Mode)
	g.filepath = gc.Spec.FilePath
	var ok bool
	if g.config, ok = gc.Spec.Config.(generatortypes.GenericConfig); !ok {
		return fmt.Errorf("unable to parse config for Generator %s with %T", g.filepath, g)
	}
	return nil
}

func (g *GenericGenerator) Name() string {
	return g.name
}

func (g *GenericGenerator) Mode() string {
	return g.mode
}

func (*GenericGenerator) GetGenerationOptions(plan plantypes.Plan, path string) []plantypes.GenerationOption {

}

func (*GenericGenerator) Generate(serviceName string, option plantypes.GenerationOption, ir irtypes.IR) (irtypes.IR, error) {

}
