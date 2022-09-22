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
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

func copyDirToContainer(ctx context.Context, cli *client.Client, containerID, src, dst string) error {
	reader := common.ReadFilesAsTar(src, dst, common.NoCompression)
	if reader == nil {
		err := fmt.Errorf("error during create tar archive from '%s'", src)
		logrus.Error(err)
		return err
	}
	defer reader.Close()
	var clientErr, err error
	doneChan := make(chan interface{})
	pr, pw := io.Pipe()
	go func() {
		clientErr = cli.CopyToContainer(ctx, containerID, "/", pr, types.CopyToContainerOptions{})
		close(doneChan)
	}()
	func() {
		defer pw.Close()
		var nBytesCopied int64
		nBytesCopied, err = io.Copy(pw, reader)
		logrus.Debugf("%d bytes copied into pipe as tar", nBytesCopied)
	}()
	<-doneChan
	if err == nil {
		err = clientErr
	}
	return err
}

func copyFromContainer(ctx context.Context, cli *client.Client, containerID string, containerPath, destPath string) (err error) {
	content, stat, err := cli.CopyFromContainer(ctx, containerID, containerPath)
	if err != nil {
		logrus.Errorf("Unable to copy from container : %s", err)
		return err
	}
	defer content.Close()
	copyInfo := archive.CopyInfo{
		Path:   containerPath,
		Exists: true,
		IsDir:  stat.Mode.IsDir(),
	}
	preArchive := content
	_, srcBase := archive.SplitPathDirEntry(copyInfo.Path)
	preArchive = archive.RebaseArchiveEntries(content, srcBase, "")
	return archive.CopyTo(preArchive, copyInfo, destPath)
}

func copyDir(ctx context.Context, cli *client.Client, containerID, src, dst string) error {
	reader := common.ReadFilesAsTar(src, dst, common.NoCompression)
	if reader == nil {
		err := fmt.Errorf("error during create tar archive from '%s'", src)
		logrus.Error(err)
		return err
	}
	defer reader.Close()
	var clientErr, err error
	doneChan := make(chan interface{})
	pr, pw := io.Pipe()
	go func() {
		clientErr = cli.CopyToContainer(ctx, containerID, "/", pr, types.CopyToContainerOptions{})
		close(doneChan)
	}()
	func() {
		defer pw.Close()
		var nBytesCopied int64
		nBytesCopied, err = io.Copy(pw, reader)
		logrus.Debugf("%d bytes copied into pipe as tar", nBytesCopied)
	}()
	<-doneChan
	if err == nil {
		err = clientErr
	}
	return err
}
