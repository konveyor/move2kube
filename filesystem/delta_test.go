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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGenerateDelta(t *testing.T) {
	t.Run("This test scenario covers the creation of source and destination directory and successful execution of function", func(t *testing.T) {
		sourceDir, err := ioutil.TempDir("", "source")
		if err != nil {
			t.Fatalf("failed to create source directory")
		}
		defer os.RemoveAll(sourceDir)

		destinationDir, err := ioutil.TempDir("", "destination")
		if err != nil {
			t.Fatalf("failed to create destination directory")
		}
		defer os.RemoveAll(destinationDir)

		storeDir, err := ioutil.TempDir("", "store")
		if err != nil {
			t.Fatalf("failed to create store directory")
		}
		defer os.RemoveAll(storeDir)

		// Add a file to the source directory that isn't in the destination
		sourceOnlyFile := filepath.Join(sourceDir, "source_only.txt")
		if err := ioutil.WriteFile(sourceOnlyFile, []byte("source only"), 0644); err != nil {
			t.Fatalf("Failed to add to source only file: %v", err)
		}

		// Add a file to both directories and then modify it in the source
		commonFile := "common.txt"
		sourceCommonFilePath := filepath.Join(sourceDir, commonFile)
		destCommonFilePath := filepath.Join(destinationDir, commonFile)
		if err := ioutil.WriteFile(sourceCommonFilePath, []byte("original"), 0644); err != nil {
			t.Fatalf("Failed to write original content to source common file: %v", err)

		}
		if err := ioutil.WriteFile(destCommonFilePath, []byte("original"), 0644); err != nil {
			t.Fatalf("Failed to write original content to destinatio common file: %v", err)

		}
		if err := ioutil.WriteFile(sourceCommonFilePath, []byte("modified"), 0644); err != nil {
			t.Fatalf("Failed to modify common file in source directory: %v", err)

		}

		// Call GenerateDelta
		err = GenerateDelta(sourceDir, destinationDir, storeDir)
		if err != nil {
			t.Errorf("GenerateDelta returned an error: %v", err)
		}

		// Check that the destination directory has the expected contents
		expectedFiles := []string{commonFile}
		files, err := ioutil.ReadDir(destinationDir)
		if err != nil {
			t.Errorf("destination directory contents mismatch. got: %v, want: %v", destFiles, expectedFiles)
		}

		var destFiles []string
		for _, file := range files {
			destFiles = append(destFiles, file.Name())
		}

		if !reflect.DeepEqual(expectedFiles, destFiles) {
			t.Errorf("destination directory contents mismatch. got: %v, want: %v", destFiles, expectedFiles)
		}

	})

	t.Run("this test covers the scenario for the presence of a modifications file and its content after calling the function", func(t *testing.T) {
		tempDir, err := ioutil.TempDir("", "test_generate_delta")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		sourceDir := filepath.Join(tempDir, "source")
		destinationDir := filepath.Join(tempDir, "destination")
		if err := os.Mkdir(sourceDir, 0755); err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}
		if err := os.Mkdir(destinationDir, 0755); err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}

		sourceFilePath := filepath.Join(sourceDir, "file.txt")
		if err := ioutil.WriteFile(sourceFilePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		err = GenerateDelta(sourceDir, destinationDir, tempDir)

		if err != nil {
			t.Errorf("Unexpected error during GenerateDelta: %v", err)
		}

		// Check for modifications file existence
		modificationsFilePath := filepath.Join(tempDir, "modifications", "file.txt")
		if _, err := os.Stat(modificationsFilePath); os.IsNotExist(err) {
			t.Errorf("Expected modifications file to be created, but it doesn't exist")
		}

		// Check modifications file contents
		contents, err := ioutil.ReadFile(modificationsFilePath)
		if err != nil {
			t.Errorf("Failed to read modifications file: %v", err)
		}

		expectedContents := []byte("content")
		if !bytes.Equal(contents, expectedContents) {
			t.Errorf("Modifications file contents mismatch. got: %v, want: %v", contents, expectedContents)
		}
	})

}

func TestGenerateDeltaAdditionCallBack(t *testing.T) {
	t.Run("this test covers the scenario where function correctly replicates a source file to the additions directory based on the provided configuration and if the replicated file is created in the expected location", func(t *testing.T) {

		tempDir, err := ioutil.TempDir("", "test_generate_delta_addition")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create source and destination files
		sourceFile := filepath.Join(tempDir, "source.txt")
		destinationDir := filepath.Join(tempDir, "destination")
		if err := os.Mkdir(destinationDir, 0755); err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}

		// Source file contents present
		content := []byte("Hello, move2kubee!")
		if err := ioutil.WriteFile(sourceFile, content, 0644); err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		config := generateDeltaConfig{
			store:                tempDir,
			sourceDirectory:      tempDir,
			destinationDirectory: destinationDir,
		}

		err = generateDeltaAdditionCallBack(sourceFile, filepath.Join(destinationDir, "newfile.txt"), config)

		if err != nil {
			t.Errorf("Unexpected error during generateDeltaAdditionCallBack: %v", err)
		}

		replicatedFilePath := filepath.Join(tempDir, additionsDir, "newfile.txt")
		if _, err := os.Stat(replicatedFilePath); os.IsNotExist(err) {
			t.Errorf("Replicated file not found at expected path: %s", replicatedFilePath)
		}

		// Check the contents of the replicated file
		replicatedContent, err := ioutil.ReadFile(replicatedFilePath)
		if err != nil {
			t.Fatalf("Failed to read replicated file: %v", err)
		}

		// Define the expected content
		expectedContent := []byte("Hello, move2kubee!")

		// Compare the contents
		if !bytes.Equal(replicatedContent, expectedContent) {
			t.Errorf("Replicated file content mismatch. got: %s, want: %s", replicatedContent, expectedContent)
		}
	})
}
