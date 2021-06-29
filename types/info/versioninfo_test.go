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

package info_test

import (
	"testing"

	"github.com/konveyor/move2kube/types/info"
)

func TestGetVersionInfo(t *testing.T) {
	vinfo := info.GetVersionInfo()
	if !vinfo.IsSameVersion() {
		t.Fatal("Versions don't match. Expected:", info.GetVersion(), "Actual:", vinfo.Version)
	}
}

func TestIsSameVersion(t *testing.T) {
	t.Run("same version", func(t *testing.T) {
		vinfo := info.GetVersionInfo()
		if !vinfo.IsSameVersion() {
			t.Fatal("The versions are same but the method says they are different. Binary version:", info.GetVersion(), "Object version:", vinfo.Version)
		}
	})

	t.Run("older version", func(t *testing.T) {
		vinfo := info.GetVersionInfo()
		vinfo.Version = "0.0.0"
		if vinfo.IsSameVersion() {
			t.Fatal("The versions are different but the method says they are equal. Binary version:", info.GetVersion(), "Object version:", vinfo.Version)
		}
	})

	t.Run("newer version", func(t *testing.T) {
		vinfo := info.GetVersionInfo()
		vinfo.Version = "100.0.0"
		if vinfo.IsSameVersion() {
			t.Fatal("The versions are different but the method says they are equal. Binary version:", info.GetVersion(), "Object version:", vinfo.Version)
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		vinfo := info.GetVersionInfo()
		vinfo.Version = "foobar"
		if vinfo.IsSameVersion() {
			t.Fatal("The versions are different but the method says they are equal. Binary version:", info.GetVersion(), "Object version:", vinfo.Version)
		}
	})
}
