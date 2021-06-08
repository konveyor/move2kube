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

package common

import (
	"archive/tar"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// TarAsString converts a directory into a string
func TarAsString(path string, ignorefiles []string) (string, error) {

	buf := bytes.NewBuffer([]byte{})
	tw := tar.NewWriter(buf)
	defer tw.Close()

	err := filepath.Walk(path, func(currpath string, finfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(finfo, finfo.Name())
		if err != nil {
			return err
		}
		if hdr.Name, err = filepath.Rel(path, currpath); err != nil {
			return err
		}
		for _, ignorefile := range ignorefiles {
			if hdr.Name == ignorefile {
				return nil
			}
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if finfo.Mode().IsDir() {
			return nil
		}
		currfile, err := os.Open(currpath)
		if err != nil {
			return err
		}
		defer currfile.Close()
		_, err = io.Copy(tw, currfile)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logrus.Warnf("Failed to create tar string: %s : %s", path, err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), err
}

// UnTarString converts a string into a directory
func UnTarString(tarstring string, path string) (err error) {
	val, err := base64.StdEncoding.DecodeString(tarstring)
	if err != nil {
		logrus.Errorf("Unable to decode tarstring : %s", err)
		return err
	}
	tr := tar.NewReader(bytes.NewReader(val))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		finfo := hdr.FileInfo()
		fileName := hdr.Name
		filepath := filepath.Join(path, fileName)
		if finfo.Mode().IsDir() {
			if err := os.MkdirAll(filepath, DefaultDirectoryPermission); err != nil {
				return err
			}
			continue
		}
		file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, finfo.Mode().Perm())
		if err != nil {
			return err
		}
		defer file.Close()
		size, err := io.Copy(file, tr)
		if err != nil {
			return err
		}
		if size != finfo.Size() {
			return fmt.Errorf("size mismatch: Wrote %d, Expected %d", size, finfo.Size())
		}
	}
	return nil
}
