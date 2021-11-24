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

	WorkspaceSource  string
	WorkspaceContext string

	GRPCQAReceiver net.Addr

	ImageName     string
	ImageWithData string
	CID           string // A started instance of ImageWithData
}

// NewPeerContainer creates an instance of peer container based environment
func NewPeerContainer(envInfo EnvInfo, grpcQAReceiver net.Addr, c environmenttypes.Container) (ei EnvironmentInstance, err error) {
	peerContainer := &PeerContainer{
		EnvInfo:        envInfo,
		ImageName:      c.Image,
		GRPCQAReceiver: grpcQAReceiver,
	}
	if c.WorkingDir != "" {
		peerContainer.WorkspaceContext = c.WorkingDir
	} else {
		peerContainer.WorkspaceContext = filepath.Join(string(filepath.Separator), types.AppNameShort)
	}
	peerContainer.WorkspaceSource = filepath.Join(string(filepath.Separator), DefaultWorkspaceDir)
	cengine := container.GetContainerEngine()
	if cengine == nil {
		return ei, fmt.Errorf("no working container runtime found")
	}
	newImageName := peerContainer.ImageName + strings.ToLower(envInfo.Name+uniuri.NewLen(5))
	err = cengine.CopyDirsIntoImage(peerContainer.ImageName, newImageName, map[string]string{envInfo.Source: peerContainer.WorkspaceSource})
	if err != nil {
		logrus.Debugf("Unable to create new container image with new data")
		if c.ContainerBuild.Context != "" {
			err = cengine.BuildImage(c.Image, filepath.Join(envInfo.Context, c.ContainerBuild.Context), c.ContainerBuild.Dockerfile)
			if err != nil {
				logrus.Errorf("Unable to build new container image for %s : %s", c.Image, err)
				return ei, err
			}
			err = cengine.CopyDirsIntoImage(c.Image, newImageName, map[string]string{envInfo.Source: peerContainer.WorkspaceSource})
			if err != nil {
				logrus.Errorf("Unable to copy paths to new container image : %s", err)
			}
		} else {
			return ei, err
		}
	}
	peerContainer.ImageWithData = newImageName
	cid, err := cengine.CreateContainer(newImageName)
	if err != nil {
		logrus.Errorf("Unable to start container with image %s : %s", newImageName, cid)
		return ei, err
	}
	peerContainer.CID = cid
	return peerContainer, nil
}

// Reset resets the PeerContainer environment
func (e *PeerContainer) Reset() error {
	cengine := container.GetContainerEngine()
	err := cengine.StopAndRemoveContainer(e.CID)
	if err != nil {
		logrus.Errorf("Unable to delete image %s : %s", e.ImageWithData, err)
	}
	cid, err := cengine.CreateContainer(e.ImageWithData)
	if err != nil {
		logrus.Errorf("Unable to start container with image %s : %s", e.ImageWithData, cid)
		return err
	}
	e.CID = cid
	return nil
}

// Exec executes a command in the container
func (e *PeerContainer) Exec(cmd environmenttypes.Command) (stdout string, stderr string, exitcode int, err error) {
	cengine := container.GetContainerEngine()
	envs := []string{}
	if e.GRPCQAReceiver != nil {
		hostname := getIP()
		port := cast.ToString(e.GRPCQAReceiver.(*net.TCPAddr).Port)
		envs = append(envs, GRPCEnvName+"="+hostname+":"+port)
	}
	return cengine.RunCmdInContainer(e.CID, cmd, e.WorkspaceContext, envs)
}

// Destroy destroys the container instance
func (e *PeerContainer) Destroy() error {
	cengine := container.GetContainerEngine()
	err := cengine.StopAndRemoveContainer(e.CID)
	if err != nil {
		logrus.Errorf("Unable to stop and remove container %s : %s", e.CID, err)
	}
	err = cengine.RemoveImage(e.ImageWithData)
	if err != nil {
		logrus.Errorf("Unable to delete image %s : %s", e.ImageWithData, err)
	}
	return nil
}

// Download downloads the path to outside the environment
func (e *PeerContainer) Download(path string) (string, error) {
	output, err := os.MkdirTemp(e.TempPath, "*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return path, err
	}
	cengine := container.GetContainerEngine()
	err = cengine.CopyDirsFromContainer(e.CID, map[string]string{path: output})
	if err != nil {
		logrus.Errorf("Unable to copy data from container : %s", err)
		return path, err
	}
	return output, nil
}

// Upload uploads the path from outside the environment into it
func (e *PeerContainer) Upload(outpath string) (envpath string, err error) {
	envpath = "/var/tmp/" + uniuri.NewLen(5) + "/" + filepath.Base(outpath)
	cengine := container.GetContainerEngine()
	err = cengine.CopyDirsIntoContainer(e.CID, map[string]string{outpath: envpath})
	if err != nil {
		logrus.Errorf("Unable to copy data from container : %s", err)
		return outpath, err
	}
	return envpath, nil
}

// GetContext returns the context within the PeerContainer environment
func (e *PeerContainer) GetContext() string {
	return e.WorkspaceContext
}

// GetSource returns the source path within the PeerContainer environment
func (e *PeerContainer) GetSource() string {
	return e.WorkspaceSource
}
