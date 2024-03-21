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
	"io/ioutil"
	"os"
	"testing"
)

func TestMergeDeletionCallBack(t *testing.T) {
	t.Run(" test for scenario when function is called with a non-existent source file and an existing destination directory", func(t *testing.T) {
		source := "/move2kube/nonexistent/source"
		destination := "/move2kube/destination"
		config := interface{}(nil)

		// Call the merge function
		err := mergeDeletionCallBack(source, destination, config)

		// Assert that the error is not nil (as the source file does not exist)
		if err == nil {
			t.Fatalf("Expected non-nil error, but got nil")
		}

		// Check if the error is due to the source file not existing
		if !os.IsNotExist(err) {
			t.Fatalf("Expected source file to not exist, but got error: %v", err)
		}

		_, destErr := os.Stat(destination)
		if !os.IsNotExist(destErr) {
			t.Error("Expected destination directory not to exist, but it exists")
		}

	})
	t.Run("test for scenario which checks for permissions of the destination directory to match the permissions of the source file when both the source file and destination directory exist", func(t *testing.T) {
		sourceDir, err := ioutil.TempDir("", "sourceDir")
		if err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}
		defer os.RemoveAll(sourceDir)

		// Create a temporary file within the source directory
		sourceFile, err := ioutil.TempFile(sourceDir, "sourceFile.txt")
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		defer os.Remove(sourceFile.Name())

		// Create a temporary destination directory
		destinationDir, err := ioutil.TempDir("", "destinationDir")
		if err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}
		defer os.RemoveAll(destinationDir)

		// Call the mergedeletion callback function
		err = mergeDeletionCallBack(sourceFile.Name(), destinationDir, nil)
		if err != nil {
			t.Fatalf("Unexpected error during merge: %v", err)
		}

		// Check if the destination directory exists
		destInfo, destErr := os.Stat(destinationDir)
		if destErr != nil {
			t.Fatalf("Failed to get destination directory info: %v", destErr)
		}

		// Checking if the destination is indeed a directory or not
		if !destInfo.IsDir() {
			t.Fatalf("Expected destination to be a directory but it is not.")
		}

		// Check if the destination directory permissions match the source file
		sourceFileInfo, sourceFileErr := os.Stat(sourceFile.Name())
		if sourceFileErr != nil {
			t.Fatalf("Failed to get source file info: %v", sourceFileErr)
		}

		if destInfo.Mode().Perm() != sourceFileInfo.Mode().Perm() {
			t.Errorf("Expected destination directory permissions to be %s, but got %s", sourceFileInfo.Mode().Perm(), destInfo.Mode().Perm())
		}
	})
}

func TestMergeProcessFileCallBack(t *testing.T) {
	t.Run("test for source and destination files with same content", func(t *testing.T) {
		nonExistentPath := ""

		sourceFile, err := ioutil.TempFile(nonExistentPath, "source")
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		defer os.Remove(sourceFile.Name())

		destinationFile, err := ioutil.TempFile(nonExistentPath, "destination")
		if err != nil {
			t.Fatalf("Failed to create destination file: %v", err)
		}
		defer os.Remove(destinationFile.Name())

		// Write content to both files
		content := "same content"
		_, err = sourceFile.WriteString(content)
		if err != nil {
			t.Fatalf("Failed to write to source file: %v", err)
		}
		_, err = destinationFile.WriteString(content)
		if err != nil {
			t.Fatalf("Failed to write to destination file: %v", err)
		}

		err = mergeProcessFileCallBack(sourceFile.Name(), destinationFile.Name(), false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Assert that the destination file content is not updated
		updatedContent, err := ioutil.ReadFile(destinationFile.Name())
		if err != nil {
			t.Fatalf("Failed to read destination file: %v", err)
		}
		if string(updatedContent) != content {
			t.Errorf("Destination file content should not be updated")
		}
	})
	t.Run("test for source and destination files with different content", func(t *testing.T) {
		nonExistentPath := ""

		sourceFile, err := ioutil.TempFile(nonExistentPath, "source")
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}
		defer os.Remove(sourceFile.Name())

		destinationFile, err := ioutil.TempFile(nonExistentPath, "destination")
		if err != nil {
			t.Fatalf("Failed to create destination file: %v", err)
		}
		defer os.Remove(destinationFile.Name())

		// Write different content to both files
		sourceContent := "source content"
		destinationContent := "different content"
		_, err = sourceFile.WriteString(sourceContent)
		if err != nil {
			t.Fatalf("Failed to write to source file: %v", err)
		}
		_, err = destinationFile.WriteString(destinationContent)
		if err != nil {
			t.Fatalf("Failed to write to destination file: %v", err)
		}

		err = mergeProcessFileCallBack(sourceFile.Name(), destinationFile.Name(), false)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// checking for destination file content is not updated
		updatedContent, err := ioutil.ReadFile(destinationFile.Name())
		if err != nil {
			t.Fatalf("Failed to read destination file: %v", err)
		}
		if string(updatedContent) != sourceContent {
			t.Errorf("Destination file content should be updated to match the source")
		}
	})
}
