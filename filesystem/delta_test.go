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
	"path/filepath"
	"testing"
)

func TestGenerateDelta(t *testing.T) {
	t.Run("this test scenario covers the creation of source and destination directory and successful execution of function", func(t *testing.T) {

		tempDir, err := ioutil.TempDir("", "test_generate_delta")
		if err != nil {
			t.Fatalf("Failed to create temporary directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		// Create source and destination directories
		sourceDir := filepath.Join(tempDir, "source")
		destinationDir := filepath.Join(tempDir, "destination")
		if err := os.Mkdir(sourceDir, 0755); err != nil {
			t.Fatalf("Failed to create source directory: %v", err)
		}
		if err := os.Mkdir(destinationDir, 0755); err != nil {
			t.Fatalf("Failed to create destination directory: %v", err)
		}

		err = GenerateDelta(sourceDir, destinationDir, tempDir)

		// Check for errors
		if err != nil {
			t.Errorf("Unexpected error during GenerateDelta: %v", err)
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

		// Check for modifications file
		modificationsFilePath := filepath.Join(tempDir, "modifications", "file.txt")
		if _, err := os.Stat(modificationsFilePath); os.IsNotExist(err) {
			t.Errorf("Expected modifications file to be created, but it doesn't exist")
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
	})
}
