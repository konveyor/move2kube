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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/sirupsen/logrus"
)

// GitVCSRepo stores git repo config
type GitVCSRepo struct {
	InputURL       string
	URL            string
	Branch         string
	Tag            string
	CommitHash     string
	PathWithinRepo string
	GitRepository  *git.Repository
	GitRepoPath    string
}

func isGitCommitHash(commithash string) bool {
	gitCommitHashRegex := regexp.MustCompile(`^[a-fA-F0-9]{40}$`)
	return gitCommitHashRegex.MatchString(commithash)
}

func isGitBranch(branch string) bool {
	gitBranchRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9\.\-_\/]*$`)
	return gitBranchRegex.MatchString(branch)
}

func isGitTag(tag string) bool {
	gitTagRegex := regexp.MustCompile(`^v[0-9]+(\.[0-9]+)?(\.[0-9]+)?$`)
	return gitTagRegex.MatchString(tag)
}

// getGitRepoStruct extracts information from the given git path and returns a struct
func getGitRepoStruct(vcsurl string) (*GitVCSRepo, error) {
	// for format visit https://move2kube.konveyor.io/concepts/git-support
	partsSplitByAt := strings.Split(vcsurl, "@")
	if len(partsSplitByAt) > 2 {
		return nil, fmt.Errorf("invalid git remote path provided. Should follow the format git+[ssh|https]://<URL>@[tag|commit hash|branch]:/path/in/the/repo but received : %s", vcsurl)
	}
	gitRepoStruct := GitVCSRepo{}
	gitRepoStruct.InputURL = vcsurl
	partsSplitByColon := strings.Split(partsSplitByAt[0], ":")
	gitUrl := partsSplitByAt[0]
	if len(partsSplitByColon) == 3 {
		gitRepoStruct.PathWithinRepo = partsSplitByColon[2]
		gitUrl = strings.Join([]string{partsSplitByColon[0], partsSplitByColon[1]}, ":")
	}
	if strings.HasPrefix(gitUrl, "git+https") {
		hostNameRegex := regexp.MustCompile(`git\+https:\/\/(.*?)\/`)
		matches := hostNameRegex.FindAllStringSubmatch(gitUrl, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("failed to extract host name from the given vcs url %v", vcsurl)
		}
		hostName := matches[0][1]
		gitRepoStruct.GitRepoPath = strings.TrimPrefix(gitUrl, "git+https://"+hostName+"/")
		gitRepoStruct.URL = strings.TrimPrefix(gitUrl, "git+")
	} else if strings.HasPrefix(gitUrl, "git+ssh") {
		hostNameRegex := regexp.MustCompile(`git\+ssh:\/\/(.*?)\/`)
		matches := hostNameRegex.FindAllStringSubmatch(gitUrl, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("failed to extract host name from the given vcs url %v", vcsurl)
		}
		hostName := matches[0][1]
		gitRepoStruct.GitRepoPath = strings.TrimPrefix(gitUrl, "git+ssh://"+hostName+"/")
		gitRepoStruct.URL = "git@" + hostName + ":" + gitRepoStruct.GitRepoPath
	} else {
		return nil, fmt.Errorf("failed to have either of the prefixes git+https or git+ssh, got %v", gitUrl)
	}

	if len(partsSplitByAt) == 2 {
		if isGitBranch(partsSplitByAt[1]) {
			gitRepoStruct.Branch = partsSplitByAt[1]
		} else if isGitCommitHash(partsSplitByAt[1]) {
			gitRepoStruct.CommitHash = partsSplitByAt[1]
		} else if isGitTag(partsSplitByAt[1]) {
			gitRepoStruct.Tag = partsSplitByAt[1]
		}
	}
	return &gitRepoStruct, nil

}

// isGitVCS checks if the given vcs url is git
func isGitVCS(vcsurl string) bool {
	// for https or ssh
	gitVCSRegex := `^git\+(https|ssh)://[a-zA-Z0-9]+([\-\.]{1}[a-zA-Z0-9]+)*\.[a-zA-Z]{2,5}(:[0-9]{1,5})?(\/.*)?$`
	matched, err := regexp.MatchString(gitVCSRegex, vcsurl)
	if err != nil {
		logrus.Fatalf("failed to match the given vcsurl %v with the git vcs regex expression %v. Error : %v", vcsurl, gitVCSRegex, err)
	}
	return matched
}

func pushGitVCS(remotePath, folderName string) error {
	if !common.IgnoreEnvironment {
		logrus.Warnf("push to remote git repositories using credentials from the environment is not yet supported.")
	}
	remotePathSplitByAt := strings.Split(remotePath, "@")
	remotePathSplitByColon := strings.Split(remotePathSplitByAt[0], ":")
	isSSH := strings.HasPrefix(remotePath, "git+ssh")
	isHTTPS := strings.HasPrefix(remotePath, "git+https")
	gitFSPath := GetClonedPath(remotePath, folderName, false)
	if (isHTTPS && len(remotePathSplitByColon) > 2) || (isSSH && len(remotePathSplitByColon) > 2) {
		gitFSPath = strings.TrimSuffix(gitFSPath, remotePathSplitByColon[len(remotePathSplitByColon)-1])
	}
	repo, err := git.PlainOpen(gitFSPath)
	if err != nil {
		return &FailedVCSPush{VCSPath: gitFSPath, Err: fmt.Errorf("failed to open the repository. Error %+v", err)}
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return &FailedVCSPush{VCSPath: gitFSPath, Err: fmt.Errorf("failed to fetch a worktree. Error %+v", err)}
	}
	_, err = worktree.Add(".")
	if err != nil {
		return &FailedVCSPush{VCSPath: gitFSPath, Err: fmt.Errorf("failed to add files to staging. Error %+v", err)}
	}

	authorName := qaengine.FetchStringAnswer(common.JoinQASubKeys(common.GitKey, "name"), "Enter git author name : ", []string{}, "", nil)
	authorEmail := qaengine.FetchStringAnswer(common.JoinQASubKeys(common.GitKey, "email"), "Enter git author email : ", []string{}, "", nil)
	commit, err := worktree.Commit("add move2kube generated output artifacts", &git.CommitOptions{
		Author: &object.Signature{
			Name:  authorName,
			Email: authorEmail,
			When:  time.Now(),
		},
	})
	logrus.Debugf("changes committed with commit hash : %+v", commit)
	if err != nil {
		return &FailedVCSPush{VCSPath: gitFSPath, Err: fmt.Errorf("failed to commit. Error : %+v", err)}
	}
	ref, err := repo.Head()
	if err != nil {
		return &FailedVCSPush{VCSPath: gitFSPath, Err: fmt.Errorf("failed to get head. Error : %+v", err)}
	}
	if isHTTPS {
		username := qaengine.FetchStringAnswer(common.JoinQASubKeys(common.GitKey, "username"), "Enter git username : ", []string{}, "", nil)
		password := qaengine.FetchPasswordAnswer(common.JoinQASubKeys(common.GitKey, "pass"), "Enter git password : ", []string{}, nil)
		err = repo.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("+%s:%s", ref.Name(), ref.Name())),
			},
			Auth: &http.BasicAuth{
				Username: username,
				Password: password,
			},
		})
		if err != nil {
			return &FailedVCSPush{VCSPath: gitFSPath, Err: fmt.Errorf("failed to push. Error : %+v", err)}
		}
	}

	if isSSH {
		authMethod, err := ssh.DefaultAuthBuilder("git")
		if err != nil {
			return fmt.Errorf("failed to get default auth builder. Error : %v", authMethod)
		}
		err = repo.Push(&git.PushOptions{
			RemoteName: "origin",
			RefSpecs: []config.RefSpec{
				config.RefSpec(fmt.Sprintf("+%s:%s", ref.Name(), ref.Name())),
			},
			Auth: authMethod,
		})
		if err != nil {
			return &FailedVCSPush{VCSPath: gitFSPath, Err: fmt.Errorf("failed to push. Error : %+v", err)}
		}
	}
	return nil
}

// Clone Clones a git repository with the given commit depth and path where to be cloned and returns final path
func (gvcsrepo *GitVCSRepo) Clone(gitCloneOptions VCSCloneOptions) (string, error) {

	if gitCloneOptions.CloneDestinationPath == "" {
		return "", fmt.Errorf("the path where the repository has to be clone is empty - %s", gitCloneOptions.CloneDestinationPath)
	}
	repoPath := filepath.Join(gitCloneOptions.CloneDestinationPath, gvcsrepo.GitRepoPath)
	_, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		logrus.Debugf("cloned output would be available at '%s'", repoPath)
	} else if gitCloneOptions.Overwrite {
		logrus.Infof("git repository might get overwritten at %s", repoPath)
		err = os.RemoveAll(repoPath)
		if err != nil {
			return "", fmt.Errorf("failed to remove the directory at the given path - %s", repoPath)
		}
	} else {
		return filepath.Join(repoPath, gvcsrepo.PathWithinRepo), nil
	}
	logrus.Infof("Cloning the repository using git into %s. This might take some time.", gitCloneOptions.CloneDestinationPath)
	if gvcsrepo.Branch != "" {
		commitDepth := 1
		if gitCloneOptions.CommitDepth != 0 {
			commitDepth = gitCloneOptions.CommitDepth
		}
		cloneOpts := git.CloneOptions{
			URL:           gvcsrepo.URL,
			Depth:         commitDepth,
			SingleBranch:  true,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", gvcsrepo.Branch)),
		}
		gvcsrepo.GitRepository, err = git.PlainClone(repoPath, false, &cloneOpts)
		if err != nil {
			logrus.Debugf("provided branch %+v does not exist in the remote, therefore creating one.", gvcsrepo.Branch)
			cloneOpts := git.CloneOptions{
				URL:   gvcsrepo.URL,
				Depth: commitDepth,
			}
			gvcsrepo.GitRepository, err = git.PlainClone(repoPath, false, &cloneOpts)
			if err != nil {
				return "", fmt.Errorf("failed to perform clone operation using git with options. Error : %+v", err)
			}
			branch := fmt.Sprintf("refs/heads/%s", gvcsrepo.Branch)
			b := plumbing.ReferenceName(branch)
			w, err := gvcsrepo.GitRepository.Worktree()
			if err != nil {
				return "", fmt.Errorf("failed return a worktree for the repostiory. Error : %+v", err)
			}

			err = w.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: b})

			if err != nil {
				err := w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b})
				if err != nil {
					return "", fmt.Errorf("failed checkout a new branch. Error : %+v", err)
				}
			}
		}
	} else if gvcsrepo.CommitHash != "" {
		commitHash := plumbing.NewHash(gvcsrepo.CommitHash)
		cloneOpts := git.CloneOptions{
			URL: gvcsrepo.URL,
		}
		gvcsrepo.GitRepository, err = git.PlainClone(repoPath, false, &cloneOpts)
		if err != nil {
			return "", fmt.Errorf("failed to perform clone operation using git with options %+v. Error : %+v", cloneOpts, err)
		}
		r, err := git.PlainOpen(repoPath)
		if err != nil {
			return "", fmt.Errorf("failed to open the git repository at the given path %+v. Error : %+v", repoPath, err)
		}
		w, err := r.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed return a worktree for the repostiory %+v. Error : %+v", r, err)
		}
		checkoutOpts := git.CheckoutOptions{
			Hash: commitHash,
		}
		err = w.Checkout(&checkoutOpts)
		if err != nil {
			return "", fmt.Errorf("failed to checkout commit hash : %s on work tree. Error : %+v", commitHash, w)
		}
	} else if gvcsrepo.Tag != "" {
		cloneOpts := git.CloneOptions{
			URL:           gvcsrepo.URL,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", gvcsrepo.Tag)),
		}
		gvcsrepo.GitRepository, err = git.PlainClone(repoPath, false, &cloneOpts)
		if err != nil {
			return "", fmt.Errorf("failed to perform clone operation using git with options %+v. Error : %+v", cloneOpts, err)
		}
	} else {
		commitDepth := 1
		cloneOpts := git.CloneOptions{
			URL:           gvcsrepo.URL,
			Depth:         commitDepth,
			SingleBranch:  true,
			ReferenceName: "refs/heads/main",
		}
		gvcsrepo.GitRepository, err = git.PlainClone(repoPath, false, &cloneOpts)
		if err != nil {
			return "", fmt.Errorf("failed to perform clone operation using git with options %+v. Error : %+v", cloneOpts, err)
		}
	}
	return filepath.Join(repoPath, gvcsrepo.PathWithinRepo), nil

}
