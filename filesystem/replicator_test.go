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
	"testing"
)
 
 func TestReplicateProcessFileCallBack(t *testing.T) {
	 t.Run("test for source file does not exist, should return an error", func(t *testing.T) {
		 sourceFilePath := "test_data/replicator_test/non_existent_file.txt"
		 destinationFilePath := "test_data/replicator_test/destination.txt"
		 config := struct{}{}
		 err := replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		 if err == nil {
			 t.Errorf("Expected error for non-existent source file, got nil")
		 }
	 })
 
	 t.Run("test for destination file doesn't exist, should copy the source file", func(t *testing.T) {
		 sourceFilePath := "test_data/replicator_test/emptyfile.txt"
		 destinationFilePath := "test_data/replicator_test/destination_file.txt"
		 config := struct{}{}
 
		 err := replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		 if err != nil {
			 t.Errorf("Expected nil error, got: %s", err)
		 }
	 })
 
	 t.Run("test for destination file exists but is different, should copy the source file", func(t *testing.T) {
		 sourceFilePath := "test_data/replicator_test/source_file.txt"
		 destinationFilePath := "test_data/replicator_test/destination_file.txt"
		 config := struct{}{}
 
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
 
		 err = replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		 if err != nil {
			 t.Errorf("Expected nil error, got: %s", err)
		 }
 
		 // Check if the source file content matches the destination file content after copying
		 sourceContent, err := ioutil.ReadFile(sourceFilePath)
		 if err != nil {
			 t.Fatal(err)
		 }
 
		 destinationContent, err := ioutil.ReadFile(destinationFilePath)
		 if err != nil {
			 t.Fatal(err)
		 }
 
		 if !bytes.Equal(sourceContent, destinationContent) {
			 t.Errorf("Expected destination content to be equal to source content, but they are different.")
		 }
	 })
 
	 t.Run("test for destination file exists and is the same, should return nil without copying", func(t *testing.T) {
		 sourceFilePath := "test_data/replicator_test/source_file.txt"
		 destinationFilePath := "test_data/replicator_test/destination_file.txt"
		 config := struct{}{}
		 
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
 
		 _, err = destinationFile.WriteString("hello, world!")
		 if err != nil {
			 t.Fatal(err)
		 }
 
		 err = replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		 if err != nil {
			 t.Errorf("Expected nil error, got: %s", err)
		 }
	 })
 }
 