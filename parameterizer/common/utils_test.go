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

package common_test

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/parameterizer/common"
)

func TestGetSubKeys(t *testing.T) {
	testcases := []struct {
		input string
		want  []string
	}{
		{input: `aaa.bbb."ccc ddd".eee.fff`, want: []string{"aaa", "bbb", "ccc ddd", "eee", "fff"}},
		{input: "aaa.bbb.ccc", want: []string{"aaa", "bbb", "ccc"}},
	}
	for i, testcase := range testcases {
		t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
			subKeys := common.GetSubKeys(testcase.input)
			if len(subKeys) != len(testcase.want) {
				t.Fatalf("failed to get the correct number of subkeys. Expected %+v Actual %+v", testcase.want, subKeys)
			}
			same := true
			for i, subKey := range subKeys {
				if subKey != testcase.want[i] {
					same = false
					break
				}
			}
			if !same {
				t.Fatalf("failed to get the subkeys properly. Expected %+v Actual %+v", testcase.want, subKeys)
			}
		})
	}
}

func TestGet2(t *testing.T) {
	key := `"contain ers".[containerName:name=nginx].ports.[portName:name]`
	resource := map[string]interface{}{
		"foo": map[string]interface{}{
			"bar": 42,
		},
		"contain ers": []interface{}{
			map[string]interface{}{"name": "nginx", "image": "docker.io/foo/nginx:latest",
				"ports": []interface{}{
					map[string]interface{}{"name": "port1", "number": 8000},
					map[string]interface{}{"name": "port2", "number": 8080},
				},
			},
			map[string]interface{}{"name": "java", "image": "docker.io/bar/java:latest",
				"ports": []interface{}{
					map[string]interface{}{"name": "port1", "number": 4000},
					map[string]interface{}{"name": "port2", "number": 4080},
				},
			},
			map[string]interface{}{"name": "nginx", "image": "docker.io/foo/nginx:v1.2.0",
				"ports": []interface{}{
					map[string]interface{}{"name": "port1", "number": 1000},
					map[string]interface{}{"name": "port2", "number": 1080},
				},
			},
		},
	}
	want := []common.RT{
		{Key: []string{"contain ers", "[0]", "ports", "[0]"}, Value: map[string]interface{}{"name": "port1", "number": 8000}, Matches: map[string]string{"containerName": "nginx", "portName": "port1"}},
		{Key: []string{"contain ers", "[0]", "ports", "[1]"}, Value: map[string]interface{}{"name": "port2", "number": 8080}, Matches: map[string]string{"containerName": "nginx", "portName": "port2"}},
		{Key: []string{"contain ers", "[2]", "ports", "[0]"}, Value: map[string]interface{}{"name": "port1", "number": 1000}, Matches: map[string]string{"containerName": "nginx", "portName": "port1"}},
		{Key: []string{"contain ers", "[2]", "ports", "[1]"}, Value: map[string]interface{}{"name": "port2", "number": 1080}, Matches: map[string]string{"containerName": "nginx", "portName": "port2"}},
	}
	results, err := common.GetAll(key, resource)
	if err != nil {
		t.Fatalf("failed to get the values for the key %s Error: %q", key, err)
	}
	if !cmp.Equal(results, want) {
		t.Fatalf("differences %+v", cmp.Diff(results, want))
	}
}
