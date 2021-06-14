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

package container

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
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

func NewDockerEngine() (*dockerEngine, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		logrus.Debugf("Unable to create docker client : %s", err)
		return nil, err
	}
	e := &dockerEngine{
		availableImages: map[string]bool{},
		cli:             cli,
		ctx:             ctx,
	}
	_, _, err = e.RunContainer(testimage, "", "", "")
	if err != nil {
		logrus.Errorf("Unable to run test container : %s", err)
	}
	return e, err
}

func (e *dockerEngine) pullImage(image string) bool {
	if a, ok := e.availableImages[image]; ok {
		return a
	}
	out, err := e.cli.ImagePull(e.ctx, image, types.ImagePullOptions{})
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
	resp, err := e.cli.ContainerCreate(e.ctx, contconfig, hostconfig, nil, "")
	if err != nil {
		logrus.Debugf("Error during container creation : %s", err)
		resp, err = e.cli.ContainerCreate(e.ctx, contconfig, nil, nil, "")
		if err != nil {
			logrus.Debugf("Container creation failed with image %s with no volumes", image)
			return "", false, err
		}
		logrus.Debugf("Container %s created with image %s with no volumes", resp.ID, image)
		defer e.cli.ContainerRemove(e.ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		if volsrc != "" && voldest != "" {
			err = copyDirToContainer(e.ctx, e.cli, resp.ID, volsrc, voldest)
			if err != nil {
				logrus.Debugf("Container data copy failed for image %s with volume %s:%s : %s", image, volsrc, voldest, err)
				return "", false, err
			}
			logrus.Debugf("Data copied from %s to %s in container %s with image %s", volsrc, voldest, resp.ID, image)
		}
	}
	logrus.Debugf("Container %s created with image %s", resp.ID, image)
	defer e.cli.ContainerRemove(e.ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
	if err = e.cli.ContainerStart(e.ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		logrus.Debugf("Error during container startup of container %s : %s", resp.ID, err)
		return "", false, err
	}
	statusCh, errCh := e.cli.ContainerWait(
		e.ctx,
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
		out, err := e.cli.ContainerLogs(e.ctx, resp.ID, options)
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
	inspectOutput, _, err := e.cli.ImageInspectWithRaw(e.ctx, image)
	if err != nil {
		return types.ImageInspect{}, err
	}

	return inspectOutput, nil
}

// CreateContainer creates a container
func (e *dockerEngine) CreateContainer(image string) (containerid string, err error) {
	if !e.pullImage(image) {
		logrus.Debugf("Unable to pull image using docker : %s", image)
		return "", fmt.Errorf("unable to pull image")
	}
	contconfig := &container.Config{
		Image: image,
	}
	logrus.Debugf("Error during container creation : %s", err)
	resp, err := e.cli.ContainerCreate(e.ctx, contconfig, nil, nil, "")
	if err != nil {
		logrus.Debugf("Container creation failed with image %s with no volumes", image)
		return "", err
	}
	logrus.Debugf("Container %s created with image %s", resp.ID, image)
	return resp.ID, nil
}

// CreateContainer creates a container
func (e *dockerEngine) StopAndRemoveContainer(containerID string) (err error) {
	err = e.cli.ContainerRemove(e.ctx, containerID, types.ContainerRemoveOptions{Force: true})
	if err != nil {
		logrus.Errorf("Unable to delete container with containerid %s : %s", containerID, err)
		return err
	}
	return nil
}

// CopyDirsIntoImage creates a container
func (e *dockerEngine) CopyDirsIntoImage(image, newImageName string, paths map[string]string) (err error) {
	if !e.pullImage(image) {
		logrus.Debugf("Unable to pull image using docker : %s", image)
		return fmt.Errorf("unable to pull image")
	}
	cid, err := e.CreateContainer(image)
	if err != nil {
		logrus.Errorf("Unable to create container with base image %s : %s", image, err)
		return err
	}
	for sp, dp := range paths {
		err = copyDirToContainer(e.ctx, e.cli, cid, sp, dp)
		if err != nil {
			logrus.Debugf("Container data copy failed for image %s with volume %s:%s : %s", image, sp, dp, err)
			return err
		}
	}
	_, err = e.cli.ContainerCommit(e.ctx, cid, types.ContainerCommitOptions{
		Reference: newImageName,
	})
	if err != nil {
		logrus.Errorf("Unable to commit container as image : %s", err)
		return err
	}
	err = e.StopAndRemoveContainer(cid)
	if err != nil {
		logrus.Errorf("Unable to stop and remove container %s : %s", cid, err)
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

// CopyDirsFromContainer creates a container
func (e *dockerEngine) CopyDirsFromContainer(containerID string, paths map[string]string) (err error) {
	for sp, dp := range paths {
		err = copyFromContainer(e.ctx, containerID, sp, dp)
		if err != nil {
			logrus.Debugf("Container data copy failed for image %s with volume %s:%s : %s", containerID, sp, dp, err)
			return err
		}
	}
	return nil
}

// BuildImage creates a container
func (e *dockerEngine) BuildImage(image, context, dockerfile string) (err error) {
	reader := readDirAsTar(context, "")
	resp, err := e.cli.ImageBuild(e.ctx, reader, types.ImageBuildOptions{
		Dockerfile: dockerfile,
		Tags:       []string{image},
	})
	if err != nil {
		logrus.Debugf("Container creation failed with image %s with no volumes", image)
		return err
	}
	defer resp.Body.Close()
	logrus.Debugf("Built image %s", image)
	return nil
}

// RemoveImage creates a container
func (e *dockerEngine) RemoveImage(image string) (err error) {
	_, err = e.cli.ImageRemove(e.ctx, image, types.ImageRemoveOptions{})
	if err != nil {
		logrus.Debugf("Container deletion failed with image %s", image)
		return err
	}
	return nil
}
