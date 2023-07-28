package filesystem

import (
	"os"
	"testing"
	"time"
)

func TestReplicateProcessFileCallBack(t *testing.T) {
	t.Run("NonExistentSourceFile", func(t *testing.T) {
		sourceFilePath := "non_existent_file.txt"
		destinationFilePath := "destination.txt"
		config := struct{}{}
		err := replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		if err == nil {
			t.Errorf("Expected error for non-existent source file, got nil")
		}
	})

	t.Run("CopySourceToDestination", func(t *testing.T) {
		sourceFile, err := os.Create("source_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			sourceFile.Close()
			os.Remove("source_file.txt")
		}()

		sourceFilePath := "source_file.txt"
		destinationFilePath := "destination_file.txt"
		config := struct{}{}

		err = replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		if err != nil {
			t.Errorf("Expected nil error, got: %s", err)
		}
	})

	t.Run("CopySourceToDestinationDifferent", func(t *testing.T) {
		sourceFile, err := os.Create("source_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			sourceFile.Close()
			os.Remove("source_file.txt")
		}()

		_, err = sourceFile.WriteString("hello, world!")
		if err != nil {
			t.Fatal(err)
		}

		sourceFile.Close() // Close and reopen the file to update the modification time
		time.Sleep(1 * time.Second)

		destinationFile, err := os.Create("destination_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			destinationFile.Close()
			os.Remove("destination_file.txt")
		}()

		_, err = destinationFile.WriteString("hello, go!")
		if err != nil {
			t.Fatal(err)
		}

		sourceFilePath := "source_file.txt"
		destinationFilePath := "destination_file.txt"
		config := struct{}{}

		err = replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		if err != nil {
			t.Errorf("Expected nil error, got: %s", err)
		}
	})

	t.Run("CopySourceToDestinationSame", func(t *testing.T) {
		sourceFile, err := os.Create("source_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			sourceFile.Close()
			os.Remove("source_file.txt")
		}()

		_, err = sourceFile.WriteString("hello, world!")
		if err != nil {
			t.Fatal(err)
		}

		destinationFile, err := os.Create("destination_file.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer func() {
			destinationFile.Close()
			os.Remove("destination_file.txt")
		}()

		_, err = destinationFile.WriteString("hello, world!")
		if err != nil {
			t.Fatal(err)
		}


		sourceFilePath := "source_file.txt"
		destinationFilePath := "destination_file.txt"
		config := struct{}{}

		err = replicateProcessFileCallBack(sourceFilePath, destinationFilePath, config)
		if err != nil {
			t.Errorf("Expected nil error, got: %s", err)
		}
	})
}
