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
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestEngine(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("1. test AddEngine", func(t *testing.T) {

		e := NewDefaultEngine()
		AddEngine(e)

		etype := reflect.TypeOf(e)
		eAfterType := reflect.TypeOf(engines[0])

		if len(engines) != 1 || etype.String() != eAfterType.String() {
			t.Fatalf("Engine was not added correctly")
		}

	})

}
