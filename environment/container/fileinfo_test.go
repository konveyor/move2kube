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
package container

import (
	"testing"
	"time"

	"github.com/docker/docker/api/types"
)

func mockFunction() types.ContainerPathStat {
	return types.ContainerPathStat{
		Name:  "mockfile.txt",
		Size:  int64(512),
		Mode:  0755,
		Mtime: time.Now(),
	}
}

func TestFileInfoName(t *testing.T) {
	t.Run("Testing Name method", func(t *testing.T) {
		// Use the mock function to get a types.ContainerPathStat object
		realStat := mockFunction()

		// Use the real stat to create a FileInfo instance
		fileInfo := &FileInfo{
			stat: realStat,
		}

		// Call the Name method and check if it returns the expected name
		actualName := fileInfo.Name()
		if actualName != realStat.Name {
			t.Errorf("Name() = %s, expected %s", actualName, realStat.Name)
		}
	})
}

func TestFileInfoSize(t *testing.T) {
	t.Run("Testing Size method", func(t *testing.T) {

		realStat := mockFunction()

		fileInfo := &FileInfo{
			stat: realStat,
		}

		// Call the Size method and check if it returns the expected size
		actualSize := fileInfo.Size()
		if actualSize != realStat.Size {
			t.Errorf("Size() = %d, expected %d", actualSize, realStat.Size)
		}
	})
}

func TestFileInfoMode(t *testing.T) {
	t.Run("Testing Mode method", func(t *testing.T) {

		realStat := mockFunction()

		fileInfo := &FileInfo{
			stat: realStat,
		}

		// Call the Mode method and check if it returns the expected mode
		actualMode := fileInfo.Mode()
		if actualMode != realStat.Mode {
			t.Errorf("Mode() = %v, expected %v", actualMode, realStat.Mode)
		}
	})
}

func TestFileInfoModTime(t *testing.T) {
	t.Run("Testing ModTime method", func(t *testing.T) {

		realStat := mockFunction()

		fileInfo := &FileInfo{
			stat: realStat,
		}

		// Call the ModTime method and check if it returns the expected mod time
		actualModTime := fileInfo.ModTime()
		if !actualModTime.Equal(realStat.Mtime) {
			t.Errorf("ModTime() = %v, expected %v", actualModTime, realStat.Mtime)
		}
	})
}

func TestFileInfoIsDir(t *testing.T) {
	t.Run("Testing IsDir method", func(t *testing.T) {

		realStat := mockFunction()

		fileInfo := &FileInfo{
			stat: realStat,
		}

		// Call the IsDir method and check if it returns the expected value
		actualIsDir := fileInfo.IsDir()
		if actualIsDir != realStat.Mode.IsDir() {
			t.Errorf("IsDir() = %v, expected %v", actualIsDir, realStat.Mode.IsDir())
		}
	})
}
