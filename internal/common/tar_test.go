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

package common_test

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"
)

// myFileInfo is used to hold the part of the expected outputs for each testcase
// This is only for testing purposes.
type myFileInfo struct {
	Name  string
	Size  int64
	Mode  os.FileMode
	IsDir bool
}

func newMyFileInfo(f os.FileInfo) myFileInfo {
	return myFileInfo{f.Name(), f.Size(), f.Mode(), f.IsDir()}
}

// myUnTarForTesting extracts information from a tar string.
// This function is for testing purposes only.
func myUnTarForTesting(tarstring string, myfinfos []myFileInfo, contents []string) error {
	val, err := base64.StdEncoding.DecodeString(tarstring)
	if err != nil {
		logrus.Errorf("Unable to decode tarstring : %s", err)
	}
	tr := tar.NewReader(bytes.NewReader(val))
	expectedTotal := len(myfinfos)

	for i := range myfinfos {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("The tar string %q contains less files than expected. Expected: %d Actual: %d", tarstring, expectedTotal, i)
		}
		if err != nil {
			return err
		}
		finfo := hdr.FileInfo()
		actualmyfinfo := newMyFileInfo(finfo)
		if actualmyfinfo != myfinfos[i] {
			return fmt.Errorf("The info for file number %d is incorrect. Expected: %v Actual %v", i, myfinfos[i], actualmyfinfo)
		}
		if finfo.Mode().IsDir() {
			continue
		}
		buf := bytes.NewBuffer([]byte{})
		size, err := io.Copy(buf, tr)
		if err != nil {
			return err
		}
		if size != finfo.Size() {
			return fmt.Errorf("Size mismatch: Wrote %d, Expected %d", size, finfo.Size())
		}
		actualContents := buf.String()
		if actualContents != contents[i] {
			return fmt.Errorf("The contents of the file at path %q is incorrect. Expected: %q Actual: %q", actualmyfinfo.Name, contents[i], actualContents)
		}
	}
	if _, err := tr.Next(); err != io.EOF {
		return fmt.Errorf("The tar string %q contains more files than expected. Expected: %d Actual: at least %d", tarstring, expectedTotal, expectedTotal+1)
	}
	return nil
}

func TestTarAsString(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("tar as string when the path doesn't exist", func(t *testing.T) {
		if _, err := common.TarAsString("foobar", []string{}); err == nil {
			t.Fatal("Should not have succeeded since the path doesn't exist.")
		}
	})

	t.Run("tar as string when the path is a file", func(t *testing.T) {
		testdatapath := "testdata/datafortestingtar/tobetarred/test1.yaml"
		tempdir := t.TempDir()
		filepath := filepath.Join(tempdir, "test1.yaml")
		if err := common.CopyFile(filepath, testdatapath); err != nil {
			t.Fatalf("Failed to copy test data from %q to %q for testing. Error %q", testdatapath, filepath, err)
		}
		// want := ""
		expectedInfos := []myFileInfo{
			{".", 217, common.DefaultFilePermission, false},
		}
		data, err := ioutil.ReadFile(filepath)
		if err != nil {
			t.Fatalf("Failed to read the test data at %q. Error %q", filepath, err)
		}
		expectedContents := []string{
			string(data),
		}
		if tarstring, err := common.TarAsString(filepath, []string{}); err != nil {
			t.Fatalf("Failed to tar the file %q. Error: %q", filepath, err)
		} else if err := myUnTarForTesting(tarstring, expectedInfos, expectedContents); err != nil {
			t.Fatal("Failed to tar the file", filepath, "properly. Error:", err)
		}
	})

	t.Run("tar as string when the path is an empty directory", func(t *testing.T) {
		parent := t.TempDir()
		path1 := filepath.Join(parent, "foobar")
		if err := os.Mkdir(path1, common.DefaultDirectoryPermission); err != nil {
			t.Fatal("Failed to make the directory", path1, "Error:", err)
		}
		// want := ""
		expectedInfos := []myFileInfo{
			{".", 0, os.ModeDir | common.DefaultDirectoryPermission, true},
		}
		expectedContents := []string{}
		if tarstring, err := common.TarAsString(path1, []string{}); err != nil {
			t.Fatalf("Failed to tar the file %q. Error: %q", path1, err)
		} else if err := myUnTarForTesting(tarstring, expectedInfos, expectedContents); err != nil {
			t.Fatal("Failed to tar the file", path1, "properly. Error:", err)
		}
	})

	t.Run("tar as string when the path is a filled directory", func(t *testing.T) {
		// Setup
		testdirpath := "testdata/datafortestingtar/tobetarred"
		// want := ""
		expectedInfos := []myFileInfo{}
		expectedContents := []string{}
		err := filepath.Walk(testdirpath, func(path string, finfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			expectedInfo := newMyFileInfo(finfo)
			if expectedInfo.Name, err = filepath.Rel(testdirpath, path); err != nil {
				return err
			}
			if finfo.IsDir() {
				expectedInfo.Size = 0
				expectedInfos = append(expectedInfos, expectedInfo)
				expectedContents = append(expectedContents, "")
				return nil
			}
			expectedInfos = append(expectedInfos, expectedInfo)
			testfilepath := filepath.Join(testdirpath, finfo.Name())
			testfilecontents, err := ioutil.ReadFile(testfilepath)
			if err != nil {
				t.Fatalf("Failed to read the testdata at path: %q Error %q", testfilepath, err)
			}
			expectedContents = append(expectedContents, string(testfilecontents))
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to read the testdata at path: %q", testdirpath)
		}

		// Test
		if tarstring, err := common.TarAsString(testdirpath, []string{}); err != nil {
			t.Fatalf("Failed to tar the file %q. Error: %q", testdirpath, err)
		} else if err := myUnTarForTesting(tarstring, expectedInfos, expectedContents); err != nil {
			t.Fatal("Failed to tar the file", testdirpath, "properly. Error:", err)
		}
	})

	t.Run("tar as string while ignoring some files", func(t *testing.T) {
		// Setup
		testdirpath := "testdata/datafortestingtar/tobetarred"
		ignoredFiles := []string{"test2.yml", "versioninfo.json", "foobar.json"}
		// want := ""
		expectedInfos := []myFileInfo{}
		expectedContents := []string{}
		err := filepath.Walk(testdirpath, func(path string, finfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			expectedInfo := newMyFileInfo(finfo)
			if expectedInfo.Name, err = filepath.Rel(testdirpath, path); err != nil {
				return err
			}
			if common.IsStringPresent(ignoredFiles, expectedInfo.Name) {
				return nil
			}
			if finfo.IsDir() {
				expectedInfo.Size = 0
				expectedInfos = append(expectedInfos, expectedInfo)
				expectedContents = append(expectedContents, "")
				return nil
			}
			expectedInfos = append(expectedInfos, expectedInfo)
			testfilepath := filepath.Join(testdirpath, finfo.Name())
			testfilecontents, err := ioutil.ReadFile(testfilepath)
			if err != nil {
				t.Fatalf("Failed to read the testdata at path: %q Error %q", testfilepath, err)
			}
			expectedContents = append(expectedContents, string(testfilecontents))
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to read the testdata at path: %q", testdirpath)
		}

		// Test
		if tarstring, err := common.TarAsString(testdirpath, ignoredFiles); err != nil {
			t.Fatalf("Failed to tar the file %q. Error: %q", testdirpath, err)
		} else if err := myUnTarForTesting(tarstring, expectedInfos, expectedContents); err != nil {
			t.Fatal("Failed to tar the file", testdirpath, "properly. Error:", err)
		}
	})

	t.Run("tar as string when the directory has files which you don't have permission to read", func(t *testing.T) {
		path1 := t.TempDir()
		path2 := filepath.Join(path1, "nopermstoread")
		if err := ioutil.WriteFile(path2, []byte("no permission to read this file"), 0); err != nil {
			t.Fatalf("Failed to create the temporary file %q for testing.", path2)
		}
		if _, err := common.TarAsString(path2, []string{}); err == nil {
			t.Fatalf("Should not have succeeded since the directory contains a file %q we don't have permissions to read.", path2)
		}
	})
}

func TestUnTarString(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	testdatadir := "testdata/datafortestingtar/untarstring.json"
	testdata := map[string]string{}
	if testdatabytes, err := ioutil.ReadFile(testdatadir); err != nil {
		t.Fatal("Failed to read the testdata at path:", testdatadir, "Error:", err)
	} else if err := json.Unmarshal(testdatabytes, &testdata); err != nil {
		t.Fatal("Failed to unmarshal the json testdata at path:", testdatadir, "Error:", err)
	}

	t.Run("untar a valid tarstring", func(t *testing.T) {
		path := t.TempDir()
		tarstring := testdata["untar_a_valid_tarstring"]
		if err := common.UnTarString(tarstring, path); err != nil {
			t.Fatalf("Failed to untar the tarstring %q to the path %q. Error: %q", tarstring, path, err)
		}
		if _, err := common.TarAsString(path, []string{}); err != nil {
			t.Fatalf("Failed to tar as string the directory %q we just untarred. Error: %q", path, err)
		}
	})

	t.Run("untar an invalid base 64 string", func(t *testing.T) {
		path := t.TempDir()
		tarstring := testdata["untar_an_invalid_base_64_string"]
		if err := common.UnTarString(tarstring, path); err == nil {
			t.Fatalf("Should have given an error since the tarstring %q is not a valid base 64 string.", tarstring)
		}
	})

	t.Run("untar an invalid tarstring", func(t *testing.T) {
		path := t.TempDir()
		tarstring := testdata["untar_an_invalid_tarstring"]
		if err := common.UnTarString(tarstring, path); err == nil {
			t.Fatalf("Should have given an error since the tarstring %q is not valid.", tarstring)
		}
	})

	t.Run("untar into a directory we don't have permission to write to", func(t *testing.T) {
		parent := t.TempDir()
		noPermDir := filepath.Join(parent, "nopermstowrite")
		if err := os.Mkdir(noPermDir, 0); err != nil {
			t.Fatalf("Failed to create the temporary directory %q for testing. Error: %q", noPermDir, err)
		}
		path := filepath.Join(noPermDir, "foobar")
		tarstring := testdata["untar_into_a_directory_we_dont_have_permission_to_write_to"]
		if err := common.UnTarString(tarstring, path); err == nil {
			t.Fatalf("Should have given an error since we don't have permission to write to the directory %q", path)
		}
	})

	t.Run("untar a single file into a directory we don't have permission to write to", func(t *testing.T) {
		parent := t.TempDir()
		noPermDir := filepath.Join(parent, "nopermstowrite")
		if err := os.Mkdir(noPermDir, 0); err != nil {
			t.Fatalf("Failed to create the temporary directory %q for testing. Error: %q", noPermDir, err)
		}
		path := filepath.Join(noPermDir, "foobar")
		tarstring := testdata["untar_a_single_file_into_a_directory_we_dont_have_permission_to_write_to"]
		if err := common.UnTarString(tarstring, path); err == nil {
			t.Fatalf("Should have given an error since we don't have permission to write to the directory %q", path)
		}
	})
}
