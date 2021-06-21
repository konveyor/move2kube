/*
Copyright IBM Corporation 2021

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

package filesystem

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func Merge(source, destination string) error {
	options := options{
		processFileCallBack: mergeProcessFileCallBack,
		additionCallBack:    mergeAdditionCallBack,
		deletionCallBack:    mergeDeletionCallBack,
		mismatchCallBack:    mergeDeletionCallBack,
	}
	return newProcessor(options).process(source, destination)
}

func mergeProcessFileCallBack(sourceFilePath, destinationFilePath string, config interface{}) error {
	si, err := os.Stat(sourceFilePath)
	if err != nil {
		logrus.Errorf("Unable to stat file %s : %s", sourceFilePath, err)
		return err
	}
	di, err := os.Stat(destinationFilePath)
	if err == nil {
		if !(si.Mode().IsRegular() != di.Mode().IsRegular() || si.Size() != di.Size() || si.ModTime() != di.ModTime()) {
			return nil
		} else {
			logrus.Warnf("Overwriting file : %s with %s", destinationFilePath, sourceFilePath)
		}
	}
	sourceReader, err := os.Open(sourceFilePath)
	if err != nil {
		logrus.Errorf("Unable to open file %s : %s", sourceFilePath, err)
		return err
	}
	defer sourceReader.Close()
	destinationWriter, err := os.Create(destinationFilePath)
	if err != nil {
		sdi, err := os.Stat(filepath.Dir(sourceFilePath))
		if err != nil {
			logrus.Errorf("Unable to stat parent dir of %s : %s", sourceFilePath, err)
			return err
		}
		if mderr := os.MkdirAll(filepath.Dir(destinationFilePath), sdi.Mode()); mderr == nil {
			destinationWriter, err = os.Create(destinationFilePath)
		}
		if err != nil {
			logrus.Errorf("Unable to create destination file %s : %s", destinationFilePath, err)
			return err
		}
	}
	defer destinationWriter.Close()
	_, err = io.Copy(destinationWriter, sourceReader)
	if err != nil {
		logrus.Errorf("Unable to copy file %s to %s : %s", sourceFilePath, destinationFilePath, err)
		return err
	}
	err = destinationWriter.Sync()
	if err != nil {
		logrus.Errorf("Unable to sync file %s to %s : %s", sourceFilePath, destinationFilePath, err)
		return err
	}
	err = os.Chmod(destinationFilePath, si.Mode())
	if err != nil {
		logrus.Errorf("Unable to copy permissions in file %s : %s", destinationFilePath, err)
		return err
	}
	return nil
}

func mergeAdditionCallBack(source, destination string, config interface{}) error {
	return nil
}

func mergeDeletionCallBack(source, destination string, config interface{}) error {
	si, err := os.Stat(source)
	if err != nil {
		logrus.Errorf("Unable to stat %s : %s", source, err)
		return err
	}
	err = os.MkdirAll(destination, si.Mode())
	if err != nil {
		logrus.Errorf("Unable to create directory %s", destination)
		return err
	}
	err = os.Chmod(destination, si.Mode())
	if err != nil {
		logrus.Errorf("Unable to copy permissions in file %s : %s", destination, err)
		return err
	}
	return nil
}
