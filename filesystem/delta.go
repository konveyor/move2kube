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
	"path/filepath"

	"github.com/sirupsen/logrus"
)

const (
	additionsDir     = "additions"
	deletionsFile    = "deletions.txt"
	modificationsDir = "modifications"
)

type generateDeltaConfig struct {
	store                string
	sourceDirectory      string
	destinationDirectory string
}

// GenerateDelta generates Delta between source and destination
func GenerateDelta(source, destination, store string) error {
	options := options{
		processFileCallBack: generateDeltaProcessFileCallBack,
		additionCallBack:    generateDeltaAdditionCallBack,
		deletionCallBack:    generateDeltaDeletionCallBack,
		mismatchCallBack:    generateDeltaAdditionCallBack,
		config: generateDeltaConfig{
			store:                store,
			sourceDirectory:      source,
			destinationDirectory: destination,
		},
	}
	return newProcessor(options).process(source, destination)
}

func generateDeltaProcessFileCallBack(sourceFilePath, destinationFilePath string, config interface{}) error {
	gconfig := config.(generateDeltaConfig)
	destRel, err := filepath.Rel(gconfig.destinationDirectory, destinationFilePath)
	if err != nil {
		logrus.Errorf("Unable to resolve destination dir %s as rel path : %s", destinationFilePath, err)
	}
	modifiedFilePath := filepath.Join(gconfig.store, modificationsDir, destRel)
	si, err := os.Stat(sourceFilePath)
	if err != nil {
		logrus.Errorf("Unable to stat file %s : %s", sourceFilePath, err)
		return err
	}
	di, err := os.Stat(destinationFilePath)
	if err == nil {
		if err == nil && !(si.Mode().IsRegular() != di.Mode().IsRegular() || si.Size() != di.Size() || si.ModTime() != di.ModTime()) {
			return nil
		} else if err != nil {
			logrus.Errorf("Unable to compare files to check if files are same %s and %s. Marking as modification: %s", sourceFilePath, destinationFilePath, err)
		}
	}
	return copyFile(modifiedFilePath, sourceFilePath, si.ModTime())
}

func generateDeltaAdditionCallBack(source, destination string, config interface{}) error {
	gconfig := config.(generateDeltaConfig)
	destRel, err := filepath.Rel(gconfig.destinationDirectory, destination)
	if err != nil {
		logrus.Errorf("Unable to resolve destination dir %s as rel path : %s", destination, err)
	}
	return Replicate(source, filepath.Join(gconfig.store, additionsDir, destRel))
}

func generateDeltaDeletionCallBack(source, destination string, config interface{}) error {
	gconfig := config.(generateDeltaConfig)
	f, err := os.OpenFile(filepath.Join(gconfig.store, deletionsFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logrus.Errorf("Unable to open modifications file to record deleteions : %s", gconfig.store)
		return err
	}
	defer f.Close()
	destRel, err := filepath.Rel(gconfig.destinationDirectory, destination)
	if err != nil {
		logrus.Errorf("Unable to resolve destination dir %s as rel path : %s", destination, err)
	}
	if _, err := f.WriteString(destRel + "\n"); err != nil {
		logrus.Errorf("Unable to record deletions to file : %s", gconfig.store)
		return err
	}
	return nil
}
