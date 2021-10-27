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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

type processor struct {
	options options
}

type options struct {
	processFileCallBack func(sourcecFilePath, destinationFilePath string, config interface{}) (err error)
	additionCallBack    func(sourcePath, destinationPath string, config interface{}) (err error)
	deletionCallBack    func(sourcePath, destinationPath string, config interface{}) (err error)
	mismatchCallBack    func(sourcePath, destinationPath string, config interface{}) (err error)
	config              interface{}
}

func newProcessor(options options) *processor {
	return &processor{
		options: options,
	}
}

func (p *processor) process(source, destination string) error {
	si, err := os.Stat(source)
	if err != nil {
		logrus.Errorf("Unable to stat %s : %s", source, err)
		return err
	}
	switch si.Mode() & os.ModeType {
	case os.ModeDir:
		if err := p.processDirectory(source, destination); err != nil {
			return err
		}
	case os.ModeSymlink:
		if err := p.processSymLink(source, destination); err != nil {
			return err
		}
	default:
		di, err := os.Stat(destination)
		if err == nil {
			if di.IsDir() {
				destination = filepath.Join(destination, filepath.Base(source))
			}
		}

		if err := p.processFile(source, destination); err != nil {
			return err
		}
	}
	return nil
}

func (p *processor) processFile(source, destination string) error {
	if p.options.processFileCallBack == nil {
		err := fmt.Errorf("no function found to process file")
		logrus.Errorf("%s", err)
		return err
	}
	err := p.options.processFileCallBack(source, destination, p.options.config)
	if err != nil {
		logrus.Errorf("Unable to process file using custom function at %s. Copying normally : %s", source, err)
		return err
	}
	return nil
}

func (p *processor) processDirectory(source, destination string) error {
	destEntryNames := map[string]bool{}
	entries, err := ioutil.ReadDir(source)
	if err != nil {
		return err
	}

	di, err := os.Stat(destination)
	if err != nil {
		if err := p.options.deletionCallBack(source, destination, p.options.config); err != nil {
			logrus.Errorf("Error during deletion callback for %s, %s", source, destination)
		}
	} else if !di.IsDir() {
		if err := p.options.mismatchCallBack(source, destination, p.options.config); err != nil {
			logrus.Errorf("Error during mismatch callback for %s, %s", source, destination)
		}
	} else {
		destEntries, err := ioutil.ReadDir(destination)
		if err != nil {
			logrus.Errorf("Unable to process directory %s : %s", destination, err)
		} else {
			for _, de := range destEntries {
				destEntryNames[de.Name()] = true
			}
		}
	}
	for _, entry := range entries {
		eN := entry.Name()
		sourcePath := filepath.Join(source, eN)
		destPath := filepath.Join(destination, eN)
		delete(destEntryNames, eN)
		if err := p.process(sourcePath, destPath); err != nil {
			logrus.Errorf("Error during processing : %s", err)
		}
	}
	for deN := range destEntryNames {
		if p.options.additionCallBack != nil {
			err := p.options.additionCallBack(filepath.Join(source, deN), filepath.Join(destination, deN), p.options.config)
			if err != nil {
				logrus.Errorf("Error during addition callback for %s", destination)
			}
		}
	}
	return nil
}

func (p *processor) processSymLink(source, destination string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, destination)
}
