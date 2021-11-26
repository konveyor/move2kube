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

	"github.com/sirupsen/logrus"
)

// Replicate replicates the source directory into destination
func Replicate(source, destination string) error {
	options := options{
		processFileCallBack: replicateProcessFileCallBack,
		additionCallBack:    replicateAdditionCallBack,
		deletionCallBack:    replicateDeletionCallBack,
		mismatchCallBack:    replicateDeletionCallBack,
	}
	return newProcessor(options).process(source, destination)
}

func replicateProcessFileCallBack(sourceFilePath, destinationFilePath string, config interface{}) error {
	si, err := os.Stat(sourceFilePath)
	if err != nil {
		logrus.Errorf("Unable to stat file %s : %s", sourceFilePath, err)
		return err
	}
	di, err := os.Stat(destinationFilePath)
	if err == nil {
		if !(si.Mode().IsRegular() != di.Mode().IsRegular() || si.Size() != di.Size() || si.ModTime() != di.ModTime()) {
			return nil
		}
	}
	return copyFile(destinationFilePath, sourceFilePath, si.ModTime())
}

func replicateAdditionCallBack(source, destination string, config interface{}) error {
	return os.RemoveAll(destination)
}

func replicateDeletionCallBack(source, destination string, config interface{}) error {
	si, err := os.Stat(source)
	if err != nil {
		logrus.Errorf("Unable to stat %s : %s", source, err)
		return err
	}
	os.RemoveAll(destination)
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
