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

package qaengine

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
)

func TestDefaultEngine(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("input type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		key := common.BaseKey + common.Delim + "input"
		answer := FetchStringAnswer(key, "Enter the name of the registry : ", []string{"Ex : " + common.DefaultRegistryURL}, common.DefaultRegistryURL)
		if answer != common.DefaultRegistryURL {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, common.DefaultRegistryURL)
		}

	})

	t.Run("select type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		key := common.BaseKey + common.Delim + "select"
		desc := "Test description"
		context := []string{"Test context"}
		def := "Option B"
		opts := []string{"Option A", "Option B", "Option C"}

		answer := FetchSelectAnswer(key, desc, context, def, opts)
		if answer != def {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, def)
		}

	})

	t.Run("multi-select type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		key := common.BaseKey + common.Delim + "multiselect"
		desc := "Test description"
		context := []string{"Test context"}
		def := []string{"Option A", "Option C"}
		opts := []string{"Option A", "Option B", "Option C", "Option D"}

		answer := FetchMultiSelectAnswer(key, desc, context, def, opts)
		if !cmp.Equal(answer, def) {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, def)
		}

	})

	t.Run("confirm type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		key := common.BaseKey + common.Delim + "confirm"
		desc := "Test description"
		context := []string{"Test context"}
		def := true

		answer := FetchBoolAnswer(key, desc, context, def)
		if answer != def {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %v, expected answer: %v ",
				answer, def)
		}

	})

	t.Run("multi-line type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		key := common.BaseKey + common.Delim + "multiline"
		desc := "Test description"
		context := []string{"Test context"}
		def := `line1
		line2
		line3`

		answer := FetchMultilineAnswer(key, desc, context, def)
		if answer != def {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, def)
		}

	})

}
