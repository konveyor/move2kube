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
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

func TestCacheEngine(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	qaTestPath := "testdata/qaenginetest.yaml"
	tmpTestPath := "/tmp/qatest.yaml"

	t.Run("Test NewInputProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewCacheEngine(qaTestPath)
		AddEngine(e)
		SetWriteCache(tmpTestPath)

		desc := "Enter the container registry username : "
		context := []string{"Enter username for container registry login"}
		def := ""

		want := "testuser"

		problem, err := qatypes.NewInputProblem(desc, context, def)
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

		if answer != want {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, want)
		}

	})

	t.Run("Test NewSelectProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewCacheEngine(qaTestPath)
		AddEngine(e)
		SetWriteCache(tmpTestPath)

		desc := "What type of container registry login do you want to use?"
		context := []string{"Docker login from config mode, will use the default config from your local machine."}
		def := "No authentication"
		opts := []string{"Use existing pull secret", "No authentication", "UserName/Password"}
		want := "UserName/Password"

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

		if answer != want {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, want)
		}

	})

	t.Run("Test NewMultilineInputProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewCacheEngine(qaTestPath)
		AddEngine(e)
		SetWriteCache(tmpTestPath)

		desc := "Multiline input problem test description : "
		context := []string{"Multiline input problem test context."}
		cachedAnswer := `line1 
line2 
line3 
`
		problem, err := qatypes.NewMultilineInputProblem(desc, context, "")
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

		if answer != cachedAnswer {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, cachedAnswer)
		}

	})

	t.Run("Test NewConfirmProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewCacheEngine(qaTestPath)
		AddEngine(e)
		SetWriteCache(tmpTestPath)

		desc := "Confirm problem test description : "
		context := []string{"Confirm input problem test context."}
		def := true
		want := true

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

		if answer != want {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %v, expected answer: %v ",
				answer, want)
		}

	})

	t.Run("Test NewMultiSelectProblem", func(t *testing.T) {

		engines = []Engine{}
		e := NewCacheEngine(qaTestPath)
		AddEngine(e)
		SetWriteCache(tmpTestPath)

		desc := "MultiSelect input problem test description : "
		context := []string{"MultiSelect input problem test context"}
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

}
