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
package githelper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

type gitRemotePath struct {
	URL            string
	Branch         string
	Tag            string
	Commit         string
	PathWithinRepo string
}

// GetGitRemotePathStruct extracts information from the given git path and returns a struct
func GetGitRemotePathStruct(gitremotepath string) (*gitRemotePath, error) {
	// reference format - git+ssh://<git remote URL>^<branch,tag,commit>^<path within the repo>
	// reference format - git+https://<git remote URL>^<branch,tag,commit>^<path within the repo>
	// reference format - git+ssh://<git remote URL>^branch_<branchname>^<path within the repo>
	// reference format - git+https://<git remote URL>^branch_<branchname>^<path within the repo>

	parts := strings.Split(gitremotepath, "^")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid git remote path provided. Should follow the format git@<git remote URL>^<branch,tag,commit>^<path within the repo>. Received : %s", gitremotepath)
	}
	gitremotepathstruct := gitRemotePath{}

	gitremotepathstruct.URL = parts[0]
	gitremotepathstruct.PathWithinRepo = parts[2]

	switch spec := strings.Split(parts[1], "_")[0]; spec {
	case "branch":
		gitremotepathstruct.Branch = strings.TrimPrefix(parts[1], spec)
	case "tag":
		gitremotepathstruct.Tag = strings.TrimPrefix(parts[1], spec)
	case "commit":
		gitremotepathstruct.Commit = strings.TrimPrefix(parts[1], spec)
	}

	return &gitremotepathstruct, nil

}

// IsGitRemotePath checks if the given path is a git remote path or not
func IsGitRemotePath(gitremotepath string) bool {
	// TODO: enhance check with regex
	return (strings.HasPrefix(gitremotepath, "git@") || strings.HasPrefix(gitremotepath, "https")) && len(strings.Split(gitremotepath, "^")) == 3
}

// GitClone Clones a git repository with the given commit depth and path where to be cloned and returns final path
func (grp *gitRemotePath) GitClone(commitdepth int, basepath string) (string, error) {
	cloneOpts := git.CloneOptions{
		URL:           grp.URL,
		Depth:         commitdepth,
		SingleBranch:  true,
		ReferenceName: "refs/heads/main",
	}
	destFolder := strings.ReplaceAll(grp.URL, "/", "_")
	destpath := filepath.Join(basepath, destFolder)
	_, err := os.Stat(destpath)
	if err == nil {
		return filepath.Join(destpath, grp.PathWithinRepo), err
	}
	_, err = git.PlainClone(destpath, false, &cloneOpts)

	return filepath.Join(destpath, grp.PathWithinRepo), err

}
