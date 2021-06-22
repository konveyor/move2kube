/*
Copyright IBM Corporation 2020, 2021

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

package container

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/sirupsen/logrus"
)

func copyDirToContainer(ctx context.Context, cli *client.Client, containerID, src, dst string) error {
	reader := readDirAsTar(src, dst)
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
	return archive.CopyTo(content, copyInfo, destPath)
}

func readDirAsTar(srcDir, basePath string) io.ReadCloser {
	errChan := make(chan error)
	pr, pw := io.Pipe()
	go func() {
		err := writeDirToTar(pw, srcDir, basePath)
		errChan <- err
	}()
	closed := false
	return ioutils.NewReadCloserWrapper(pr, func() error {
		if closed {
			return errors.New("reader already closed")
		}
		perr := pr.Close()
		if err := <-errChan; err != nil {
			closed = true
			if perr == nil {
				return err
			}
			return fmt.Errorf("%s - %s", perr, err)
		}
		closed = true
		return nil
	})
}

func writeDirToTar(w *io.PipeWriter, srcDir, basePath string) error {
	defer w.Close()
	tw := tar.NewWriter(w)
	defer tw.Close()
	return filepath.Walk(srcDir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			logrus.Debugf("Error walking folder to copy to container : %s", err)
			return err
		}
		if fi.Mode()&os.ModeSocket != 0 {
			return nil
		}
		var header *tar.Header
		if fi.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(file)
			if err != nil {
				return err
			}
			// Ensure that symlinks have Linux link names
			header, err = tar.FileInfoHeader(fi, filepath.ToSlash(target))
			if err != nil {
				return err
			}
		} else {
			header, err = tar.FileInfoHeader(fi, fi.Name())
			if err != nil {
				return err
			}
		}
		relPath, err := filepath.Rel(srcDir, file)
		if err != nil {
			logrus.Debugf("Error walking folder to copy to container : %s", err)
			return err
		} else if relPath == "." {
			return nil
		}
		header.Name = filepath.ToSlash(filepath.Join(basePath, relPath))
		if err := tw.WriteHeader(header); err != nil {
			logrus.Debugf("Error walking folder to copy to container : %s", err)
			return err
		}
		if fi.Mode().IsRegular() {
			f, err := os.Open(file)
			if err != nil {
				logrus.Debugf("Error walking folder to copy to container : %s", err)
				return err
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				logrus.Debugf("Error walking folder to copy to container : %s", err)
				return err
			}
		}
		return nil
	})
}

func copyDir(ctx context.Context, cli *client.Client, containerID, src, dst string) error {
	reader := readDirAsTar(src, dst)
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
