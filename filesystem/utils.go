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
	"time"

	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

// Copies file and sets mod time
func copyFile(sf, df string, modTime time.Time) error {
	err := common.CopyFile(sf, df)
	if err != nil {
		logrus.Errorf("Unable to copy file %s to %s : %s", sf, df, err)
		return err
	}
	err = os.Chtimes(df, modTime, modTime)
	if err != nil {
		logrus.Errorf("Unable to change timestamp for file %s : %s", df, err)
		return err
	}
	return nil
}
