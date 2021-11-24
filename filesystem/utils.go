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

func copyFile(sf, df string) error {
	si, err := os.Stat(sf)
	if err != nil {
		logrus.Errorf("Unable to stat file %s : %s", sf, err)
		return err
	}
	sdi, err := os.Stat(filepath.Dir(sf))
	if err != nil {
		logrus.Errorf("Unable to stat parent dir of %s : %s", sf, err)
		return err
	}
	err = os.MkdirAll(filepath.Dir(df), sdi.Mode())
	if err != nil {
		logrus.Errorf("Unable to make dir %s : %s", filepath.Dir(df), err)
		return err
	}
	sbytes, err := os.ReadFile(sf)
	if err != nil {
		logrus.Errorf("Unable to read file %s : %s", sf, err)
		return err
	}
	err = os.WriteFile(df, sbytes, si.Mode())
	if err != nil {
		logrus.Errorf("Unable to write file %s : %s", df, err)
		return err
	}
	err = os.Chtimes(df, si.ModTime(), si.ModTime())
	if err != nil {
		logrus.Errorf("Unable to change timestamp for file %s : %s", df, err)
		return err
	}
	return nil
}
