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
	 "testing"
	 "github.com/stretchr/testify/assert"
	 
 )
 
 func TestNewProcessor(t *testing.T) {
	 t.Run("checks for a processor instance with given options", func(t *testing.T) {
		 // Create a sample options struct
		 mockOptions := options{
			 processFileCallBack: func(sourcecFilePath, destinationFilePath string, config interface{}) (err error) {
				 return nil
			 },
			 additionCallBack: func(sourcePath, destinationPath string, config interface{}) (err error) {
				 return nil
			 },
			 deletionCallBack: func(sourcePath, destinationPath string, config interface{}) (err error) {
				 return nil
			 },
			 mismatchCallBack: func(sourcePath, destinationPath string, config interface{}) (err error) {
				 return nil
			 },
			 config: nil,
		 }
 
		
		 processor := newProcessor(mockOptions)
 
		 // Check that the returned processor is not nil
		 if processor == nil {
			 t.Error("Expected non-nil processor, but got nil")
		 }
 
		
	 })
 }
 func TestProcessFile(t *testing.T) {
 
    t.Run("test for the scenario where the processing of a file is successful", func(t *testing.T) {
	source := "path/to/source/file.txt"
    destination := "path/to/destination/file.txt"
    options := options{
        processFileCallBack: func(sourcecFilePath, destinationFilePath string, config interface{}) (err error) {
            
            return nil
        },
        additionCallBack: func(sourcePath, destinationPath string, config interface{}) (err error) {
           
            return nil
        },
        deletionCallBack: func(sourcePath, destinationPath string, config interface{}) (err error) {
          
            return nil
        },
        mismatchCallBack: func(sourcePath, destinationPath string, config interface{}) (err error) {
           
            return nil
        },
        config: nil,
    }
    processor := newProcessor(options)
        // Invoke the code under test
        err := processor.processFile(source, destination)

        // Assert the result
        assert.NoError(t, err)
    })
}
 