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

package lib

import (
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/filesystem"
	"github.com/sirupsen/logrus"
)

// CheckAndCopyCustomizations checks if the customizations path is an existing directory and copies to assets
func CheckAndCopyCustomizations(customizationsPath string) {
	if customizationsPath == "" {
		return
	}
	customizationsPath, err := filepath.Abs(customizationsPath)
	if err != nil {
		logrus.Fatalf("Unable to make the customizations directory path %q absolute. Error: %q", customizationsPath, err)
	}
	fi, err := os.Stat(customizationsPath)
	if os.IsNotExist(err) {
		logrus.Fatalf("The given customizations directory %s does not exist. Error: %q", customizationsPath, err)
	}
	if err != nil {
		logrus.Fatalf("Error while accessing the given customizations directory %s Error: %q", customizationsPath, err)
	}
	if !fi.IsDir() {
		logrus.Fatalf("The given customizations path %s is a file. Expected a directory. Exiting.", customizationsPath)
	}
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Failed to get the current working directory. Error: %q", err)
	}
	if common.IsParent(pwd, customizationsPath) {
		logrus.Fatalf("The given customizations directory %s is a parent of the current working directory.", customizationsPath)
	}
	if err = CopyCustomizationsAssetsData(customizationsPath); err != nil {
		logrus.Fatalf("Unable to copy customizations data : %s", err)
	}
}

// CopyCustomizationsAssetsData copies an customizations to the assets directory
func CopyCustomizationsAssetsData(customizationsPath string) (err error) {
	if customizationsPath == "" {
		return nil
	}
	// Return the absolute version of customizations directory.
	customizationsPath, err = filepath.Abs(customizationsPath)
	if err != nil {
		logrus.Errorf("Unable to make the customizations directory path %q absolute. Error: %q", customizationsPath, err)
		return err
	}
	assetsPath, err := filepath.Abs(common.AssetsPath)
	if err != nil {
		logrus.Errorf("Unable to make the assets path %q absolute. Error: %q", assetsPath, err)
		return err
	}
	customizationsAssetsPath := filepath.Join(assetsPath, "custom")

	// Create the subdirectory and copy the assets into it.
	if err = os.MkdirAll(customizationsAssetsPath, common.DefaultDirectoryPermission); err != nil {
		logrus.Errorf("Unable to create the custom assets directory at path %q Error: %q", customizationsAssetsPath, err)
		return err
	}
	if err = filesystem.Replicate(customizationsPath, customizationsAssetsPath); err != nil {
		logrus.Errorf("Failed to copy the customizations %s over to the directory at path %s Error: %q", customizationsPath, customizationsAssetsPath, err)
		return err
	}

	return nil
}
