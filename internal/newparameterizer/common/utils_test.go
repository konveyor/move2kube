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
	"testing"

	"github.com/konveyor/move2kube/internal/newparameterizer/common"
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
