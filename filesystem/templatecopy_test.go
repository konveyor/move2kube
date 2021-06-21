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

package filesystem

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestWriteTemplateToFile(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	tcs := []struct {
		name   string
		inTmpl string
		inData interface{}
		outStr string
		outErr bool
	}{
		{"fill an empty template with an empty string", "", "", "", false},
		{"fill an empty template with an empty struct with no fields", "", struct{}{}, "", false},
		{"fill an empty template with an empty struct", "", struct {
			Name string
			ID   int
		}{}, "", false},
		{"fill an empty template with a filled struct", "", struct {
			Name string
			ID   int
		}{"foobar", 42}, "", false},
		{"fill a template with an empty struct with no fields", "Hello! My name is {{.Name}} and my ID is {{.ID}}", struct{}{}, "", true},
		{"fill a template with an empty struct", "Hello! My name is {{.Name}} and my ID is {{.ID}}", struct {
			Name string
			ID   int
		}{}, "Hello! My name is  and my ID is 0", false},
		{"fill a template with a filled struct", "Hello! My name is {{.Name}} and my ID is {{.ID}}", struct {
			Name string
			ID   int
		}{"foobar", 42}, "Hello! My name is foobar and my ID is 42", false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			path := t.TempDir() + "test1.go"
			err := writeTemplateToFile(tc.inTmpl, tc.inData, path, os.ModePerm)
			if !tc.outErr {
				if err != nil {
					t.Fatalf("Got an error while trying to fill the tempate %q with the data %v and write it to the path %q. Expected: %q Actual: Error: %v", tc.inTmpl, tc.inData, path, tc.outStr, err)
				}
				filledBytes, err := ioutil.ReadFile(path)
				if err != nil {
					t.Fatalf("Failed to read the file %q that we just wrote.", path)
				}
				if filled := string(filledBytes); filled != tc.outStr {
					t.Fatalf("Failed to fill the tempate %q with the data %v properly. Expected: %q Actual: %v", tc.inTmpl, tc.inData, tc.outStr, filled)
				}
			} else {
				if err == nil {
					t.Fatalf("Should not have succeeded in filling the tempate %q with the data %v. Expected an error.", tc.inTmpl, tc.inData)
				}
			}
		})
	}
	t.Run("try to write filled template when the path doesn't exist", func(t *testing.T) {
		inTmpl := "Hello! My name is {{.Name}} and my ID is {{.ID}}"
		inData := struct {
			Name string
			ID   int
		}{"foobar", 42}
		path := "/this/path/does/not/exist/foobar"
		err := writeTemplateToFile(inTmpl, inData, path, os.ModePerm)
		if err == nil {
			t.Fatalf("Should not have succeeded in writing to the path %q since it doesn't exit. Expected an error.", path)
		}
	})
}
