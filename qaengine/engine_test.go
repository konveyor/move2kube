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
	"github.com/sirupsen/logrus"
)

func TestEngine(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("1. test AddEngine", func(t *testing.T) {

		//make sure the engines slice is empty
		engines = []Engine{}

		e := NewDefaultEngine()
		want := NewDefaultEngine()
		AddEngine(e)

		if len(engines) != 1 {
			t.Fatalf("Engine was not added correctly to the engines slice. Length of engines slice: %d", len(engines))
		}

		if !cmp.Equal(engines[0], want) {
			t.Fatalf("Engine was not added correctly. Difference:\n%s", cmp.Diff(want, engines[0]))
		}

	})

}
