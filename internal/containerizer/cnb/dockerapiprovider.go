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

package cnb

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/spf13/cast"

	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/common"
)

type dockerAPIProvider struct {
}

var (
	isSockAccessible      = "unknown"
	availableDockerImages = []string{}
)

func (r *dockerAPIProvider) getAllBuildpacks(builders []string) (map[string][]string, error) { //[Containerization target option value] buildpacks
	buildpacks := map[string][]string{}
	if available := r.isSockAccessible(); !available {
		return buildpacks, errors.New("Container runtime not supported in this instance")
	}
	log.Debugf("Getting data of all builders %s", builders)
	for _, builder := range builders {
		inspectOutput, err := r.inspectImage(builder)
		log.Debugf("Inspecting image %s", builder)
		if err != nil {
			log.Debugf("Unable to inspect image %s : %s, %+v", builder, err, inspectOutput)
			continue
		}
		buildpacks[builder] = getBuildersFromLabel(inspectOutput.Config.Labels[orderLabel])
	}

	return buildpacks, nil
}

func (r *dockerAPIProvider) isSockAccessible() bool {
	if isSockAccessible == "unknown" {
		image := "hello-world"
		err := r.pullImage(image)
		if err != nil {
			isSockAccessible = "false"
			return false
		}
		_, err = r.runContainer(image, "", "", "")
		if err != nil {
			isSockAccessible = "false"
			return false
		}
		isSockAccessible = "true"
		return true
	}
	return cast.ToBool(isSockAccessible)
}

func (r *dockerAPIProvider) isBuilderAvailable(builder string) bool {
	if !r.isSockAccessible() {
		return false
	}
	if common.IsStringPresent(availableDockerImages, builder) {
		return true
	}

	err := r.pullImage(builder)
	log.Debugf("Pulling image %s", builder)
	if err != nil {
		log.Warnf("Error while pulling builder %s : %s", builder, err)
		return false
	}
	availableDockerImages = append(availableDockerImages, builder)
	return true
}

func (r *dockerAPIProvider) isBuilderSupported(path string, builder string) (bool, error) {
	if !r.isBuilderAvailable(builder) {
		return false, fmt.Errorf("Builder image not available : %s", builder)
	}
	p, err := filepath.Abs(path)
	if err != nil {
		log.Warnf("Unable to resolve to absolute path : %s", err)
	}
	output, err := r.runContainer(builder, "/cnb/lifecycle/detector", p, "/workspace")
	log.Debugf("Running detect on image %s", builder)
	if err != nil {
		log.Debugf("Detect failed %s : %s : %s", builder, err, output)
		return false, nil
	}
	return true, nil
}

func (r *dockerAPIProvider) pullImage(image string) error {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	out, err := cli.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	if b, err := ioutil.ReadAll(out); err == nil {
		log.Debug(b)
	}
	return nil
}

func (r *dockerAPIProvider) inspectImage(image string) (types.ImageInspect, error) {
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

func (r *dockerAPIProvider) runContainer(image string, cmd string, volsrc string, voldest string) (output string, err error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	contconfig := &container.Config{
		Image: image,
	}
	if cmd != "" {
		contconfig.Cmd = []string{cmd}
	}
	hostconfig := &container.HostConfig{}
	if volsrc != "" && voldest != "" {
		hostconfig.Mounts = []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: volsrc,
				Target: voldest,
			},
		}
	}
	resp, err := cli.ContainerCreate(ctx, contconfig, hostconfig, nil, "")
	if err != nil {
		return "", err
	}
	if err = cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", err
	}

	statusCh, errCh := cli.ContainerWait(
		ctx,
		resp.ID,
		container.WaitConditionNotRunning,
	)
	select {
	case err := <-errCh:
		if err != nil {
			log.Debug(err)
			return "", err
		}
	case status := <-statusCh:
		log.Debugf("Container exited with status code: %#+v", status.StatusCode)
		if status.StatusCode != 0 {
			return "", fmt.Errorf("Container execution terminated with error code : %d", status.StatusCode)
		}
	}

	options := types.ContainerLogsOptions{ShowStdout: true}
	// Replace this ID with a container that really exists
	out, err := cli.ContainerLogs(ctx, resp.ID, options)
	if err != nil {
		return "", err
	}
	if b, err := ioutil.ReadAll(out); err == nil {
		return cast.ToString(b), nil
	}
	if err = cli.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{}); err != nil {
		return "", err
	}
	return "", err
}
