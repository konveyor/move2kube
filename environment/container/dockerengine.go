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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"

	// "github.com/docker/docker/pkg/stdcopy"
	"github.com/konveyor/move2kube/common"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

const (
	testimage = "quay.io/konveyor/hello-world"
)

type dockerEngine struct {
	availableImages map[string]bool
	cli             *client.Client
	ctx             context.Context
}

// newDockerEngine creates a new docker engine instance
func newDockerEngine() (*dockerEngine, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create the docker client. Error: %w", err)
	}
	engine := &dockerEngine{
		cli: cli,
		ctx: ctx,
	}
	if err := engine.updateAvailableImages(); err != nil {
		return engine, fmt.Errorf("failed to update the list of available images. Error: %w", err)
	}
	if _, _, err := engine.RunContainer(testimage, environmenttypes.Command{}, "", ""); err != nil {
		return engine, fmt.Errorf("failed to run the test image '%s' as a container. Error: %w", testimage, err)
	}
	return engine, nil
}

// updateAvailableImages updates the list of available images using the local cache (docker images)
func (e *dockerEngine) updateAvailableImages() error {
	images, err := e.cli.ImageList(e.ctx, types.ImageListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list the images. Error: %w", err)
	}
	e.availableImages = map[string]bool{}
	for _, image := range images {
		for _, repoTag := range image.RepoTags {
			e.availableImages[repoTag] = true
		}
	}
	return nil
}

func (e *dockerEngine) pullImage(image string) error {
	if _, ok := e.availableImages[image]; ok {
		return nil
	}
	logrus.Infof("Pulling container image %s. This could take a few mins.", image)
	out, err := e.cli.ImagePull(e.ctx, image, types.ImagePullOptions{})
	if err != nil {
		e.availableImages[image] = false
		return fmt.Errorf("failed to pull the image '%s' using the docker client. Error: %q", image, err)
	}
	if b, err := io.ReadAll(out); err == nil {
		logrus.Debug(cast.ToString(b))
	}
	e.availableImages[image] = true
	return nil
}

// RunCmdInContainer executes a container
func (e *dockerEngine) RunCmdInContainer(containerID string, cmd environmenttypes.Command, workingdir string, env []string) (stdout, stderr string, exitCode int, err error) {
	execConfig := types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		WorkingDir:   workingdir,
		Env:          env,
	}
	cresp, err := e.cli.ContainerExecCreate(e.ctx, containerID, execConfig)
	if err != nil {
		return
	}
	aresp, err := e.cli.ContainerExecAttach(e.ctx, cresp.ID, types.ExecStartCheck{})
	if err != nil {
		return
	}
	defer aresp.Close()

	var outBuf, errBuf bytes.Buffer
	outputDone := make(chan error)

	// log the container output so we can see what's happening for long running tasks
	ff, err := bufio.NewReader(aresp.Reader).ReadString('\n')
	buf := &bytes.Buffer{}
	for err == nil {
		buf.Write([]byte(ff))
		logrus.Debugf("msg from cmd running in the container is: %s", ff)
		ff, err = bufio.NewReader(aresp.Reader).ReadString('\n')
	}
	if err == io.EOF && len(ff) > 0 {
		buf.Write([]byte(ff))
	}

	go func() {
		_, err = stdcopy.StdCopy(&outBuf, &errBuf, buf)
		outputDone <- err
	}()

	select {
	case err = <-outputDone:
		if err != nil {
			return
		}
		break

	case <-e.ctx.Done():
		return "", "", 0, e.ctx.Err()
	}

	stdoutbytes := outBuf.Bytes()
	stderrbytes := errBuf.Bytes()
	res, err := e.cli.ContainerExecInspect(e.ctx, cresp.ID)
	if err != nil {
		return
	}
	exitCode = res.ExitCode
	stdout = string(stdoutbytes)
	stderr = string(stderrbytes)
	return
}

// InspectImage returns inspect output for an image
func (e *dockerEngine) InspectImage(image string) (types.ImageInspect, error) {
	inspectOutput, _, err := e.cli.ImageInspectWithRaw(e.ctx, image)
	if err != nil {
		return types.ImageInspect{}, err
	}

	return inspectOutput, nil
}

// CreateContainer creates a container
func (e *dockerEngine) CreateContainer(container environmenttypes.Container) (containerid string, err error) {
	if err := e.pullImage(container.Image); err != nil {
		return "", fmt.Errorf("failed to pull the image '%s'. Error: %w", container.Image, err)
	}
	contconfig := &containertypes.Config{
		Image: container.Image,
	}
	if len(container.KeepAliveCommand) > 0 {
		contconfig.Cmd = container.KeepAliveCommand
	}
	resp, err := e.cli.ContainerCreate(e.ctx, contconfig, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create the container with the image '%s' and no volumes attached. Error: %w", container.Image, err)
	}
	if err := e.cli.ContainerStart(e.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start the container with the ID '%s', image '%s' and no volumes attached. Error: %w", resp.ID, container.Image, err)
	}
	logrus.Debugf("Container with ID '%s' created with the image '%s'", resp.ID, container.Image)
	return resp.ID, nil
}

// CreateContainer creates a container
func (e *dockerEngine) StopAndRemoveContainer(containerID string) error {
	if err := e.cli.ContainerRemove(e.ctx, containerID, types.ContainerRemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("failed to remove the container with ID '%s' . Error: %w", containerID, err)
	}
	return nil
}

// CopyDirsIntoImage creates a container
func (e *dockerEngine) CopyDirsIntoImage(image, newImageName string, paths map[string]string) (err error) {
	if err := e.pullImage(image); err != nil {
		return fmt.Errorf("failed to pull the image '%s'. Error: %q", image, err)
	}
	cid, err := e.CreateContainer(environmenttypes.Container{Image: image})
	if err != nil {
		return fmt.Errorf("failed to create container with base image %s . Error: %q", image, err)
	}
	for sp, dp := range paths {
		if err := copyDirToContainer(e.ctx, e.cli, cid, sp, dp); err != nil {
			return fmt.Errorf("container data copy failed for image '%s' with volume %s:%s . Error: %q", image, sp, dp, err)
		}
	}
	if _, err := e.cli.ContainerCommit(e.ctx, cid, types.ContainerCommitOptions{Reference: newImageName}); err != nil {
		return fmt.Errorf("failed to commit the container with the input data as a new image. Error: %q", err)
	}
	e.availableImages[newImageName] = true
	if err := e.StopAndRemoveContainer(cid); err != nil {
		return fmt.Errorf("failed to stop and remove container with id '%s' . Error: %q", cid, err)
	}
	return nil
}

func (e *dockerEngine) CopyDirsIntoContainer(containerID string, paths map[string]string) (err error) {
	for sp, dp := range paths {
		err = copyDirToContainer(e.ctx, e.cli, containerID, sp, dp)
		if err != nil {
			logrus.Debugf("Container data copy failed for image %s with volume %s:%s : %s", containerID, sp, dp, err)
			return err
		}
	}
	return nil
}

func (e *dockerEngine) Stat(containerID string, name string) (fs.FileInfo, error) {
	stat, err := e.cli.ContainerStatPath(e.ctx, containerID, name)
	if err != nil {
		return nil, err
	}
	return &FileInfo{
		stat: stat,
	}, err
}

// CopyDirsFromContainer creates a container
func (e *dockerEngine) CopyDirsFromContainer(containerID string, paths map[string]string) (err error) {
	for sp, dp := range paths {
		if err := copyFromContainer(e.ctx, e.cli, containerID, sp, dp); err != nil {
			return fmt.Errorf("failed to copy data from the container with ID '%s' from source path '%s' to destination path '%s' . Error: %w", containerID, sp, dp, err)
		}
	}
	return nil
}

// BuildImage creates a container
func (e *dockerEngine) BuildImage(image, context, dockerfile string) (err error) {
	logrus.Infof("Building container image %s. This could take a few mins.", image)
	reader := common.ReadFilesAsTar(context, "", common.NoCompression)
	resp, err := e.cli.ImageBuild(e.ctx, reader, types.ImageBuildOptions{
		Dockerfile: dockerfile,
		Tags:       []string{image},
	})
	if err != nil {
		logrus.Infof("Image creation failed with image %s with no volumes : %s", image, err)
		return err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		logrus.Errorf("Unable to read data from image build process : %s", err)
		return err
	}
	logrus.Debugf("%s", response)
	e.availableImages[image] = true
	logrus.Debugf("Built image %s", image)
	return nil
}

// RemoveImage creates a container
func (e *dockerEngine) RemoveImage(image string) (err error) {
	_, err = e.cli.ImageRemove(e.ctx, image, types.ImageRemoveOptions{Force: true})
	if err != nil {
		logrus.Debugf("Container deletion failed with image %s", image)
		return err
	}
	return nil
}

// RunContainer executes a container
func (e *dockerEngine) RunContainer(image string, cmd environmenttypes.Command, volsrc string, voldest string) (output string, containerStarted bool, err error) {
	if err := e.pullImage(image); err != nil {
		return "", false, fmt.Errorf("failed to pull the image '%s'. Error: %q", image, err)
	}
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", false, fmt.Errorf("failed to create a docker client. Error: %q", err)
	}
	contconfig := &containertypes.Config{Image: image}
	if (volsrc == "" && voldest != "") || (volsrc != "" && voldest == "") {
		logrus.Warnf("Either volume source (%s) or destination (%s) is empty. Ingoring volume mount.", volsrc, voldest)
	}
	hostconfig := &containertypes.HostConfig{}
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
	resp, err := cli.ContainerCreate(ctx, contconfig, hostconfig, nil, nil, "")
	if err != nil {
		logrus.Debugf("failed to create the container with contconfig %+v and hostconfig %+v . Error: %q", contconfig, hostconfig, err)
		resp, err = cli.ContainerCreate(ctx, contconfig, nil, nil, nil, "")
		if err != nil {
			return "", false, fmt.Errorf("container creation failed for image '%s' with no volumes", image)
		}
		logrus.Debugf("Container %s created with image %s with no volumes", resp.ID, image)
		defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		if volsrc != "" && voldest != "" {
			err = copyDir(ctx, cli, resp.ID, volsrc, voldest)
			if err != nil {
				return "", false, fmt.Errorf("container data copy failed for image '%s' with volume (%s:%s). Error: %q", image, volsrc, voldest, err)
			}
			logrus.Debugf("Data copied from (%s) to (%s) in container '%s' with image '%s'", volsrc, voldest, resp.ID, image)
		}
	}
	logrus.Debugf("Container %s created with image %s", resp.ID, image)
	defer cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", false, fmt.Errorf("failed to startup the container '%s' . Error: %q", resp.ID, err)
	}
	statusCh, errCh := cli.ContainerWait(
		ctx,
		resp.ID,
		containertypes.WaitConditionNotRunning,
	)
	containerLogsStream, err := cli.ContainerLogs(ctx, resp.ID, types.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		return "", false, fmt.Errorf("failed to get the container logs. Error: %q", err)
	}
	containerLogs := bufio.NewReader(containerLogsStream)
	go func() {
		defer containerLogsStream.Close()
		text, err := containerLogs.ReadString('\n')
		for err == nil {
			updatedText := strings.TrimSpace(text)
			if updatedText != "" {
				logrus.Debugf("msg from container is: %s", updatedText)
			}
			text, err = containerLogs.ReadString('\n')
		}
		if err != nil {
			logrus.Debugf("container msg loop ended. Error: %q", err)
		}
	}()
	select {
	case err := <-errCh:
		if err != nil {
			return "", false, fmt.Errorf("error while waiting for container. Error: %q", err)
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
		if b, err := io.ReadAll(out); err == nil {
			logs = cast.ToString(b)
		}
		if status.StatusCode != 0 {
			return logs, true, fmt.Errorf("container execution terminated with error code : %d", status.StatusCode)
		}
		return logs, true, nil
	}
	return "", false, nil
}
