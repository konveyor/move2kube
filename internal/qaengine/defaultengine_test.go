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
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

func TestDefaultEngine(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("Test NewInputProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		problem, err := qatypes.NewInputProblem("Enter the name of the registry : ",
			[]string{"Ex : " + common.DefaultRegistryURL},
			common.DefaultRegistryURL)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}

		problem, err = FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}

		answer, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}

		if answer != common.DefaultRegistryURL {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, common.DefaultRegistryURL)
		}

	})

	t.Run("Test NewSelectProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		desc := "Test description"
		context := []string{"Test context"}
		def := "Option B"
		opts := []string{"Option A", "Option B", "Option C"}

		problem, err := qatypes.NewSelectProblem(desc, context, def, opts)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}

		problem, err = FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}

		answer, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}

		if answer != def {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, def)
		}

	})

	t.Run("Test NewMultiSelectProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		desc := "Test description"
		context := []string{"Test context"}
		def := []string{"Option A", "Option C"}
		opts := []string{"Option A", "Option B", "Option C", "Option D"}

		problem, err := qatypes.NewMultiSelectProblem(desc, context, def, opts)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}

		problem, err = FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}

		answer, err := problem.GetSliceAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}

		if !cmp.Equal(answer, def) {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, def)
		}

	})

	t.Run("Test NewConfirmProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		desc := "Test description"
		context := []string{"Test context"}
		def := true

		problem, err := qatypes.NewConfirmProblem(desc, context, def)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}

		problem, err = FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}

		answer, err := problem.GetBoolAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}

		if answer != def {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %v, expected answer: %v ",
				answer, def)
		}

	})

	t.Run("Test NewMultilineInputProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		desc := "Test description"
		context := []string{"Test context"}
		def := `line1
		line2
		line3`

		problem, err := qatypes.NewMultilineInputProblem(desc, context, def)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}

		problem, err = FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}

		answer, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}

		if answer != def {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, def)
		}

	})

	t.Run("Test NewPasswordProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewDefaultEngine()
		AddEngine(e)

		desc := "Test description"
		context := []string{"Password:"}

		problem, err := qatypes.NewPasswordProblem(desc, context)
		if err != nil {
			log.Fatalf("Unable to create problem : %s", err)
		}

		problem, err = FetchAnswer(problem)
		if err != nil {
			log.Fatalf("Unable to fetch answer : %s", err)
		}

		answer, err := problem.GetStringAnswer()
		if err != nil {
			log.Fatalf("Unable to get answer : %s", err)
		}

		if answer != "" {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, "")
		}

	})

}
