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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/types"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

// Local manages a local machine environment
type Local struct {
	Name     string
	Source   string
	Context  string
	TempPath string

	WorkspaceSource  string
	WorkspaceContext string

	GRPCQAReceiver net.Addr
}

// NewLocal creates a new Local environment
func NewLocal(name, source, context, tempPath string, grpcQAReceiver net.Addr) (ei EnvironmentInstance, err error) {
	local := &Local{
		Name:           name,
		Source:         source,
		Context:        context,
		GRPCQAReceiver: grpcQAReceiver,
		TempPath:       tempPath,
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

// Reset resets the environment to fresh state
func (e *Local) Reset() error {
	if err := filesystem.Replicate(e.Context, e.WorkspaceContext); err != nil {
		logrus.Errorf("Unable to copy contents to directory %s, %s : %s", e.Context, e.WorkspaceContext, err)
		return err
	}
	if err := filesystem.Replicate(e.Source, e.WorkspaceSource); err != nil {
		logrus.Errorf("Unable to copy contents to directory %s, %s : %s", e.Source, e.WorkspaceSource, err)
		return err
	}
	return nil
}

// Exec executes an executable within the environment
func (e *Local) Exec(cmd environmenttypes.Command) (string, string, int, error) {
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
	execcmd.Dir = e.WorkspaceContext
	execcmd.Stdout = &outb
	execcmd.Stderr = &errb
	execcmd.Env = os.Environ()
	if e.GRPCQAReceiver != nil {
		execcmd.Env = append(execcmd.Env, GRPCEnvName+"="+e.GRPCQAReceiver.String())
	}
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

// Destroy destroys all artifacts specific to the environment
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

// Download downloads the path to outside the environment
func (e *Local) Download(path string) (string, error) {
	output, err := ioutil.TempDir(e.TempPath, "*")
	if err != nil {
		logrus.Errorf("Unable to create temp dir : %s", err)
		return path, err
	}
	ps, err := os.Stat(path)
	if err != nil {
		logrus.Errorf("Unable to stat source : %s", path)
		return "", err
	}
	if ps.Mode().IsRegular() {
		output = filepath.Join(output, filepath.Base(path))
	}
	err = filesystem.Replicate(path, output)
	if err != nil {
		logrus.Errorf("Unable to replicate in syncoutput : %s", err)
		return path, err
	}
	return output, nil
}

// GetContext returns the context of Local
func (e *Local) GetContext() string {
	return e.WorkspaceContext
}

// GetSource returns the source of Local
func (e *Local) GetSource() string {
	return e.WorkspaceSource
}
