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

	qatypes "github.com/konveyor/move2kube/types/qaengine"

	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
)

func TestDefaultEngine(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("Fetch the default answer", func(t *testing.T) {

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
}
