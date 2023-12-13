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
}
