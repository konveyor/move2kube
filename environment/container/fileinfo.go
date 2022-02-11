/*
 *  Copyright IBM Corporation 2020, 2021
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
	"io/fs"
	"time"

	"github.com/docker/docker/api/types"
)

// FileInfo implements fs.FileInfo interface
type FileInfo struct {
	stat types.ContainerPathStat
}

// Name implements fs.FileInfo interface
func (f *FileInfo) Name() string { // base name of the file
	return f.stat.Name
}

// Size implements fs.FileInfo interface
func (f *FileInfo) Size() int64 { // length in bytes for regular files; system-dependent for others
	return f.stat.Size
}

// Mode implements fs.FileInfo interface
func (f *FileInfo) Mode() fs.FileMode { // file mode bits
	return f.stat.Mode
}

// ModTime implements fs.FileInfo interface
func (f *FileInfo) ModTime() time.Time { // modification time
	return f.stat.Mtime
}

// IsDir implements fs.FileInfo interface
func (f *FileInfo) IsDir() bool { // abbreviation for Mode().IsDir()
	return f.stat.Mode.IsDir()
}

// Sys implements fs.FileInfo interface
func (f *FileInfo) Sys() interface{} { // underlying data source (can return nil)
	return nil
}
