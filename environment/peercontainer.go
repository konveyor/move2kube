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

package environment

import (
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/konveyor/move2kube/environment/container"
	"github.com/konveyor/move2kube/types"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

const (
	// DefaultWorkspaceDir is the default workspace directory
	DefaultWorkspaceDir = "workspace"
)

// PeerContainer is supports spawning peer containers to run the environment
type PeerContainer struct {
	EnvInfo
	// WorkspaceSource is the directory where the input data resides in the new image.
	WorkspaceSource string
	// OriginalImage is the original image name (for the image without the input data).
	OriginalImage string
	// ContainerInfo contains info about the running container.
	ContainerInfo environmenttypes.Container
	// GRPCQAReceiver is used to ask questions and get answers using the GRPC protocol.
	GRPCQAReceiver net.Addr
}

// NewPeerContainer creates an instance of peer container based environment
func NewPeerContainer(envInfo EnvInfo, grpcQAReceiver net.Addr, containerInfo environmenttypes.Container, spawnContainers bool) (EnvironmentInstance, error) {
	cengine, err := container.GetContainerEngine(spawnContainers)
	if err != nil {
		return nil, fmt.Errorf("failed to get the container engine. Error: %w", err)
	}
	if containerInfo.WorkingDir == "" {
		containerInfo.WorkingDir = filepath.Join(string(filepath.Separator), types.AppNameShort)
	}
	peerContainer := &PeerContainer{
		EnvInfo:        envInfo,
		OriginalImage:  containerInfo.Image,
		ContainerInfo:  containerInfo,
		GRPCQAReceiver: grpcQAReceiver,
	}
	peerContainer.WorkspaceSource = filepath.Join(string(filepath.Separator), DefaultWorkspaceDir)
	peerContainer.ContainerInfo.Image = peerContainer.ContainerInfo.Image + strings.ToLower(envInfo.Name+uniuri.NewLen(5))
	logrus.Debug("trying to create a new image with the input data")
	if err := cengine.CopyDirsIntoImage(
		peerContainer.OriginalImage,
		peerContainer.ContainerInfo.Image,
		map[string]string{envInfo.Source: peerContainer.WorkspaceSource},
	); err != nil {
		err := fmt.Errorf("failed to create a new container image with the input data copied into the container. Error: %w", err)
		if peerContainer.ContainerInfo.ImageBuild.Context == "" {
			return nil, err
		}
		logrus.Debug(err)
		logrus.Debug("trying to build the original image before creating a new image with the input data")
		imageBuildContext := filepath.Join(envInfo.Context, peerContainer.ContainerInfo.ImageBuild.Context)
		if err := cengine.BuildImage(
			peerContainer.ContainerInfo.Image,
			imageBuildContext,
			peerContainer.ContainerInfo.ImageBuild.Dockerfile,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to build the container image '%s' using the context directory '%s' and the Dockerfile at path '%s' . Error: %w",
				peerContainer.ContainerInfo.Image,
				imageBuildContext,
				peerContainer.ContainerInfo.ImageBuild.Dockerfile,
				err,
			)
		}
		if err := cengine.CopyDirsIntoImage(
			peerContainer.OriginalImage,
			peerContainer.ContainerInfo.Image,
			map[string]string{envInfo.Source: peerContainer.WorkspaceSource},
		); err != nil {
			return nil, fmt.Errorf("failed to copy paths to new container image. Error: %w", err)
		}
	}
	cid, err := cengine.CreateContainer(peerContainer.ContainerInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to create a container with the new image '%s' . Error: %w", peerContainer.ContainerInfo.Image, err)
	}
	peerContainer.ContainerInfo.ID = cid
	return peerContainer, nil
}

// Reset resets the PeerContainer environment
func (e *PeerContainer) Reset() error {
	cengine, err := container.GetContainerEngine(false)
	if err != nil {
		return fmt.Errorf("failed to get the container engine. Error: %w", err)
	}
	if err := cengine.StopAndRemoveContainer(e.ContainerInfo.ID); err != nil {
		return fmt.Errorf("failed to delete the image '%s' . Error: %q", e.ContainerInfo.Image, err)
	}
	cid, err := cengine.CreateContainer(e.ContainerInfo)
	if err != nil {
		return fmt.Errorf("failed to start a container with the info: %+v . Error: %w", e.ContainerInfo, err)
	}
	e.ContainerInfo.ID = cid
	return nil
}

// Stat returns stat info of the file/dir in the env
func (e *PeerContainer) Stat(name string) (fs.FileInfo, error) {
	cengine, err := container.GetContainerEngine(false)
	if err != nil {
		return nil, fmt.Errorf("failed to get the container engine. Error: %w", err)
	}
	return cengine.Stat(e.ContainerInfo.ID, name)
}

// Exec executes a command in the container
func (e *PeerContainer) Exec(cmd environmenttypes.Command) (stdout string, stderr string, exitcode int, err error) {
	cengine, err := container.GetContainerEngine(false)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get the container engine. Error: %w", err)
	}
	envs := []string{}
	if e.GRPCQAReceiver != nil {
		hostname := getIP()
		port := cast.ToString(e.GRPCQAReceiver.(*net.TCPAddr).Port)
		envs = append(envs, GRPCEnvName+"="+hostname+":"+port)
	}
	return cengine.RunCmdInContainer(e.ContainerInfo.ID, cmd, e.ContainerInfo.WorkingDir, envs)
}

// Destroy destroys the container instance
func (e *PeerContainer) Destroy() error {
	cengine, err := container.GetContainerEngine(false)
	if err != nil {
		return fmt.Errorf("failed to get the container engine. Error: %w", err)
	}
	if err := cengine.StopAndRemoveContainer(e.ContainerInfo.ID); err != nil {
		return fmt.Errorf("failed to stop and remove the container with ID '%s' . Error: %w", e.ContainerInfo.ID, err)
	}
	if err := cengine.RemoveImage(e.ContainerInfo.Image); err != nil {
		return fmt.Errorf("failed to delete the image '%s' . Error :%w", e.ContainerInfo.Image, err)
	}
	return nil
}

// Download downloads the path to outside the environment
func (e *PeerContainer) Download(path string) (string, error) {
	output, err := os.MkdirTemp(e.TempPath, "*")
	if err != nil {
		return path, fmt.Errorf("failed to create temp dir. Error: %w", err)
	}
	cengine, err := container.GetContainerEngine(false)
	if err != nil {
		return "", fmt.Errorf("failed to get the container engine. Error: %w", err)
	}
	if err := cengine.CopyDirsFromContainer(e.ContainerInfo.ID, map[string]string{path: output}); err != nil {
		return path, fmt.Errorf("failed to copy data from the container with ID '%s' . Error: %w", e.ContainerInfo.ID, err)
	}
	return output, nil
}

// Upload uploads the path from outside the environment into it
func (e *PeerContainer) Upload(outpath string) (string, error) {
	envpath := "/var/tmp/" + uniuri.NewLen(5) + "/" + filepath.Base(outpath)
	cengine, err := container.GetContainerEngine(false)
	if err != nil {
		return envpath, fmt.Errorf("failed to get the container engine. Error: %w", err)
	}
	if err := cengine.CopyDirsIntoContainer(e.ContainerInfo.ID, map[string]string{outpath: envpath}); err != nil {
		return envpath, fmt.Errorf("failed to copy data into the container with ID '%s' . Error: %w", e.ContainerInfo.ID, err)
	}
	return envpath, nil
}

// GetContext returns the working directory inside the container.
func (e *PeerContainer) GetContext() string {
	return e.ContainerInfo.WorkingDir
}

// GetSource returns the directory where the input data resides in the new image.
func (e *PeerContainer) GetSource() string {
	return e.WorkspaceSource
}
