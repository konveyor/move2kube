/*
 *  Copyright IBM Corporation 2021, 2022
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

package common_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/common"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

const (
	emptyJson8NoBomPath string = "empty-json-data-in-utf-8-without-bom.json"
	json8NoBomPath      string = "data-in-utf-8-without-bom.json"
	json8BomPath        string = "data-in-utf-8-with-bom.json"
	json16BomPath       string = "data-in-utf-16-with-bom.json"
	json16BeBomPath     string = "data-in-utf-16-with-bom-with-be.json"
)

func setup(tempDir string) {
	const emptyJsonData = `{}`
	emptyJsonUtf8NoBomBytes := []byte(emptyJsonData)
	if err := os.WriteFile(filepath.Join(tempDir, emptyJson8NoBomPath), emptyJsonUtf8NoBomBytes, 0666); err != nil {
		panic(err)
	}
	const data = `{"foo": "bar"}`
	utf8NoBomBytes := []byte(data)
	if err := os.WriteFile(filepath.Join(tempDir, json8NoBomPath), utf8NoBomBytes, 0666); err != nil {
		panic(err)
	}
	// --------------------------
	t1 := unicode.UTF8BOM.NewEncoder()
	buf := &bytes.Buffer{}
	if _, err := transform.NewWriter(buf, t1).Write(utf8NoBomBytes); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, json8BomPath), buf.Bytes(), 0666); err != nil {
		panic(err)
	}
	// --------------------------
	t2 := unicode.UTF16(unicode.LittleEndian, unicode.ExpectBOM).NewEncoder()
	buf16 := &bytes.Buffer{}
	if _, err := transform.NewWriter(buf16, t2).Write(utf8NoBomBytes); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, json16BomPath), buf16.Bytes(), 0666); err != nil {
		panic(err)
	}
	// --------------------------
	t3 := unicode.UTF16(unicode.BigEndian, unicode.ExpectBOM).NewEncoder()
	buf16be := &bytes.Buffer{}
	if _, err := transform.NewWriter(buf16be, t3).Write(utf8NoBomBytes); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, json16BeBomPath), buf16be.Bytes(), 0666); err != nil {
		panic(err)
	}
}

func TestConvertUtf8AndUtf16ToUtf8(t *testing.T) {
	tempDir := t.TempDir()
	setup(tempDir)
	t.Run("empty json data utf-8 bytes with no BOM", func(t *testing.T) {
		var x1 interface{}
		f1 := filepath.Join(tempDir, emptyJson8NoBomPath)
		if err := common.ReadJSON(f1, &x1); err != nil {
			t.Fatalf("failed to read the json file at path '%s'. Error: %q", f1, err)
		}
		vmap, ok := x1.(map[string]interface{})
		if !ok {
			t.Fatalf("failed to type assert. Actual type is '%T' and value is %+v", x1, x1)
		}
		if len(vmap) != 0 {
			t.Fatalf("expected an empty map. Actual type is '%T' and value is %+v", x1, x1)
		}
	})
	t.Run("json utf-8 bytes with no BOM", func(t *testing.T) {
		var x1 interface{}
		f1 := filepath.Join(tempDir, json8NoBomPath)
		if err := common.ReadJSON(f1, &x1); err != nil {
			t.Fatalf("failed to read the json file at path '%s'. Error: %q", f1, err)
		}
		vmap, ok := x1.(map[string]interface{})
		if !ok {
			t.Fatalf("failed to type assert. Actual type is '%T' and value is %+v", x1, x1)
		}
		if vmap["foo"] != "bar" {
			t.Fatalf("failed to find the key 'foo' in the map. Actual type is '%T' and value is %+v", x1, x1)
		}
	})
	t.Run("json utf-8 bytes with BOM", func(t *testing.T) {
		var x1 interface{}
		f1 := filepath.Join(tempDir, json8BomPath)
		if err := common.ReadJSON(f1, &x1); err != nil {
			t.Fatalf("failed to read the json file at path '%s'. Error: %q", f1, err)
		}
		vmap, ok := x1.(map[string]interface{})
		if !ok {
			t.Fatalf("failed to type assert. Actual type is '%T' and value is %+v", x1, x1)
		}
		if vmap["foo"] != "bar" {
			t.Fatalf("failed to find the key 'foo' in the map. Actual type is '%T' and value is %+v", x1, x1)
		}
	})

	t.Run("json utf-8 bytes with BOM", func(t *testing.T) {
		var x1 interface{}
		f1 := filepath.Join(tempDir, json16BomPath)
		if err := common.ReadJSON(f1, &x1); err != nil {
			t.Fatalf("failed to read the json file at path '%s'. Error: %q", f1, err)
		}
		vmap, ok := x1.(map[string]interface{})
		if !ok {
			t.Fatalf("failed to type assert. Actual type is '%T' and value is %+v", x1, x1)
		}
		if vmap["foo"] != "bar" {
			t.Fatalf("failed to find the key 'foo' in the map. Actual type is '%T' and value is %+v", x1, x1)
		}
	})
	t.Run("json utf-8 bytes with BOM", func(t *testing.T) {
		var x1 interface{}
		f1 := filepath.Join(tempDir, json16BeBomPath)
		if err := common.ReadJSON(f1, &x1); err != nil {
			t.Fatalf("failed to read the json file at path '%s'. Error: %q", f1, err)
		}
		vmap, ok := x1.(map[string]interface{})
		if !ok {
			t.Fatalf("failed to type assert. Actual type is '%T' and value is %+v", x1, x1)
		}
		if vmap["foo"] != "bar" {
			t.Fatalf("failed to find the key 'foo' in the map. Actual type is '%T' and value is %+v", x1, x1)
		}
	})
	t.Run("launchSettings.json data utf-8 bytes with BOM", func(t *testing.T) {
		var x1 interface{}
		f1 := filepath.Join("testdata", "utf8and16bom", "launchSettings.json")
		if err := common.ReadJSON(f1, &x1); err != nil {
			t.Fatalf("failed to read the json file at path '%s'. Error: %q", f1, err)
		}
		vmap, ok := x1.(map[string]interface{})
		if !ok {
			t.Fatalf("failed to type assert. Actual type is '%T' and value is %+v", x1, x1)
		}
		//  map[iisSettings:map[anonymousAuthentication:true iisExpress:map[applicationUrl:http://localhost:53891 sslPort:44359]
		expected := map[string]interface{}{
			"iisSettings": map[string]interface{}{
				"anonymousAuthentication": true,
				"iisExpress": map[string]interface{}{
					"applicationUrl": "http://localhost:53891",
					"sslPort":        float64(44359),
				},
				"windowsAuthentication": false,
			},
			"profiles": map[string]interface{}{
				"IIS Express": map[string]interface{}{
					"commandName": "IISExpress",
					"environmentVariables": map[string]interface{}{
						"ASPNETCORE_ENVIRONMENT": "Development",
					},
					"launchBrowser": true,
				},
				"dotnet5angular": map[string]interface{}{
					"applicationUrl": "https://localhost:5001;http://localhost:5000",
					"commandName":    "Project",
					"environmentVariables": map[string]interface{}{
						"ASPNETCORE_ENVIRONMENT": "Development",
					},
					"launchBrowser": true,
				},
			},
		}
		if diff := cmp.Diff(vmap, expected); diff != "" {
			t.Fatalf("expected an empty map. Actual type is '%T' and value is %#v. Differences: %s", x1, x1, diff)
		}
	})
}
