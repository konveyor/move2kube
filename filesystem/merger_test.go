
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

		// Additional checks based on the expected behavior after fixing the source path
		// For example, check the state of the destination directory or other behavior
		_, destErr := os.Stat(destination)
		if !os.IsNotExist(destErr) {
			t.Error("Expected destination directory not to exist, but it exists")
		}

		// Additional checks specific to your expected behavior can be added here
	})
}
