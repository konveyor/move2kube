/*
 *  Copyright IBM Corporation 2023
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

package lib

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/move2kube/common"
)

func TestLibUtils(t *testing.T) {

	t.Run("test for copying customizations assets data", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "tempBaseDir")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)
		customizationsPath := filepath.Join(tmpDir, "customizations")
		err = os.Mkdir(customizationsPath, 0755)
		if err != nil {
			t.Fatalf("failed to create customizations dir: %v", err)
		}

		err = CopyCustomizationsAssetsData("")
		if err != nil {
			t.Errorf("failed to check and copy customizations. Error : %v", err)
		}

		err = CopyCustomizationsAssetsData("invalid")
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Expected error %v, but got: %v", os.ErrNotExist, err)
		}

		err = CopyCustomizationsAssetsData(customizationsPath)
		if err != nil {
			t.Errorf("failed to check and copy customizations. Error : %v", err)
		}
		assetsPath, err := filepath.Abs(common.AssetsPath)
		if err != nil {
			t.Errorf("failed to make the assets path '%s' absolute. Error: %v", assetsPath, err)
		}
		customizationsAssetsPath := filepath.Join(assetsPath, "custom")
		if _, err := os.Stat(customizationsAssetsPath); os.IsNotExist(err) {
			t.Errorf("failed as customizations assets directory does not exist. Error : %v", err)
		}
	})

	t.Run("test for check and copy customizations", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "tempBaseDir")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)
		customizationsPath := filepath.Join(tmpDir, "customizations")
		err = os.Mkdir(customizationsPath, 0755)
		if err != nil {
			t.Fatalf("failed to create customizations dir: %v", err)
		}

		err = CheckAndCopyCustomizations("")
		if err != nil {
			t.Errorf("failed to check and copy customizations. Error : %v", err)
		}

		err = CheckAndCopyCustomizations("invalid")
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("Expected error %v, but got: %v", os.ErrNotExist, err)
		}

		err = CheckAndCopyCustomizations(customizationsPath)
		if err != nil {
			t.Errorf("failed to check and copy customizations. Error : %v", err)
		}
		assetsPath, err := filepath.Abs(common.AssetsPath)
		if err != nil {
			t.Errorf("failed to make the assets path '%s' absolute. Error: %v", assetsPath, err)
		}
		customizationsAssetsPath := filepath.Join(assetsPath, "custom")
		if _, err := os.Stat(customizationsAssetsPath); os.IsNotExist(err) {
			t.Errorf("failed as customizations assets directory does not exist. Error : %v", err)
		}
	})

}
