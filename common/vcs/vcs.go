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

package vcs

import (
	"fmt"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

// VCSCloneOptions stores version control system clone options
type VCSCloneOptions struct {
	CommitDepth          int
	CloneDestinationPath string
	Overwrite            bool
}

// VCS defines interface for version control system
type VCS interface {
	Clone(VCSCloneOptions) (string, error)
}

// NoCompatibleVCSFound is the error when no VCS is found suitable for the given remote input path
type NoCompatibleVCSFound struct {
	URLInput string
}

// FailedVCSPush is the error when push is failed
type FailedVCSPush struct {
	VCSPath string
	Err     error
}

// GetClonedPath takes input and folder name performs clone with the appropriate VCS and then return the file system and remote paths
func GetClonedPath(input, folderName string, overwrite bool) string {
	vcsRepo, err := GetVCSRepo(input)
	var vcsSrcPath string
	if err != nil {
		_, ok := err.(*NoCompatibleVCSFound)
		if ok {
			logrus.Debugf("source path provided is not compatible with any of the supported VCS. Info : %q", err)
		} else {
			logrus.Fatalf("failed to get vcs repo for the provided source path %s. Error : %v", input, err)
		}
		vcsSrcPath = ""
	} else {
		tempPath, err := filepath.Abs(common.RemoteTempPath)
		if err != nil {
			logrus.Fatalf("failed to get absolute path for the temp path - %s", tempPath)
		}
		cloneOpts := VCSCloneOptions{CommitDepth: 1, Overwrite: overwrite, CloneDestinationPath: filepath.Join(tempPath, folderName)}
		logrus.Debugf("%+v", vcsRepo)
		vcsSrcPath, err = vcsRepo.Clone(cloneOpts)
		if err != nil {
			logrus.Fatalf("failed to clone a repository with the provided vcs url %s and clone options %+v. Error : %+v", input, cloneOpts, err)
		}
	}
	return vcsSrcPath
}

// IsRemotePath returns if the provided input is a remote path or not
func IsRemotePath(input string) bool {
	return isGitVCS(input)
}

// Error returns the error message for no valid vcs is found
func (e *NoCompatibleVCSFound) Error() string {
	return fmt.Sprintf("no valid vcs is match for the given input %s", e.URLInput)
}

// Error returns the error message for failed push
func (e *FailedVCSPush) Error() string {
	return fmt.Sprintf("failed to commit and push for the given VCS path %s. Error : %+v", e.VCSPath, e.Err)
}

// GetVCSRepo extracts information from the given vcsurl and returns a relevant vcs repo struct
func GetVCSRepo(vcsurl string) (VCS, error) {
	if isGitVCS(vcsurl) {
		vcsRepo, err := getGitRepoStruct(vcsurl)
		if err != nil {
			return nil, fmt.Errorf("failed to get git vcs repo for the input %s. Error : %v", vcsurl, err)
		}
		return vcsRepo, nil
	}
	return nil, &NoCompatibleVCSFound{URLInput: vcsurl}
}

func PushVCSRepo(remotePath, folderName string) error {
	return pushGitVCS(remotePath, folderName)
}
