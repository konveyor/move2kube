/*
Copyright IBM Corporation 2020

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

package containerexec

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

type dockerEngine struct {
	availableImages map[string]bool
}

func newDockerEngine() *dockerEngine {
	return &dockerEngine{
		availableImages: map[string]bool{},
	}
}

func (e *dockerEngine) pullImage(image string) bool {
	if a, ok := e.availableImages[image]; ok {
		return a
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logrus.Debugf("Unable to pull image %s : %s", image, err)
		e.availableImages[image] = false
		return false
	}
	out, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		logrus.Debugf("Unable to pull image %s : %s", image, err)
		e.availableImages[image] = false
		return false
	}
	if b, err := ioutil.ReadAll(out); err == nil {
		logrus.Debug(cast.ToString(b))
	}
	e.availableImages[image] = true
	return true
}

// RunContainer executes a container
func (e *dockerEngine) RunContainer(image string, cmd string, volsrc string, voldest string) (output string, containerStarted bool, err error) {
	if !e.pullImage(image) {
		logrus.Debugf("Unable to pull image using docker : %s", image)
		return "", false, fmt.Errorf("unable to pull image")
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logrus.Debugf("Error during docker client creation : %s", err)
		return "", false, err
	}
	contconfig := &container.Config{
		Image: image,
	}
	if cmd != "" {
		contconfig.Cmd = []string{cmd}
	}
	if (volsrc == "" && voldest != "") || (volsrc != "" && voldest == "") {
		logrus.Warnf("Either volume source (%s) or destination (%s) is empty. Ingoring volume mount.", volsrc, voldest)
	}
	hostconfig := &container.HostConfig{}
	if volsrc != "" && voldest != "" {
		hostconfig.Mounts = []mount.Mount{
			{
				Type:     mount.TypeBind,
				Source:   volsrc,
				Target:   voldest,
				ReadOnly: true,
			},
		}
	}
	resp, err := cli.ContainerCreate(ctx, contconfig, hostconfig, nil, "")
	if err != nil {
		logrus.Debugf("Error during container creation : %s", err)
		resp, err = cli.ContainerCreate(ctx, contconfig, nil, nil, "")
		if err != nil {
			logrus.Debugf("Container creation failed with image %s with no volumes", image)
			return "", false, err
		}
		logrus.Debugf("Container %s created with image %s with no volumes", resp.ID, image)
		defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		if volsrc != "" && voldest != "" {
			err = copyDir(ctx, cli, resp.ID, volsrc, voldest)
			if err != nil {
				logrus.Debugf("Container data copy failed for image %s with volume %s:%s : %s", image, volsrc, voldest, err)
				return "", false, err
			}
			logrus.Debugf("Data copied from %s to %s in container %s with image %s", volsrc, voldest, resp.ID, image)
		}
	}
	logrus.Debugf("Container %s created with image %s", resp.ID, image)
	defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Debugf("Error during container startup of container %s : %s", resp.ID, err)
		return "", false, err
	}
	statusCh, errCh := cli.ContainerWait(
		ctx,
		resp.ID,
		container.WaitConditionNotRunning,
	)
	select {
	case err := <-errCh:
		if err != nil {
			logrus.Debugf("Error during waiting for container : %s", err)
			return "", false, err
		}
	case status := <-statusCh:
		logrus.Debugf("Container exited with status code: %#+v", status.StatusCode)
		options := types.ContainerLogsOptions{ShowStdout: true}
		out, err := cli.ContainerLogs(ctx, resp.ID, options)
		if err != nil {
			logrus.Debugf("Error while getting container logs : %s", err)
			return "", true, err
		}
		logs := ""
		if b, err := ioutil.ReadAll(out); err == nil {
			logs = cast.ToString(b)
		}
		if status.StatusCode != 0 {
			return logs, true, fmt.Errorf("container execution terminated with error code : %d", status.StatusCode)
		}
		return logs, true, nil
	}
	return "", false, err
}

// InspectImage returns inspect output for an image
func (e *dockerEngine) InspectImage(image string) (types.ImageInspect, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return types.ImageInspect{}, err
	}
	inspectOutput, _, err := cli.ImageInspectWithRaw(ctx, image)
	if err != nil {
		return types.ImageInspect{}, err
	}

	return inspectOutput, nil
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
