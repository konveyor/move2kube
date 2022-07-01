/*
 *  Copyright IBM Corporation 2021
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package qaengine

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

func TestCacheEngine(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	qaTestPath := "testdata/qaenginetest.yaml"
	// tmpTestPath := "/tmp/qatest.yaml"

	t.Run("input type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewStoreEngineFromCache(qaTestPath, false)
		AddEngine(e)
		// SetWriteConfig(tmpTestPath)

		key := common.JoinQASubKeys(common.BaseKey, "input")
		desc := "Enter the container registry username : "
		context := []string{"Enter username for container registry login"}
		def := ""

		want := "testuser"

		answer := FetchStringAnswer(key, desc, context, def)
		if answer != want {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, want)
		}

	})

	t.Run("select type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewStoreEngineFromCache(qaTestPath, false)
		AddEngine(e)
		// SetWriteConfig(tmpTestPath)

		key := common.JoinQASubKeys(common.BaseKey, "select")
		desc := "What type of container registry login do you want to use?"
		context := []string{"Docker login from config mode, will use the default config from your local machine."}
		def := "No authentication"
		opts := []string{"Use existing pull secret", "No authentication", "UserName/Password"}
		want := "UserName/Password"

		answer := FetchSelectAnswer(key, desc, context, def, opts)
		if answer != want {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, want)
		}

	})

	t.Run("multi-line input type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewStoreEngineFromCache(qaTestPath, false)
		AddEngine(e)
		// SetWriteConfig(tmpTestPath)

		key := common.JoinQASubKeys(common.BaseKey, "multline")
		desc := "Multiline input problem test description : "
		context := []string{"Multiline input problem test context."}
		cachedAnswer := `line1 
line2 
line3 
`

		answer := FetchMultilineInputAnswer(key, desc, context, "")
		if answer != cachedAnswer {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, cachedAnswer)
		}

	})

	t.Run("confirm type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewStoreEngineFromCache(qaTestPath, false)
		AddEngine(e)
		// SetWriteConfig(tmpTestPath)

		key := common.JoinQASubKeys(common.BaseKey, "confirm")
		desc := "Confirm problem test description : "
		context := []string{"Confirm input problem test context."}
		def := true
		want := true

		answer := FetchBoolAnswer(key, desc, context, def)
		if answer != want {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %v, expected answer: %v ",
				answer, want)
		}

	})

	t.Run("multi-select type problem", func(t *testing.T) {

		engines = []Engine{}
		e := NewStoreEngineFromCache(qaTestPath, false)
		AddEngine(e)
		// SetWriteConfig(tmpTestPath)

		key := common.JoinQASubKeys(common.BaseKey, "multiselect")
		desc := "MultiSelect input problem test description : "
		context := []string{"MultiSelect input problem test context"}
		def := []string{"Option A", "Option C"}
		opts := []string{"Option A", "Option B", "Option C", "Option D"}

		answer := FetchMultiSelectAnswer(key, desc, context, def, opts)
		if !cmp.Equal(answer, def) {
			t.Fatalf("Fetched answer was different from the default one. Fetched answer: %s, expected answer: %s ",
				answer, def)
		}

	})

}
