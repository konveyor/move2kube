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
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/filesystem"
	"github.com/konveyor/move2kube/types"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

// Local manages a local machine environment
type Local struct {
	EnvInfo
	WorkspaceSource  string
	WorkspaceContext string
	GRPCQAReceiver   net.Addr
}

// NewLocal creates a new Local environment
func NewLocal(envInfo EnvInfo, grpcQAReceiver net.Addr) (EnvironmentInstance, error) {
	local := &Local{
		EnvInfo:        envInfo,
		GRPCQAReceiver: grpcQAReceiver,
	}
	if envInfo.Isolated {
		var err error
		local.WorkspaceContext, err = os.MkdirTemp(local.TempPath, types.AppNameShort)
		if err != nil {
			return local, fmt.Errorf("failed to create the temp directory at path '%s' with pattern '%s' . Error: %w", local.TempPath, types.AppNameShort, err)
		}
		local.WorkspaceSource, err = os.MkdirTemp(local.TempPath, workspaceDir)
		if err != nil {
			return local, fmt.Errorf("failed to create the temp directory at path '%s' with pattern '%s' . Error: %w", local.TempPath, workspaceDir, err)
		}
	} else {
		local.WorkspaceContext = local.Context
		local.WorkspaceSource = local.Source
	}

	if err := local.Reset(); err != nil {
		return local, fmt.Errorf("failed to reset the local environment. Error: %w", err)
	}
	return local, nil
}

// Reset resets the environment to fresh state
func (e *Local) Reset() error {
	if e.Isolated {
		if err := filesystem.Replicate(e.Context, e.WorkspaceContext); err != nil {
			return fmt.Errorf("failed to copy contents to directory %s, %s . Error: %w", e.Context, e.WorkspaceContext, err)
		}
		if err := filesystem.Replicate(e.Source, e.WorkspaceSource); err != nil {
			return fmt.Errorf("failed to copy contents to directory %s, %s . Error: %w", e.Source, e.WorkspaceSource, err)
		}
	}
	return nil
}

// Stat returns stat info of the file/dir in the env
func (e *Local) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}

// AddEnvironmentVariablesToInstance adds the environment variables after the environment is created
func (e *Local) AddEnvironmentVariablesToInstance(envList []string) error {
	e.EnvKeyValueList = append(e.EnvKeyValueList, envList...)
	return nil
}

// Exec executes an executable within the environment
func (e *Local) Exec(cmd environmenttypes.Command) (stdout string, stderr string, exitcode int, err error) {
	if common.DisableLocalExecution {
		return "", "", 0, fmt.Errorf("local execution prevented by %s flag", common.DisableLocalExecutionFlag)
	}
	var outb, errb bytes.Buffer
	var execcmd *exec.Cmd
	if len(cmd) > 0 {
		execcmd = exec.Command(cmd[0], cmd[1:]...)
	} else {
		return "", "", 0, fmt.Errorf("no command found to execute")
	}
	execcmd.Dir = e.WorkspaceContext
	execcmd.Stdout = &outb
	execcmd.Stderr = &errb
	execcmd.Env = e.getEnv()
	execcmd.Env = append(execcmd.Env, e.EnvKeyValueList...)
	if err := execcmd.Run(); err != nil {
		var ee *exec.ExitError
		var pe *os.PathError
		if errors.As(err, &ee) {
			exitcode = ee.ExitCode()
			err = nil
		} else if errors.As(err, &pe) {
			logrus.Errorf("PathError during execution of command: %v", pe)
			err = pe
		} else {
			logrus.Errorf("Generic error during execution of command. Error: %q", err)
		}
	}
	return outb.String(), errb.String(), exitcode, err
}

// Destroy destroys all artifacts specific to the environment
func (e *Local) Destroy() error {
	if e.Isolated {
		if err := os.RemoveAll(e.WorkspaceSource); err != nil {
			return fmt.Errorf("failed to remove the workspace source directory '%s' . Error: %w", e.WorkspaceSource, err)
		}
		if err := os.RemoveAll(e.WorkspaceContext); err != nil {
			return fmt.Errorf("failed to remove the workspace context directory '%s' . Error: %w", e.WorkspaceContext, err)
		}
	}
	return nil
}

// Download downloads the path to outside the environment
func (e *Local) Download(sourcePath string) (string, error) {
	destPath, err := os.MkdirTemp(e.TempPath, "*")
	if err != nil {
		return sourcePath, fmt.Errorf("failed to create the temp dir at path '%s' with pattern '*' . Error: %w", e.TempPath, err)
	}
	ps, err := os.Stat(sourcePath)
	if err != nil {
		return sourcePath, fmt.Errorf("failed to stat source directory at path '%s' . Error: %w", sourcePath, err)
	}
	if ps.Mode().IsRegular() {
		destPath = filepath.Join(destPath, filepath.Base(sourcePath))
	}
	if err := filesystem.Replicate(sourcePath, destPath); err != nil {
		return sourcePath, fmt.Errorf("failed to replicate in sync output from source path '%s' to destination path '%s' . Error: %w", sourcePath, destPath, err)
	}
	return destPath, nil
}

// Upload uploads the path from outside the environment into it
func (e *Local) Upload(sourcePath string) (string, error) {
	destPath, err := os.MkdirTemp(e.TempPath, "*")
	if err != nil {
		return destPath, fmt.Errorf("failed to create the temp dir at path '%s' with pattern '*' . Error: %w", e.TempPath, err)
	}
	ps, err := os.Stat(sourcePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat source '%s' . Error: %w", sourcePath, err)
	}
	if ps.Mode().IsRegular() {
		destPath = filepath.Join(destPath, filepath.Base(sourcePath))
	}
	if err := filesystem.Replicate(sourcePath, destPath); err != nil {
		return destPath, fmt.Errorf("failed to replicate in sync output from source path '%s' to destination path '%s' . Error: %w", sourcePath, destPath, err)
	}
	return destPath, nil
}

// GetContext returns the context of Local
func (e *Local) GetContext() string {
	return e.WorkspaceContext
}

// GetSource returns the source of Local
func (e *Local) GetSource() string {
	return e.WorkspaceSource
}

func (e *Local) getEnv() []string {
	environ := os.Environ()
	if e.GRPCQAReceiver != nil {
		environ = append(environ, GRPCEnvName+"="+e.GRPCQAReceiver.String())
	}
	return environ
}
