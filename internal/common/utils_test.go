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

package common_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/info"
	log "github.com/sirupsen/logrus"
)

func TestGetFilesByExt(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get files by extension when the path doesn't exist", func(t *testing.T) {
		path1 := "foobar"
		if _, err := common.GetFilesByExt(path1, []string{".yaml", ".yml"}); err == nil {
			t.Fatal("Should have given an error since the path is non existent.")
		}
	})

	t.Run("get files by extension when the path is a file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.yaml"
		filepaths, err := common.GetFilesByExt(path1, []string{".yaml", ".yml"})
		if err != nil {
			t.Fatal("Should not have given any error. Error:", err)
		}
		if len(filepaths) != 1 || filepaths[0] != path1 {
			t.Fatal("Failed to get the correct paths. Expected:", path1, "Actual:", filepaths)
		}
	})

	t.Run("get files by extension in the normal use case", func(t *testing.T) {
		path1 := "testdata/validfiles"
		filepaths, err := common.GetFilesByExt(path1, []string{".yaml", ".yml"})
		if err != nil {
			t.Fatal("Should not have given any error. Error:", err)
		}
		want := []string{"testdata/validfiles/test1.yaml", "testdata/validfiles/test2.yml", "testdata/validfiles/versioninfo.yaml"}
		if !reflect.DeepEqual(filepaths, want) {
			t.Fatal("Failed to get the correct paths. Expected:", want, "Actual:", filepaths)
		}
	})

	t.Run("get files by extension when the directory is empty", func(t *testing.T) {
		path1 := t.TempDir()
		filepaths, err := common.GetFilesByExt(path1, []string{".yaml", ".yml"})
		if err != nil {
			t.Fatal("Should not have given any error. Error:", err)
		}
		if len(filepaths) != 0 {
			t.Fatal("Should not have returned any paths. Paths returned:", filepaths)
		}
	})

	t.Run("get files by extension when you don't have permissions for the directory", func(t *testing.T) {
		parentDir := t.TempDir()
		path1 := filepath.Join(parentDir, "app1")
		if err := os.Mkdir(path1, 0); err != nil {
			t.Fatal("Failed to create the temporary directory", path1, "for testing. Error:", err)
		}
		if _, err := common.GetFilesByExt(path1, []string{".yaml", ".yml"}); err == nil {
			t.Fatal("Should have given an error since we don't have permissions to read the directory.")
		}
	})
}

func TestGetFilesByName(t *testing.T) {
	t.Run("get files by name when the path doesn't exist", func(t *testing.T) {
		path1 := "foobar"
		if _, err := common.GetFilesByName(path1, []string{"test1.yaml", "test2.yml"}); err == nil {
			t.Fatal("Should have given an error since the path is non existent. Error:", err)
		}
	})

	t.Run("get files by name when the path is a file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.yaml"
		filepaths, err := common.GetFilesByName(path1, []string{"test1.yaml", "test2.yml"})
		if err != nil {
			t.Fatal("Should not have given any error. Error:", err)
		}
		if len(filepaths) != 1 || filepaths[0] != path1 {
			t.Fatal("Failed to get the correct paths. Expected:", path1, "Actual:", filepaths)
		}
	})

	t.Run("get files by name in the normal use case", func(t *testing.T) {
		path1 := "testdata/validfiles"
		filepaths, err := common.GetFilesByName(path1, []string{"test1.yaml", "test2.yml"})
		if err != nil {
			t.Fatal("Should not have given any error. Error:", err)
		}
		want := []string{"testdata/validfiles/test1.yaml", "testdata/validfiles/test2.yml"}
		if !reflect.DeepEqual(filepaths, want) {
			t.Fatal("Failed to get the correct paths. Expected:", want, "Actual:", filepaths)
		}
	})

	t.Run("get files by name when the directory is empty", func(t *testing.T) {
		path1 := t.TempDir()
		filepaths, err := common.GetFilesByName(path1, []string{"test1.yaml", "test2.yml"})
		if err != nil {
			t.Fatal("Should not have given any error. Error:", err)
		}
		if len(filepaths) != 0 {
			t.Fatal("Should not have returned any paths. Paths returned:", filepaths)
		}
	})

	t.Run("get files by name when you don't have permissions for the directory", func(t *testing.T) {
		parentDir := t.TempDir()
		path1 := filepath.Join(parentDir, "app1")
		if err := os.Mkdir(path1, 0); err != nil {
			t.Fatal("Failed to create the temporary directory", path1, "for testing. Error:", err)
		}
		if _, err := common.GetFilesByName(path1, []string{"test1.yaml", "test2.yml"}); err == nil {
			t.Fatal("Should have given an error since we don't have permissions to read the directory.")
		}
	})
}

func TestYamlAttrPresent(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get attribute from non existent path", func(t *testing.T) {
		path1 := "foobar"
		attr1 := "attr1"
		if ok, _ := common.YamlAttrPresent(path1, attr1); ok {
			t.Fatal("Should not have succeeded. The file", path1, "does not exist.")
		}
	})

	t.Run("get attribute from invalid yaml file", func(t *testing.T) {
		path1 := "testdata/invalidfiles/test1.yaml"
		attr1 := "attr1"
		if ok, _ := common.YamlAttrPresent(path1, attr1); ok {
			t.Fatal("Should not have succeeded. The file", path1, "is not a valid yaml file.")
		}
	})

	t.Run("get non existent attribute from yaml file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.yaml"
		attr1 := "attr1"
		if ok, _ := common.YamlAttrPresent(path1, attr1); ok {
			t.Fatal("Should not have succeeded. The file", path1, "does not contain the attribute", attr1)
		}
	})

	t.Run("get attribute from yaml file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.yaml"
		attr1 := "kind"
		want := "ClusterMetadata"
		if ok, val := common.YamlAttrPresent(path1, attr1); !ok || val != want {
			t.Fatal("Failed to get the attribute", attr1, "from the file", path1, "properly. Expected:", want, "Actual:", val)
		}
	})
}

func TestGetImageNameAndTag(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get imagename and tag", func(t *testing.T) {
		wantImageName := "getting-started"
		wantTag := "1.2.3-alpha.beta.gamma+hello.123.world"
		imageNameAndTag := "konveyor/" + wantImageName + ":" + wantTag
		imageName, tag := common.GetImageNameAndTag(imageNameAndTag)
		if imageName != wantImageName {
			t.Fatal("Tag is incorrect. Expected:", wantImageName, "Actual:", imageName)
		} else if tag != wantTag {
			t.Fatal("Tag is incorrect. Expected:", wantTag, "Actual:", tag)
		}
	})

	t.Run("get imagename and tag when there is no tag", func(t *testing.T) {
		wantImageName := "getting-started"
		wantTag := "latest"
		imageNameAndTag := "konveyor/" + wantImageName
		imageName, tag := common.GetImageNameAndTag(imageNameAndTag)
		if imageName != wantImageName {
			t.Fatal("Tag is incorrect. Expected:", wantImageName, "Actual:", imageName)
		} else if tag != wantTag {
			t.Fatal("Tag is incorrect. Expected:", wantTag, "Actual:", tag)
		}
	})

}

type givesYamlError struct{}

func (*givesYamlError) MarshalYAML() (interface{}, error) {
	return nil, fmt.Errorf("Can't marshal this type to yaml.")
}

func TestWriteYaml(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("write some data to an invalid path", func(t *testing.T) {
		path1 := "/this/does/not/exist/foobar.yaml"
		data1 := "contents1"
		if err := common.WriteYaml(path1, data1); err == nil {
			t.Fatal("Should not have succeeded since the path", path1, "is invalid.")
		}
	})

	t.Run("write some data to a yaml file", func(t *testing.T) {
		path1 := t.TempDir() + "foobar.yaml"
		data1 := struct {
			Foo string
			Bar int
		}{"contents1", 42}
		if err := common.WriteYaml(path1, data1); err != nil {
			t.Fatal("Failed to write the data", data1, "to the file path", path1, ". Error:", err)
		}
		want := "foo: contents1\nbar: 42\n"
		if yamldata, err := ioutil.ReadFile(path1); err != nil {
			t.Fatal("Failed to read the file we just wrote. Error:", err)
		} else if yamldatastr := string(yamldata); yamldatastr != want {
			t.Fatal("Failed to encode the data to yaml properly. Expected:", want, "Actual:", yamldatastr)
		}
	})

	t.Run("write some data that cannot be encoded to yaml to file", func(t *testing.T) {
		path1 := t.TempDir() + "foobar.yaml"
		data1 := &givesYamlError{}
		if err := common.WriteYaml(path1, data1); err == nil {
			t.Fatal("Should not have succeeded since the data", data1, "cannot be marshalled to yaml.")
		}
	})
}

func TestReadYaml(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("read some data from non existent path", func(t *testing.T) {
		path1 := "foobar"
		data1 := struct {
			Name string
			Tag  string
		}{"foo", "bar"}
		if err := common.ReadYaml(path1, &data1); err == nil {
			t.Fatal("Should not have succeeded. The file", path1, "does not exist.")
		}
	})

	t.Run("read some data from invalid yaml file", func(t *testing.T) {
		path1 := "testdata/invalidfiles/test1.yaml"
		data1 := struct {
			Name string
			Tag  string
		}{"foo", "bar"}
		if err := common.ReadYaml(path1, &data1); err == nil {
			t.Fatal("Should not have succeeded. The file", path1, "is not a valid yaml file.")
		}
	})

	t.Run("read some non existent keys from yaml file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.yaml"
		data1 := struct {
			Name string
			Tag  string
		}{"foo", "bar"}
		want := struct {
			Name string
			Tag  string
		}{"foo", "bar"}
		if err := common.ReadYaml(path1, &data1); err != nil {
			t.Fatal("Failed to read yaml from the file", path1, "Error:", err)
		}
		if data1 != want {
			t.Fatal("Failed to read the data properly. Expected:", want, "Actual:", data1)
		}
	})

	t.Run("read some data from yaml file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.yaml"
		data1 := struct {
			Kind        string `yaml:"kind"`
			ContextName string `yaml:"contextName"`
		}{"foo", "bar"}
		want := struct {
			Kind        string `yaml:"kind"`
			ContextName string `yaml:"contextName"`
		}{"ClusterMetadata", "name1"}
		if err := common.ReadYaml(path1, &data1); err != nil {
			t.Fatal("Failed to read yaml from the file", path1, "Error:", err)
		}
		if data1 != want {
			t.Fatal("Failed to read the data properly. Expected:", want, "Actual:", data1)
		}
	})

	t.Run("read version info from yaml file", func(t *testing.T) {
		path1 := "testdata/validfiles/versioninfo.yaml"
		data1 := info.VersionInfo{Version: "0.0.0", GitCommit: "0.0.0", GitTreeState: "0.0.0", GoVersion: "0.0.0"}
		want := info.VersionInfo{Version: "0.0.0", GitCommit: "1.0.0", GitTreeState: "1.1.0", GoVersion: "1.1.1"}
		if err := common.ReadYaml(path1, &data1); err != nil {
			t.Fatal("Failed to read yaml from the file", path1, "Error:", err)
		}
		if data1 != want {
			t.Fatal("Failed to read the data properly. Expected:", want, "Actual:", data1)
		}
	})
}

type givesJSONError struct{}

func (*givesJSONError) MarshalJSON() ([]byte, error) {
	return nil, fmt.Errorf("Can't marshal this type to json.")
}

func TestWriteJSON(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("write some data to an invalid path", func(t *testing.T) {
		path1 := "/this/does/not/exist/foobar.json"
		data1 := "contents1"
		if err := common.WriteJSON(path1, data1); err == nil {
			t.Fatal("Should not have succeeded since the path", path1, "is invalid.")
		}
	})

	t.Run("write some data to a json file", func(t *testing.T) {
		path1 := t.TempDir() + "foobar.json"
		data1 := struct {
			Foo string
			Bar int
		}{"contents1", 42}
		if err := common.WriteJSON(path1, data1); err != nil {
			t.Fatal("Failed to write the data", data1, "to the file path", path1, ". Error:", err)
		}
		want := "{\"Foo\":\"contents1\",\"Bar\":42}\n"
		if yamldata, err := ioutil.ReadFile(path1); err != nil {
			t.Fatal("Failed to read the file we just wrote. Error:", err)
		} else if yamldatastr := string(yamldata); yamldatastr != want {
			//log.Errorf("%q", yamldatastr)
			t.Fatal("Failed to encode the data to json properly. Expected:", want, "Actual:", yamldatastr)
		}
	})

	t.Run("write some data that cannot be encoded to json to file", func(t *testing.T) {
		path1 := t.TempDir() + "foobar.json"
		data1 := &givesJSONError{}
		if err := common.WriteJSON(path1, data1); err == nil {
			t.Fatal("Should not have succeeded since the data", data1, "cannot be marshalled to json.")
		}
	})
}

func TestReadJSON(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("read some data from non existent path", func(t *testing.T) {
		path1 := "foobar"
		data1 := struct {
			Name string
			Foo  int
			Bar  []string
		}{}
		if err := common.ReadJSON(path1, &data1); err == nil {
			t.Fatal("Should not have succeeded. The file", path1, "does not exist.")
		}
	})

	t.Run("read some data from invalid json file", func(t *testing.T) {
		path1 := "testdata/invalidfiles/test1.json"
		data1 := struct {
			Name string
			Foo  int
			Bar  []string
		}{}
		if err := common.ReadJSON(path1, &data1); err == nil {
			t.Fatal("Should not have succeeded. The file", path1, "is not a valid json file.")
		}
	})

	t.Run("read some non existent keys from json file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.json"
		data1 := struct {
			Key1 string
			Key2 string
		}{"foo", "bar"}
		want := struct {
			Key1 string
			Key2 string
		}{"foo", "bar"}
		if err := common.ReadJSON(path1, &data1); err != nil {
			t.Fatal("Failed to read json from the file", path1, "Error:", err)
		}
		if data1 != want {
			t.Fatal("Failed to read the data properly. Expected:", want, "Actual:", data1)
		}
	})

	t.Run("read some data from json file", func(t *testing.T) {
		path1 := "testdata/validfiles/test1.json"
		data1 := struct {
			Name string
			Foo  int
			Bar  []string
		}{}
		want := struct {
			Name string
			Foo  int
			Bar  []string
		}{"name1", 42, []string{"bar"}}
		if err := common.ReadJSON(path1, &data1); err != nil {
			t.Fatal("Failed to read json from the file", path1, "Error:", err)
		}
		if !reflect.DeepEqual(data1, want) {
			t.Fatal("Failed to read the data properly. Expected:", want, "Actual:", data1)
		}
	})

	t.Run("read version info from json file", func(t *testing.T) {
		path1 := "testdata/validfiles/versioninfo.json"
		data1 := info.VersionInfo{Version: "0.0.0", GitCommit: "0.0.0", GitTreeState: "0.0.0", GoVersion: "0.0.0"}
		want := info.VersionInfo{Version: "0.0.0", GitCommit: "1.0.0", GitTreeState: "1.1.0", GoVersion: "1.1.1"}
		if err := common.ReadJSON(path1, &data1); err != nil {
			t.Fatal("Failed to read json from the file", path1, "Error:", err)
		}
		if data1 != want {
			t.Fatal("Failed to read the data properly. Expected:", want, "Actual:", data1)
		}
	})
}

func TestNormalizeForFilename(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct{ name, in, out string }{
		{"normalize an invalid filename", "foobar%${/2\n\tinv.json.yaml.", "2-inv.json.yaml_d65d80a1c389718f"},
		{"normalize a valid filename", "foobar", "foobar_534a426c0464b01e"},
		{"normalize a long valid filename", "thisisalongfilenamefoobar", "thisisalongfile_730bb88a395ce114"},
		{"normalize a valid filename with an extension", "foobar.json", "foobar.json_f161da8efa921f1f"},
		{"normalize a valid filepath", "path/to/a/file/foobar.json", "foobar.json_b1760918996ebb3"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := common.NormalizeForFilename(tc.in)
			if output != tc.out {
				t.Fatalf("Failed to normalize the string %q properly. Expected: %q Actual: %q", tc.in, tc.out, output)
			}
		})
	}
}

func TestNormalizeForServiceName(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct{ name, in, out string }{
		// {"normalize an invalid service name", "foobar%${/2\n\tinv.website.registration.", "foobar%${/2\n\tinv-website-registration-"}, // TODO: does this edge case have to be handled?
		{"normalize an invalid service name", "foobar.website.registration.", "foobar-website-registration-"},
		{"normalize a valid service name", "foobar", "foobar"},
		{"normalize a long valid service name", "thisisalongservicenamefoobar", "thisisalongservicenamefoobar"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := common.NormalizeForServiceName(tc.in)
			if output != tc.out {
				t.Fatalf("Failed to normalize the string %q properly. Expected: %q Actual: %q", tc.in, tc.out, output)
			}
		})
	}
}

func TestIsStringPresent(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name    string
		inArr   []string
		inQuery string
		out     bool
	}{
		{"find a string in the array", []string{"foo", "bar"}, "foo", true},
		{"find a non existent string in the array", []string{"foo", "bar"}, "str1", false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			found := common.IsStringPresent(tc.inArr, tc.inQuery)
			if found != tc.out {
				if tc.out {
					t.Fatalf("Failed to find the string %q in the array %v", tc.inQuery, tc.inArr)
				} else {
					t.Fatalf("Should not have found the string %q in the array %v", tc.inQuery, tc.inArr)
				}
			}
		})
	}
}

func TestIsIntPresent(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name    string
		inArr   []int
		inQuery int
		out     bool
	}{
		{"find a int in an empty array", []int{}, 0, false},
		{"find a int in an array", []int{100, 0, 1, -1, -42}, 0, true},
		{"find a non existent int in an array", []int{100, 0, 1, -1, -42}, 200, false},
		{"find a int in an array when there are duplicates", []int{100, 0, -42, -42, 1}, -42, true},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			found := common.IsIntPresent(tc.inArr, tc.inQuery)
			if found != tc.out {
				if tc.out {
					t.Fatalf("Failed to find the int %d in the array %v", tc.inQuery, tc.inArr)
				} else {
					t.Fatalf("Should not have found the int %d in the array %v", tc.inQuery, tc.inArr)
				}
			}
		})
	}
}

func TestMergeStringSlices(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name   string
		inArr1 []string
		inArr2 []string
		out    []string
	}{
		{"merge 2 empty arrays", []string{}, []string{}, []string{}},
		{"merge a filled array into an empty array", []string{}, []string{"foo", "bar"}, []string{"foo", "bar"}},
		{"merge an empty array into a filled array", []string{"foo", "bar"}, []string{}, []string{"foo", "bar"}},
		{"merge 2 filled arrays", []string{"foo", "bar"}, []string{"foo", "bar", "item1", "item2"}, []string{"foo", "bar", "item1", "item2"}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			merged := common.MergeStringSlices(tc.inArr1, tc.inArr2)
			if !reflect.DeepEqual(merged, tc.out) {
				t.Fatalf("Failed to merge the arrays properly. Array1: %v Array2: %v Expected: %v Actual: %v", tc.inArr1, tc.inArr2, tc.out, merged)
			}
		})
	}
}

func TestMergeIntSlices(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name   string
		inArr1 []int
		inArr2 []int
		out    []int
	}{
		{"merge 2 empty arrays", []int{}, []int{}, []int{}},
		{"merge a filled array into an empty array", []int{}, []int{100, 0}, []int{100, 0}},
		{"merge an empty array into a filled array", []int{100, -42, -1, 0}, []int{}, []int{100, -42, -1, 0}},
		{"merge 2 filled arrays", []int{100, -42, -1, 0}, []int{10, -1, -1, 0, 2}, []int{100, -42, -1, 0, 10, 2}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			merged := common.MergeIntSlices(tc.inArr1, tc.inArr2)
			if !reflect.DeepEqual(merged, tc.out) {
				t.Fatalf("Failed to merge the arrays properly. Array1: %v Array2: %v Expected: %v Actual: %v", tc.inArr1, tc.inArr2, tc.out, merged)
			}
		})
	}
}

func TestGetStringFromTemplate(t *testing.T) {
	log.SetLevel(log.DebugLevel)

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
			filled, err := common.GetStringFromTemplate(tc.inTmpl, tc.inData)
			if !tc.outErr {
				if err != nil {
					t.Fatalf("Got an error while trying to fill the tempate %q with the data %v properly. Expected: %q Actual: Error: %v", tc.inTmpl, tc.inData, tc.outStr, err)
				}
				if filled != tc.outStr {
					t.Fatalf("Failed to fill the tempate %q with the data %v properly. Expected: %q Actual: %v", tc.inTmpl, tc.inData, tc.outStr, filled)
				}
			} else {
				if err == nil {
					t.Fatalf("Should not have succeeded in filling the tempate %q with the data %v. Expected an error. Actual: %v", tc.inTmpl, tc.inData, filled)
				}
			}
		})
	}
}

func TestWriteTemplateToFile(t *testing.T) {
	log.SetLevel(log.DebugLevel)

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
			err := common.WriteTemplateToFile(tc.inTmpl, tc.inData, path, os.ModePerm)
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
		err := common.WriteTemplateToFile(inTmpl, inData, path, os.ModePerm)
		if err == nil {
			t.Fatalf("Should not have succeeded in writing to the path %q since it doesn't exit. Expected an error.", path)
		}
	})
}

func TestGetClosestMatchingString(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name    string
		inArr   []string
		inQuery string
		out     string
	}{
		{"find the closest in an empty array", []string{}, "foo", ""},
		{"find the closest in the array when the string exists", []string{"foo", "bar"}, "foo", "foo"},
		{"find the closest in the array when the string doesn't exist", []string{"foo", "bar"}, "bar2", "bar"},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			foundStr := common.GetClosestMatchingString(tc.inArr, tc.inQuery)
			if foundStr != tc.out {
				t.Fatalf("Failed to find the closest string to %q in the array %v. Expected: %q Actual: %q", tc.inQuery, tc.inArr, tc.out, foundStr)
			}
		})
	}
}

func TestMergeStringMaps(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	type mss = map[string]string
	tcs := []struct {
		name   string
		inMap1 mss
		inMap2 mss
		out    mss
	}{
		{"merge 2 empty maps", mss{}, mss{}, mss{}},
		{"merge a filled map into an empty map", mss{}, mss{"key1": "val1"}, mss{"key1": "val1"}},
		{"merge an empty map into a filled map", mss{"key1": "val1"}, mss{}, mss{"key1": "val1"}},
		{"merge 2 filled maps", mss{"key1": "val1", "key2": "val2"}, mss{"key2": "newval2", "key3": "val3"}, mss{"key1": "val1", "key2": "newval2", "key3": "val3"}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			merged := common.MergeStringMaps(tc.inMap1, tc.inMap2)
			if !reflect.DeepEqual(merged, tc.out) {
				t.Fatalf("Failed to merge the maps properly. Map1: %v Map2: %v Expected: %v Actual: %v", tc.inMap1, tc.inMap2, tc.out, merged)
			}
		})
	}
}

func TestMakeFileNameCompliant(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name string
		in   string
		out  string
	}{
		{"normalize an empty name", "", ""},
		{"normalize an invalid name", "foo\n123.bar%4.inv#22.-", "foo-123.bar-4.inv-22.-"},
		{"normalize an invalid name", "foo/bar/", "bar"},
		{"normalize an invalid name", "path/prefix/foo_bar_baz", "foo-bar-baz"},
		{"normalize a valid name", "foo.bar.baz", "foo.bar.baz"},
		{"normalize a valid long name", "01234567890123456789012345678901234567890123456789012345678901234567890123456789", "01234567890123456789012345678901234567890123456789012345678901234567890123456789"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := common.MakeFileNameCompliant(tc.in)
			if output != tc.out {
				t.Fatalf("Failed to normalize the string %q to DNS-1123 standard properly. Expected: %q Actual: %q", tc.in, tc.out, output)
			}
		})
	}
}

func TestCleanAndFindCommonDirectory(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name string
		in   []string
		out  string
	}{
		{"find common directory when list is empty", []string{}, ""},
		{"normal use case", []string{"/foo/bar/baz", "/foo/bar", "/foo"}, "/foo"},
		{"normal use case and common directory is root", []string{"/foo/bar/baz", "/foo/bar", "/app1/service1/module1"}, "/"},
		{"find common directory when list has unclean paths", []string{"/app1/./service1/", "/app1/service1/module2/", "/app1/./service1/../service1/module1"}, "/app1/service1"},
		{"find common directory when list has unclean paths and common directory is root", []string{"/foo/bar///baz", "/foo/bar///.", "/app1/./service1/../service1/module1"}, "/"},
		{"list has identical paths", []string{"/foo/bar/baz", "/foo/bar/baz", "/foo/bar/baz"}, "/foo/bar/baz"},
		{"list contains root", []string{"/", "/.", "/..", "/.app/.bar", "/.app/.bar"}, "/"},
		{"list contains root", []string{"/foo/bar", "/", "/foo/bar/baz"}, "/"},
		{"list has identical but unclean paths", []string{"/foo/bar/baz////", "/foo/bar/baz", "/foo/bar/baz/././../baz/"}, "/foo/bar/baz"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := common.CleanAndFindCommonDirectory(tc.in)
			if output != tc.out {
				t.Fatal("Expected:", tc.out, "Actual:", output)
			}
		})
	}
}

func TestFindCommonDirectory(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	tcs := []struct {
		name string
		in   []string
		out  string
	}{
		{"find common directory when list is empty", []string{}, ""},
		{"normal use case", []string{"/foo/bar/baz", "/foo/bar", "/foo"}, "/foo"},
		{"normal use case and common directory is root", []string{"/foo/bar/baz", "/foo/bar", "/app1/service1/module1"}, "/"},
		{"list has identical paths", []string{"/foo/bar/baz", "/foo/bar/baz", "/foo/bar/baz"}, "/foo/bar/baz"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			output := common.FindCommonDirectory(tc.in)
			if output != tc.out {
				t.Fatal("Expected:", tc.out, "Actual:", output)
			}
		})
	}
}
