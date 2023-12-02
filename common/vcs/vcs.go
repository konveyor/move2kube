/*
 *  Copyright IBM Corporation 2023
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
	Overwrite            bool
	MaxSize              int64
	CloneDestinationPath string
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

var (
	// maxRepoCloneSize is the maximum size (in bytes) allowed when cloning VCS repos
	// default -1 means infinite
	maxRepoCloneSize int64 = -1
)

// SetMaxRepoCloneSize sets the maximum size (in bytes) for cloning a repo
func SetMaxRepoCloneSize(size int64) {
	maxRepoCloneSize = size
}

// Error returns the error message for no valid vcs is found
func (e *NoCompatibleVCSFound) Error() string {
	return fmt.Sprintf("no valid vcs is match for the given input %s", e.URLInput)
}

// Error returns the error message for failed push
func (e *FailedVCSPush) Error() string {
	return fmt.Sprintf("failed to commit and push for the given VCS path %s. Error: %+v", e.VCSPath, e.Err)
}

// IsRemotePath returns if the provided input is a remote path or not
func IsRemotePath(input string) bool {
	return isGitVCS(input)
}

// PushVCSRepo commits and pushes the changes in the provide vcs remote path
func PushVCSRepo(remotePath, folderName string) error {
	return pushGitVCS(remotePath, folderName, maxRepoCloneSize)
}

// GetVCSRepo extracts information from the given vcsurl and returns a relevant vcs repo struct
func GetVCSRepo(vcsurl string) (VCS, error) {
	if isGitVCS(vcsurl) {
		vcsRepo, err := getGitRepoStruct(vcsurl)
		if err != nil {
			return nil, fmt.Errorf("failed to get git vcs repo for the input '%s' . Error: %w", vcsurl, err)
		}
		return vcsRepo, nil
	}
	return nil, &NoCompatibleVCSFound{URLInput: vcsurl}
}

// GetClonedPath takes a vcsurl and a folder name,
// performs a clone with the appropriate VCS,
// and then returns the file system and remote paths.
// If the VCS is not supported, the returned path will be an empty string.
func GetClonedPath(vcsurl, destDirName string, overwrite bool) (string, error) {
	vcsRepo, err := GetVCSRepo(vcsurl)
	if err != nil {
		if _, ok := err.(*NoCompatibleVCSFound); ok {
			logrus.Debugf("the vcsurl '%s' is not compatible with any supported VCS. Error: %+v", vcsurl, err)
			return "", nil
		}
		return "", fmt.Errorf("failed to get a VCS for the provided vcsurl '%s'. Error: %w", vcsurl, err)
	}
	logrus.Debugf("vcsRepo: %+v", vcsRepo)
	tempPath, err := filepath.Abs(common.RemoteTempPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for the temp path '%s'", common.RemoteTempPath)
	}
	cloneOpts := VCSCloneOptions{
		CommitDepth:          1,
		Overwrite:            overwrite,
		MaxSize:              maxRepoCloneSize,
		CloneDestinationPath: filepath.Join(tempPath, destDirName),
	}
	vcsSrcPath, err := vcsRepo.Clone(cloneOpts)
	if err != nil {
		return "", fmt.Errorf("failed to clone using vcs url '%s' and clone options %+v. Error: %w", vcsurl, cloneOpts, err)
	}
	return vcsSrcPath, nil
}
