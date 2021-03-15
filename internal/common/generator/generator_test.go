// +build !excludecodegen

/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMakeConstants(t *testing.T) {
	t.Run("try to generate code for non existent directory", func(t *testing.T) {
		path := "testdata/nonexistent"
		if err := makeConstants(path, maptemp); err == nil {
			t.Fatalf("Should have failed since the directory %q does not exist.", path)
		}
	})

	t.Run("read empty directory and generate code with maptemp", func(t *testing.T) {
		// Setup
		testparentdir := t.TempDir()
		testdir := filepath.Join(testparentdir, "foobar")
		if err := os.Mkdir(testdir, os.ModePerm); err != nil {
			t.Fatalf("Failed to create the test directory at path %q. Error: %q", testdir, err)
		}
		fpath := filepath.Join(testdir, "constants.go")
		testdatapath := "testdata/maptempempty.txt"
		testdatabytes, err := ioutil.ReadFile(testdatapath)
		if err != nil {
			t.Fatalf("Failed to read the testdata at path %q. Error: %q", testdatapath, err)
		}
		want := string(testdatabytes)

		// Test
		if err := makeConstants(testdir, maptemp); err != nil {
			t.Fatalf("Failed to generate the code for directory %q with maps template Error: %q", testdir, err)
		}
		databytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			t.Fatal("Failed to create the constants.go file (or failed to read it after creation). Error:", err)
		}
		data := string(databytes)
		if data != want {
			t.Fatal("Failed to generate the code properly. Expected:", want, "Actual:", data)
		}
	})

	t.Run("read empty directory and generate code with conststemp", func(t *testing.T) {
		// Setup
		testparentdir := t.TempDir()
		testdir := filepath.Join(testparentdir, "foobar")
		if err := os.Mkdir(testdir, os.ModePerm); err != nil {
			t.Fatalf("Failed to create the test directory at path %q. Error: %q", testdir, err)
		}
		fpath := filepath.Join(testdir, "constants.go")
		testdatapath := "testdata/conststempempty.txt"
		testdatabytes, err := ioutil.ReadFile(testdatapath)
		if err != nil {
			t.Fatalf("Failed to read the testdata at path %q. Error: %q", testdatapath, err)
		}
		want := string(testdatabytes)

		// Test
		if err := makeConstants(testdir, conststemp); err != nil {
			t.Fatalf("Failed to generate the code for directory %q with consts template Error: %q", testdir, err)
		}
		databytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			t.Fatal("Failed to create the constants.go file (or failed to read it after creation). Error:", err)
		}
		data := string(databytes)
		if data != want {
			t.Fatal("Failed to generate the code properly. Expected:", want, "Actual:", data)
		}
	})

	t.Run("read filled directory and generate code with maptemp", func(t *testing.T) {
		// Setup
		testdir := "testdata/datafortempfilled"
		fpath := filepath.Join(testdir, "constants.go")
		// Remove the constants.go file if it exists from previous runs.
		if err := os.Remove(fpath); err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove the file at path %q. Error: %q", fpath, err)
		}
		testdatapath := "testdata/maptempfilled.txt"
		testdatabytes, err := ioutil.ReadFile(testdatapath)
		if err != nil {
			t.Fatalf("Failed to read the testdata at path %q. Error: %q", testdatapath, err)
		}
		want := string(testdatabytes)

		// Test
		if err := makeConstants(testdir, maptemp); err != nil {
			t.Fatalf("Failed to generate the code for directory %q with maps template Error: %q", testdir, err)
		}
		defer os.Remove(fpath)
		databytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			t.Fatal("Failed to create the constants.go file (or failed to read it after creation). Error:", err)
		}
		data := string(databytes)
		if data != want {
			t.Fatal("Failed to generate the code properly. Expected:", want, "Actual:", data)
		}
	})

	t.Run("read filled directory and generate code with conststemp", func(t *testing.T) {
		// Setup
		testdir := "testdata/datafortempfilled"
		fpath := filepath.Join(testdir, "constants.go")
		// Remove the constants.go file if it exists from previous runs.
		if err := os.Remove(fpath); err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove the file at path %q. Error: %q", fpath, err)
		}
		testdatapath := "testdata/conststempfilled.txt"
		testdatabytes, err := ioutil.ReadFile(testdatapath)
		if err != nil {
			t.Fatalf("Failed to read the testdata at path %q. Error: %q", testdatapath, err)
		}
		want := string(testdatabytes)

		// Test
		if err := makeConstants(testdir, conststemp); err != nil {
			t.Fatalf("Failed to generate the code for directory %q with consts template Error: %q", testdir, err)
		}
		defer os.Remove(fpath)
		databytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			t.Fatal("Failed to create the constants.go file (or failed to read it after creation). Error:", err)
		}
		data := string(databytes)
		if data != want {
			t.Fatal("Failed to generate the code properly. Expected:", want, "Actual:", data)
		}
	})

	t.Run("generate code from directory containing files that we have no permissions to read", func(t *testing.T) {
		// Setup
		testdir := t.TempDir()
		fpath := filepath.Join(testdir, "foobar")
		if err := ioutil.WriteFile(fpath, []byte("no permission to read this file"), 0); err != nil {
			t.Fatalf("Failed to create the temporary file %q for testing.", fpath)
		}

		// Test
		if err := makeConstants(testdir, conststemp); err == nil {
			t.Fatalf("Should not have succeeded since the directory contains a file %q we don't have permissions to read.", fpath)
		}
	})

	t.Run("generate code from directory that we have no permissions to write to", func(t *testing.T) {
		// Setup
		tempdir := t.TempDir()
		testdir := filepath.Join(tempdir, "foobar")
		if err := os.Mkdir(testdir, 0400); err != nil {
			t.Fatalf("Failed to create the temporary directory %q for testing. Error: %q", testdir, err)
		}

		// Test
		if err := makeConstants(testdir, conststemp); err == nil {
			t.Fatalf("Should not have succeeded since we don't have permissions to write into the directory %q", testdir)
		}
	})
}

func TestMakeTar(t *testing.T) {
	t.Run("make a tar using a filled directory", func(t *testing.T) {
		// Setup
		testdir := "testdata/datafortempfilled"
		fpath := filepath.Join(testdir, "constants.go")
		// Remove the constants.go file if it exists from previous runs.
		if err := os.Remove(fpath); err != nil && !os.IsNotExist(err) {
			t.Fatalf("Failed to remove the file at path %q. Error: %q", fpath, err)
		}
		testdatapath := "testdata/tartempfilledandtar.txt"
		testdatabytes, err := ioutil.ReadFile(testdatapath)
		if err != nil {
			t.Fatalf("Failed to read the testdata at path %q. Error: %q", testdatapath, err)
		}
		want := string(testdatabytes)

		// Test
		if err := makeTar(testdir); err != nil {
			t.Fatalf("Failed to generate the code for directory %q with tar template Error: %q", testdir, err)
		}
		defer os.Remove(fpath)
		databytes, err := ioutil.ReadFile(fpath)
		if err != nil {
			t.Fatal("Failed to create the constants.go file (or failed to read it after creation). Error:", err)
		}
		data := string(databytes)
		lines := strings.Split(data, "\n")
		lines = lines[:len(lines)-1] // Skip the last line since the tar string has a timestamp
		data = strings.Join(lines, "\n")
		if data != want {
			t.Fatal("Failed to generate the code properly. Expected:", want, "Actual:", data)
		}
	})

	t.Run("make a tar when the directory has files which we have no permissions to read", func(t *testing.T) {
		tempdir := t.TempDir()
		fpath := filepath.Join(tempdir, "nopermstoread")
		if err := ioutil.WriteFile(fpath, []byte("no permission to read this file"), 0); err != nil {
			t.Fatalf("Failed to create the temporary file %q for testing.", fpath)
		}
		if err := makeTar(tempdir); err == nil {
			t.Fatalf("Should not have succeeded since the directory contains a file %q we don't have permissions to read.", fpath)
		}
	})

	t.Run("make a tar in a directory that we have no permissions to write to", func(t *testing.T) {
		// Setup
		tempdir := t.TempDir()
		testdir := filepath.Join(tempdir, "foobar")
		if err := os.Mkdir(testdir, 0400); err != nil {
			t.Fatalf("Failed to create the temporary directory %q for testing. Error: %q", testdir, err)
		}

		// Test
		if err := makeTar(testdir); err == nil {
			t.Fatalf("Should not have succeeded since we don't have permissions to write into the directory %q", testdir)
		}
	})
}
