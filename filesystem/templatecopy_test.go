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

package filesystem

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestTemplateCopy(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("test for writing template to file", func(t *testing.T) {
		type TestTPLValues struct {
			TplVariable string
		}
		addOnConfig := AddOnConfig{Config: TestTPLValues{TplVariable: "World"}}
		testfilledFile := "test_filled.yaml"
		testDataFile := "test.yaml"
		templateDir := "template"
		testDir := "test_data"
		sourceFilePath := filepath.Join(testDir, templateDir, testDataFile)
		src, err := os.ReadFile(sourceFilePath)
		if err != nil {
			t.Fatalf("failed to read file at location %s. Error : %v", sourceFilePath, err)
		}
		destTempDir, err := ioutil.TempDir("", "destTempDir")
		if err != nil {
			t.Fatal("Error creating temp directory:", err)
		}
		defer os.RemoveAll(destTempDir)
		destFilePath := filepath.Join(destTempDir, testDataFile)
		si, err := os.Stat(sourceFilePath)
		if err != nil {
			t.Fatalf("failed to stat file at location %s. Error : %v", sourceFilePath, err)
		}
		err = writeTemplateToFile(string(src), addOnConfig.Config,
			destFilePath, si.Mode(),
			addOnConfig.OpeningDelimiter, addOnConfig.ClosingDelimiter)
		if err != nil {
			t.Fatalf("unable to copy templated file %s to %s : %s", sourceFilePath, destFilePath, err)
		}

		executedTemplateFile := filepath.Join(testDir, templateDir, testfilledFile)
		sourceFileCont, err := os.ReadFile(executedTemplateFile)
		sourceFileContStr := string(sourceFileCont)
		if err != nil {
			t.Fatalf("unable to read content from file %v. Error : %v", sourceFileContStr, err)
		}

		destFileCont, err := os.ReadFile(destFilePath)
		destFileContStr := string(destFileCont)
		if err != nil {
			t.Fatalf("unable to read content from file %v. Error : %v", destFileContStr, err)
		}

		if destFileContStr != sourceFileContStr {
			t.Fatalf("destination file and source file content mismatch, test failed.")
		}
	})

	t.Run("test for template copy delete call back", func(t *testing.T) {
		type TestTPLValues struct {
			TplVariable string
		}
		addOnConfig := AddOnConfig{Config: TestTPLValues{TplVariable: "world"}}
		testDataFile := "test.yaml"
		sourceTestDir := "template"
		testDataDir := "test_data"
		sourceFilePath := filepath.Join(testDataDir, sourceTestDir, testDataFile)
		destTempDir, err := ioutil.TempDir("", "destTempDir")
		if err != nil {
			t.Fatal("Error creating temp directory:", err)
		}
		defer os.RemoveAll(destTempDir)
		destFilePath := filepath.Join(destTempDir, testDataFile)
		err = templateCopyDeletionCallBack(sourceFilePath, destFilePath, addOnConfig)
		if err != nil {
			t.Fatalf("failed to do template copy deletion. Error : %v", err)
		}
	})

	t.Run("test for template copy process file", func(t *testing.T) {
		type TestTPLValues struct {
			TplVariable string
		}
		addOnConfig := AddOnConfig{Config: TestTPLValues{TplVariable: "World"}}
		testDataFile := "test.yaml"
		sourceTestDir := "template"
		testDataDir := "test_data"
		sourceFilePath := filepath.Join(testDataDir, sourceTestDir, testDataFile)
		destTempDir, err := ioutil.TempDir("", "destTempDir")
		if err != nil {
			t.Fatal("Error creating temp directory:", err)
		}
		defer os.RemoveAll(destTempDir)
		destFilePath := filepath.Join(destTempDir, testDataFile)

		err = templateCopyProcessFileCallBack(sourceFilePath, destFilePath, addOnConfig)
		if err != nil {
			t.Fatalf("unable to copy templated file %s to %s : %s", sourceFilePath, destFilePath, err)
		}

		executedTemplateFile := filepath.Join(testDataDir, sourceTestDir, "test_filled.yaml")
		sourceFileCont, err := os.ReadFile(executedTemplateFile)
		sourceFileContStr := string(sourceFileCont)
		if err != nil {
			t.Fatalf("unable to read content from file %v. Error : %v", sourceFileContStr, err)
		}

		destFileCont, err := os.ReadFile(destFilePath)
		destFileContStr := string(destFileCont)
		if err != nil {
			t.Fatalf("unable to read content from file %v. Error : %v", destFileContStr, err)
		}

		if destFileContStr != sourceFileContStr {
			t.Fatalf("destination file and source file content mismatch, test failed.")
		}
	})
}
