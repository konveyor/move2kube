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

func TestFileSystemUtils(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("test for copying file from source path to dest path with last modified time", func(t *testing.T) {
		testDataFile := "test.txt"
		destTempDir, err := ioutil.TempDir("", "destTempDir")
		if err != nil {
			t.Fatal("Error creating temp directory:", err)
		}
		defer os.RemoveAll(destTempDir)
		testDataDir := "testdata"
		sourceTempDir := "copyfile"

		destFilePath := filepath.Join(destTempDir, testDataFile)
		sourceFilePath, err := filepath.Abs(filepath.Join(testDataDir, sourceTempDir, testDataFile))
		if err != nil {
			t.Fatalf("unable to get absolute path for %s, %s. Error : %v", sourceTempDir, testDataFile, err)
		}
		sourceFileStat, err := os.Stat(sourceFilePath)
		if err != nil {
			t.Fatalf("Unable to stat file %s : %s", sourceFilePath, err)
		}
		sourceFileModTime := sourceFileStat.ModTime()
		err = copyFile(destFilePath, sourceFilePath, sourceFileModTime)
		if err != nil {
			t.Fatalf("copyFile failed for destination file path %v, source file path %v. Error : %v", destFilePath, sourceFilePath, err)
		}
		destinationFileStat, err := os.Stat(destFilePath)
		if err != nil {
			t.Fatalf("unable to stat file %s : %s", destFilePath, err)
		}
		destFileModTime := destinationFileStat.ModTime()
		if destFileModTime != sourceFileModTime {
			t.Fatalf("mod times did not match. need %v, got %v", sourceFileModTime, destFileModTime)
		}

		sourceFileCont, err := os.ReadFile(sourceFilePath)
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
