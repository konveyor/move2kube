/*
Copyright IBM Corporation 2021

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

package environment

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/dchest/uniuri"
	"github.com/konveyor/move2kube/environment/container"
	"github.com/konveyor/move2kube/types"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

const (
	DefaultWorkspaceDir = "workspace"
)

type PeerContainer struct {
	Name     string
	Source   string
	TempPath string

	WorkspaceSource  string
	WorkspaceContext string

	ImageName     string
	ImageWithData string
	CID           string // A started instance of ImageWithData
}

func NewPeerContainer(name, source, context, tempPath string, c environmenttypes.Container) (ei EnvironmentInstance, err error) {
	peerContainer := &PeerContainer{
		Name:      name,
		Source:    source,
		TempPath:  tempPath,
		ImageName: c.Image,
	}
	peerContainer.TempPath = tempPath
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
	newImageName := peerContainer.ImageName + strings.ToLower(name+uniuri.NewLen(5))
	err = cengine.CopyDirsIntoImage(peerContainer.ImageName, newImageName, map[string]string{source: peerContainer.WorkspaceSource})
	if err != nil {
		logrus.Debugf("Unable to create new container image with new data")
		if c.ContainerBuild.Context != "" {
			err = cengine.BuildImage(c.Image, filepath.Join(context, c.ContainerBuild.Context), c.ContainerBuild.Dockerfile)
			if err != nil {
				logrus.Errorf("Unable to build new container image for %s : %s", c.Image, err)
				return ei, err
			}
			err = cengine.CopyDirsIntoImage(c.Image, newImageName, map[string]string{source: peerContainer.WorkspaceSource})
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

func (e *PeerContainer) Exec(cmd environmenttypes.Command) (string, string, int, error) {
	cengine := container.GetContainerEngine()
	return cengine.RunCmdInContainer(e.CID, cmd, e.WorkspaceContext)
}

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

func (e *PeerContainer) Download(path string) (string, error) {
	output, err := ioutil.TempDir(e.TempPath, "*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return path, err
	}
	if ps, err := os.Stat(path); err != nil {
		logrus.Errorf("Unable to stat source : %s", path)
		return "", err
	} else {
		if ps.Mode().IsRegular() {
			output = filepath.Join(output, filepath.Base(path))
		}
	}
	cengine := container.GetContainerEngine()
	err = cengine.CopyDirsFromContainer(e.CID, map[string]string{path: output})
	if err != nil {
		logrus.Errorf("Unable to copy data from container : %s", err)
		return path, err
	}
	return output, nil
}

func (e *PeerContainer) GetContext() string {
	return e.WorkspaceContext
}

func (e *PeerContainer) GetSource() string {
	return e.WorkspaceSource
}
