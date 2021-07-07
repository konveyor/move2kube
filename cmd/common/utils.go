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

package common

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	internalcommon "github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"
)

const (
	// SourceFlag is the name of the flag that contains path to the source folder
	SourceFlag = "source"
	// OutputFlag is the name of the flag that contains path to the output folder
	OutputFlag = "output"
	// NameFlag is the name of the flag that contains the project name
	NameFlag = "name"
	// PlanFlag is the name of the flag that contains the path to the plan file
	PlanFlag = "plan"
	// IgnoreEnvFlag is the name of the flag that tells us whether to use data collected from the local machine
	IgnoreEnvFlag = "ignoreenv"
	// QASkipFlag is the name of the flag that lets you skip all the question answers
	QASkipFlag = "qaskip"
	// ConfigOutFlag is the name of the flag that will point the location to output the config file
	ConfigOutFlag = "configout"
	// QACacheOutFlag is the name of the flag that will point the location to output the cache file
	QACacheOutFlag = "qacacheout"
	// ConfigFlag is the name of the flag that contains list of config files
	ConfigFlag = "config"
	// SetConfigFlag is the name of the flag that contains list of key-value configs
	SetConfigFlag = "setconfig"
	// PreSetFlag is the name of the flag that contains list of preset configurations to use
	PreSetFlag = "preset"
	// OverwriteFlag is the name of the flag that lets you overwrite the output directory if it exists
	OverwriteFlag = "overwrite"
	// CustomizationsFlag is the path to customizations directory
	CustomizationsFlag = "customizations"
)

// TransformFlags to store values from command line parameters
type TransformFlags struct {
	// IgnoreEnv tells us whether to use data collected from the local machine
	IgnoreEnv bool
	// Planfile is contains the path to the plan file
	Planfile string
	// Outpath contains the path to the output folder
	Outpath string
	// SourceFlag contains path to the source folder
	Srcpath string
	// Name contains the project name
	Name string
	// ConfigOut contains the location to output the config
	ConfigOut string
	// QACacheOut contains the location to output the cache
	QACacheOut string
	// Configs contains a list of config files
	Configs []string
	// Configs contains a list of key-value configs
	Setconfigs []string
	// Qaskip lets you skip all the question answers
	Qaskip bool
	// Overwrite lets you overwrite the output directory if it exists
	Overwrite bool
	// PreSets contains a list of preset configurations
	PreSets []string
	// CustomizationsPaths contains the path to the customizations directory
	CustomizationsPath string
}

// CheckSourcePath checks if the source path is an existing directory.
func CheckSourcePath(srcpath string) {
	fi, err := os.Stat(srcpath)
	if os.IsNotExist(err) {
		logrus.Fatalf("The given source directory %s does not exist. Error: %q", srcpath, err)
	}
	if err != nil {
		logrus.Fatalf("Error while accessing the given source directory %s Error: %q", srcpath, err)
	}
	if !fi.IsDir() {
		logrus.Fatalf("The given source path %s is a file. Expected a directory. Exiting.", srcpath)
	}
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Failed to get the current working directory. Error: %q", err)
	}
	if internalcommon.IsParent(pwd, srcpath) {
		logrus.Fatalf("The given source directory %s is a parent of the current working directory.", srcpath)
	}
}

// CheckOutputPath checks if the output path is already in use.
func CheckOutputPath(outpath string, overwrite bool) {
	fi, err := os.Stat(outpath)
	if os.IsNotExist(err) {
		logrus.Debugf("Transformed artifacts will be written to %s", outpath)
		return
	}
	if err != nil {
		logrus.Fatalf("Error while accessing output directory at path %s Error: %q . Exiting", outpath, err)
	}
	if !overwrite {
		logrus.Fatalf("Output directory %s exists. Exiting", outpath)
	}
	if !fi.IsDir() {
		logrus.Fatalf("Output path %s is a file. Expected a directory. Exiting", outpath)
	}
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Failed to get the current working directory. Error: %q", err)
	}
	if internalcommon.IsParent(pwd, outpath) {
		logrus.Fatalf("The given output directory %s is a parent of the current working directory.", outpath)
	}
	logrus.Infof("Output directory %s exists. The contents might get overwritten.", outpath)
}

// NormalizePaths cleans the paths and makes them absolute
// If any of the paths are directories it will walk through
// them collecting paths to file having a particular set of extensions
func NormalizePaths(paths []string, extensions []string) ([]string, error) {
	newPaths := []string{}
	for _, path := range paths {
		newPath, err := filepath.Abs(path)
		if err != nil {
			return newPaths, fmt.Errorf("failed to make the path %s absolute. Error: %q", path, err)
		}
		finfo, err := os.Stat(newPath)
		if err != nil {
			if os.IsNotExist(err) {
				logrus.Errorf("The path %s does not exist.", newPath)
			} else {
				logrus.Errorf("Failed to access the path %s . Error: %q", newPath, err)
			}
			continue
		}
		if !finfo.IsDir() {
			newPaths = append(newPaths, newPath)
			continue
		}
		err = filepath.Walk(newPath, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && internalcommon.IsStringPresent(extensions, filepath.Ext(path)) {
				newPaths = append(newPaths, path)
			}
			return nil
		})
		if err != nil {
			logrus.Warnf("Failed to walk through the files in the directory %s . Error: %q", newPath, err)
		}
	}
	return newPaths, nil
}
