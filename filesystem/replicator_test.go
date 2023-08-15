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

package filesystem

import (
	"bytes"
	"os"
	"testing"
)

func TestReplicateProcessFileCallBack(t *testing.T) {
	t.Run("test for source file does not exist, should return an error", func(t *testing.T) {
		sourceFilePath := "testdata/replicator_test/non_existent_file.txt"
		destinationFilePath := "testdata/replicator_test/destination.txt"
		err := replicateProcessFileCallBack(sourceFilePath, destinationFilePath, nil)
		if err == nil {
			t.Errorf("Expected error for non-existent source file, got nil")
		}
	})

	t.Run("test for destination file doesn't exist, should copy the source file", func(t *testing.T) {
		sourceFilePath := "testdata/replicator_test/emptyfile.txt"
		destinationFilePath := "testdata/replicator_test/destination_file.txt"

		err := replicateProcessFileCallBack(sourceFilePath, destinationFilePath, nil)
		if err != nil {
			t.Errorf("Expected nil error, got: %s", err)
		}

		os.Remove(destinationFilePath)
	})

	t.Run("test for destination file exists but is different, should copy the source file", func(t *testing.T) {
		sourceFilePath := "testdata/replicator_test/source_file.txt"
		destinationFilePath := "testdata/replicator_test/destination_file.txt"

		sourceFile, err := os.Create(sourceFilePath)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			sourceFile.Close()
			os.Remove(sourceFilePath)
		}()

		_, err = sourceFile.WriteString("hello, world!")
		if err != nil {
			t.Fatal(err)
		}

		destinationFile, err := os.Create(destinationFilePath)
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			destinationFile.Close()
			os.Remove(destinationFilePath)
		}()

		_, err = destinationFile.WriteString("hello, go!")
		if err != nil {
			t.Fatal(err)
		}

		err = replicateProcessFileCallBack(sourceFilePath, destinationFilePath, nil)
		if err != nil {
			t.Errorf("Expected nil error, got: %s", err)
		}

		// Check if the source file content matches the destination file content after copying
		sourceContent, err := os.ReadFile(sourceFilePath)
		if err != nil {
			t.Fatal(err)
		}

		destinationContent, err := os.ReadFile(destinationFilePath)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(sourceContent, destinationContent) {
			t.Errorf("Expected destination content to be equal to source content, but they are different.")
		}
	})

	t.Run("test for destination file exists and is the same, should return nil without copying", func(t *testing.T) {
		sourceFilePath := "testdata/replicator_test/source_same.txt"
		destinationFilePath := "testdata/replicator_test/destination_same.txt"

		// Store the destination file's modification time before calling replicateProcessFileCallBack
		info, err := os.Stat(destinationFilePath)
		if err != nil {
			t.Errorf("Destination file should exist, but got an error: %s", err)
		}
		originalModTime := info.ModTime()
		err = replicateProcessFileCallBack(sourceFilePath, destinationFilePath, nil)
		if err != nil {
			t.Errorf("Expected nil error, got: %s", err)
		}

		// Check if the destination file still exists after calling replicateProcessFileCallBack
		info, err = os.Stat(destinationFilePath)
		if err != nil {
			t.Errorf("Destination file should exist, but got an error: %s", err)
		}

		// Check if the destination file's modification time hasn't changed
		updatedModTime := info.ModTime()
		if !updatedModTime.Equal(originalModTime) {
			t.Errorf("Destination file's modification time has changed, expected: %v, got: %v", originalModTime, updatedModTime)
		}

		// Check if the destination file content remains unchanged
		destinationContent, err := os.ReadFile(destinationFilePath)
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(destinationContent, []byte("hello, world!")) {
			t.Errorf("Expected destination content to remain unchanged, but got different content.")
		}
	 })
 }
 func TestReplicateAdditionCallBack(t *testing.T) {
	t.Run("test for destination dir is removed", func(t *testing.T) {
		sourceDir := "testdata/source"
		destinationDir := "testdata/destination"
		destinationFilePath := destinationDir + "/existing-file.txt"
		// Prepare the destination directory with an existing file
		err := os.MkdirAll(destinationDir, 0755)
		if err != nil {
			t.Fatal(err)
		}
		_, err = os.Create(destinationFilePath)
		 if err != nil {
			 t.Fatal(err)
		 }

		// Call the addition callback
		err = replicateAdditionCallBack(sourceDir, destinationDir, nil)
		if err != nil {
			t.Fatalf("Expected no error, got %s", err)
		}

		// Check if the destination directory is removed
		_, err = os.Stat(destinationDir)
		if err == nil {
			t.Error("Expected destination directory to be removed, but it still exists")
		}
	})
}
 