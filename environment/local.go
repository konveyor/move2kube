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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/types"
	"github.com/sirupsen/logrus"
)

type Local struct {
	Name     string
	Source   string
	Context  string
	TempPath string

	WorkspaceSource  string
	WorkspaceContext string
}

func NewLocal(name, source, context, tempPath string) (ei EnvironmentInstance, err error) {
	local := &Local{
		Name:    name,
		Source:  source,
		Context: context,
	}
	local.TempPath = tempPath
	local.WorkspaceContext, err = ioutil.TempDir(local.TempPath, types.AppNameShort)
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return local, err
	}
	local.WorkspaceSource, err = ioutil.TempDir(local.TempPath, workspaceDir)
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
	}
	local.Reset()
	return local, nil
}

func (e *Local) Reset() error {
	if err := filesystem.Replicate(e.Context, e.WorkspaceContext); err != nil {
		logrus.Errorf("Unable to copy contents to directory %s, dp: %s", e.Context, e.WorkspaceContext, err)
		return err
	}
	if err := filesystem.Replicate(e.Source, e.WorkspaceSource); err != nil {
		logrus.Errorf("Unable to copy contents to directory %s, dp: %s", e.Source, e.WorkspaceSource, err)
		return err
	}
	return nil
}

func (e *Local) Exec(cmd []string) (string, string, int, error) {
	var exitcode int
	var outb, errb bytes.Buffer
	var execcmd *exec.Cmd
	if len(cmd) > 0 {
		execcmd = exec.Command(cmd[0], cmd[1:]...)
	} else {
		err := fmt.Errorf("no command found to execute")
		logrus.Errorf("%s", err)
		return "", "", 0, err
	}
	execcmd.Dir = e.Context
	execcmd.Dir = e.Context
	execcmd.Stdout = &outb
	execcmd.Stderr = &errb
	err := execcmd.Run()
	if err != nil {
		var ee *exec.ExitError
		var pe *os.PathError
		if errors.As(err, &ee) {
			exitcode = ee.ExitCode()
			err = nil
		} else if errors.As(err, &pe) {
			logrus.Errorf("PathError during execution of command: %v", pe)
			err = pe
		} else {
			logrus.Errorf("Generic error during execution of command: %v", err)
		}
	}
	return outb.String(), errb.String(), exitcode, err
}

func (e *Local) Destroy() error {
	err := os.RemoveAll(e.WorkspaceSource)
	if err != nil {
		logrus.Errorf("Unable to remove directory %s : %s", e.WorkspaceSource, err)
	}
	err = os.RemoveAll(e.WorkspaceContext)
	if err != nil {
		logrus.Errorf("Unable to remove directory %s : %s", e.WorkspaceContext, err)
	}
	return nil
}

func (e *Local) Download(path string) (string, error) {
	output, err := ioutil.TempDir(e.TempPath, "*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return path, err
	}
	err = filesystem.Replicate(path, output)
	if err != nil {
		logrus.Errorf("Unable to replicate in syncoutput : %s", err)
		return path, err
	}
	return path, nil
}

func (e *Local) GetContext() string {
	return e.WorkspaceContext
}

func (e *Local) GetSource() string {
	logrus.Infof("Workspace source : %s", e.WorkspaceSource)
	return e.WorkspaceSource
}
