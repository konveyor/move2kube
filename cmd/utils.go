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

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/konveyor/move2kube/assets"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	qaenginetypes "github.com/konveyor/move2kube/types/qaengine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

// checkSourcePath checks if the source path is an existing directory.
func checkSourcePath(srcpath string) {
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
	if common.IsParent(pwd, srcpath) {
		logrus.Fatalf("The given source directory %s is a parent of the current working directory.", srcpath)
	}
}

// checkOutputPath checks if the output path is already in use.
func checkOutputPath(outpath string, overwrite bool) {
	fi, err := os.Stat(outpath)
	if os.IsNotExist(err) {
		logrus.Debugf("Transformed artifacts will be written to a directory at path '%s'", outpath)
		return
	}
	if err != nil {
		logrus.Fatalf("Error while accessing output directory at path '%s' Error: %q . Exiting", outpath, err)
	}
	if !overwrite {
		logrus.Fatalf(`The output directory '%s' already exists.
Please either:
- remove the output directory
- specify the '--%s' flag to overwrite the output directory
- use the '--%s' flag to specify a different output directory
Exiting.`, outpath, overwriteFlag, outputFlag)
	}
	if !fi.IsDir() {
		logrus.Fatalf("Output path '%s' is a file. Expected a directory. Exiting", outpath)
	}
	pwd, err := os.Getwd()
	if err != nil {
		logrus.Fatalf("Failed to get the current working directory. Error: %q", err)
	}
	if common.IsParent(pwd, outpath) {
		logrus.Fatalf("The given output directory '%s' is a parent of the current working directory.", outpath)
	}
	logrus.Infof("Output directory '%s' exists. The contents might get overwritten.", outpath)
}

func getCustomMappingFilePath() (qaenginetypes.QAMappings, error) {
	// Read the QA categories from the QA mapping file
	var qaMapping qaenginetypes.QAMappings
	customizationsDir := filepath.Join(common.AssetsPath, "custom")
	_, err := os.Stat(customizationsDir)
	if err == nil {
		yamlPaths, err := common.GetFilesByExt(customizationsDir, []string{".yml", ".yaml"})
		if err != nil {
			return qaMapping, fmt.Errorf("failed to look for yaml files in the directory '%s' . Error: %w", customizationsDir, err)
		}
		for _, yamlPath := range yamlPaths {
			if err := common.ReadMove2KubeYaml(yamlPath, &qaMapping); err != nil {
				logrus.Debugf("failed to read the mappings file metadata from the yaml file at path '%s' . Error: %q", yamlPath, err)
				continue
			}
			if qaMapping.Kind != qaenginetypes.QAMappingsKind {
				logrus.Debugf(
					"the file at path '%s' is not a valid cluster metadata. Expected kind: '%s' Actual kind: '%s'",
					yamlPath, qaenginetypes.QAMappingsKind, qaMapping.Kind,
				)
				continue
			}
			logrus.Infof("Found custom QA mappings file '%s' at path '%s'", qaMapping.ObjectMeta.Name, yamlPath)
			return qaMapping, nil
		}
	}
	logrus.Infof("Using the default QA mappings file")
	qaMappingFilepath := filepath.Join("built-in/qa", "qamappings.yaml")
	file, err := assets.AssetsDir.ReadFile(qaMappingFilepath)
	if err != nil {
		return qaMapping, fmt.Errorf("failed to read the mappings file metadata from the yaml file at path '%s' . Error: %w", qaMappingFilepath, err)
	}
	if err := yaml.Unmarshal(file, &qaMapping); err != nil {
		return qaMapping, fmt.Errorf("failed to decode qa-mapping file. Error: %w", err)
	}
	return qaMapping, nil
}

func initDisabledCategories(flags qaflags) {
	// qa-disable and qa=enable are mutually exclusive
	if len(flags.qaEnabledCategories) > 0 && len(flags.qaDisabledCategories) > 0 {
		logrus.Fatalf("--qa-enable and --qa-disable cannot be used together.\n")
	}
	// Read the QA categories from the QA mapping file
	qaMapping, err := getCustomMappingFilePath()
	if err != nil {
		logrus.Fatalf("failed to read the QAMappings file. Error: %q", err)
	}
	for _, category := range qaMapping.Spec.Categories {
		common.QACategoryMap[category.Name] = category.Questions
	}
	common.QACategoryMap["default"] = []string{}
	common.QACategoryMap["external"] = []string{}
	// if --qa-enable is passed, all categories are disabled by default. Otherwise, only categories passed to --qa-disable
	// are disabled
	for _, category := range qaMapping.Spec.Categories {
		if !category.Enabled {
			common.DisabledCategories = append(common.DisabledCategories, category.Name)
		}
	}
	if len(flags.qaEnabledCategories) > 0 {
		for k := range common.QACategoryMap {
			if !common.IsStringPresent(flags.qaEnabledCategories, k) {
				common.DisabledCategories = append(common.DisabledCategories, k)
			}
		}
	} else {
		common.DisabledCategories = append(common.DisabledCategories, flags.qaDisabledCategories...)
	}
	if len(common.DisabledCategories) > 0 {
		logrus.Infof("Disabling the questions in the following categories: %v", common.DisabledCategories)
	}
}

func startQA(flags qaflags) {
	initDisabledCategories(flags)
	qaengine.StartEngine(flags.qaskip, flags.qaport, flags.qadisablecli)
	if flags.configOut == "" {
		qaengine.SetupConfigFile("", flags.setconfigs, flags.configs, flags.preSets, flags.persistPasswords)
	} else {
		if flags.configOut == "." {
			qaengine.SetupConfigFile(common.ConfigFile, flags.setconfigs, flags.configs, flags.preSets, flags.persistPasswords)
		} else if fi, err := os.Stat(flags.configOut); err == nil {
			if fi.IsDir() {
				qaengine.SetupConfigFile(filepath.Join(flags.configOut, common.ConfigFile), flags.setconfigs, flags.configs, flags.preSets, flags.persistPasswords)
			} else {
				qaengine.SetupConfigFile(flags.configOut, flags.setconfigs, flags.configs, flags.preSets, flags.persistPasswords)
			}
		} else if strings.Contains(filepath.Base(flags.configOut), ".") {
			os.MkdirAll(filepath.Dir(flags.configOut), common.DefaultDirectoryPermission)
			qaengine.SetupConfigFile(flags.configOut, flags.setconfigs, flags.configs, flags.preSets, flags.persistPasswords)
		} else {
			os.MkdirAll(flags.configOut, common.DefaultDirectoryPermission)
			qaengine.SetupConfigFile(filepath.Join(flags.configOut, common.ConfigFile), flags.setconfigs, flags.configs, flags.preSets, flags.persistPasswords)
		}
	}
	if flags.qaCacheOut != "" {
		if flags.qaCacheOut == "." {
			qaengine.SetupWriteCacheFile(common.QACacheFile, flags.persistPasswords)
		} else if fi, err := os.Stat(flags.qaCacheOut); err == nil {
			if fi.IsDir() {
				qaengine.SetupWriteCacheFile(filepath.Join(flags.qaCacheOut, common.QACacheFile), flags.persistPasswords)
			} else {
				qaengine.SetupWriteCacheFile(flags.qaCacheOut, flags.persistPasswords)
			}
		} else if strings.Contains(filepath.Base(flags.qaCacheOut), ".") {
			os.MkdirAll(filepath.Dir(flags.qaCacheOut), common.DefaultDirectoryPermission)
			qaengine.SetupWriteCacheFile(flags.qaCacheOut, flags.persistPasswords)
		} else {
			os.MkdirAll(flags.qaCacheOut, common.DefaultDirectoryPermission)
			qaengine.SetupWriteCacheFile(filepath.Join(flags.qaCacheOut, common.QACacheFile), flags.persistPasswords)
		}
	}
	if err := qaengine.WriteStoresToDisk(); err != nil {
		logrus.Warnf("Failed to write the stores to disk. Error: %q", err)
	}
}

func startPlanProgressServer(port int) {
	logrus.Trace("startPlanProgressServer start")
	var server http.Server
	r := mux.NewRouter()
	r.HandleFunc("/progress", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"files": common.PlanProgressNumDirectories, "transformers": common.PlanProgressNumBaseDetectTransformers})
	}).Methods("GET")
	server.Handler = r
	server.Addr = ":" + cast.ToString(port)
	go func() {
		logrus.Debugf("listening on port %d", port)
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logrus.Errorf("failed to shutdown the plan progress server gracefully. Error: %q", err)
		}
	}()
	logrus.Trace("startPlanProgressServer end")
}
