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

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/go-git/go-git/v5/storage/filesystem"
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

var (
	// for https or ssh git repo urls
	gitVCSRegex = regexp.MustCompile(`^git\+(https|ssh)://[a-zA-Z0-9]+([\-\.]{1}[a-zA-Z0-9]+)*\.[a-zA-Z]{2,5}(:[0-9]{1,5})?(\/.*)?$`)
)

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

// isGitVCS checks if the given vcs url is a git repo url
func isGitVCS(vcsurl string) bool {
	return gitVCSRegex.MatchString(vcsurl)
}

func pushGitVCS(remotePath, folderName string, maxSize int64) error {
	if !common.IgnoreEnvironment {
		logrus.Warnf("push to remote git repositories using credentials from the environment is not yet supported.")
	}
	remotePathSplitByAt := strings.Split(remotePath, "@")
	remotePathSplitByColon := strings.Split(remotePathSplitByAt[0], ":")
	isSSH := strings.HasPrefix(remotePath, "git+ssh")
	isHTTPS := strings.HasPrefix(remotePath, "git+https")
	gitFSPath, err := GetClonedPath(remotePath, folderName, false)
	if err != nil {
		return fmt.Errorf("failed to clone the repo. Error: %w", err)
	}
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

// Clone clones a git repository with the given commit depth
// and path where it is to be cloned and returns the final path inside the repo
func (gvcsrepo *GitVCSRepo) Clone(cloneOptions VCSCloneOptions) (string, error) {
	if cloneOptions.CloneDestinationPath == "" {
		return "", fmt.Errorf("the path where the repository has to be cloned cannot be empty")
	}
	repoPath := filepath.Join(cloneOptions.CloneDestinationPath, gvcsrepo.GitRepoPath)
	repoDirInfo, err := os.Stat(repoPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to stat the git repo clone destination path '%s'. error: %w", repoPath, err)
		}
		logrus.Debugf("the cloned git repo will be available at '%s'", repoPath)
	} else {
		if !cloneOptions.Overwrite {
			if !repoDirInfo.IsDir() {
				return "", fmt.Errorf("a file already exists at the git repo clone destination path '%s'", repoPath)
			}
			logrus.Infof("Assuming that the directory at '%s' is the cloned repo", repoPath)
			return filepath.Join(repoPath, gvcsrepo.PathWithinRepo), nil
		}
		logrus.Infof("git repository clone will overwrite the files/directories at '%s'", repoPath)
		if err := os.RemoveAll(repoPath); err != nil {
			return "", fmt.Errorf("failed to remove the files/directories at '%s' . error: %w", repoPath, err)
		}
	}
	logrus.Infof("Cloning the repository using git into '%s' . This might take some time.", cloneOptions.CloneDestinationPath)

	// ------------
	var repoDirWt, dotGitDir billy.Filesystem
	repoDirWt = osfs.New(repoPath)
	dotGitDir, _ = repoDirWt.Chroot(git.GitDirName)
	fStorer := filesystem.NewStorage(dotGitDir, cache.NewObjectLRUDefault())
	limitStorer := Limit(fStorer, cloneOptions.MaxSize)
	// ------------

	commitDepth := 1
	if cloneOptions.CommitDepth != 0 {
		commitDepth = cloneOptions.CommitDepth
	}
	if gvcsrepo.Branch != "" {
		cloneOpts := git.CloneOptions{
			URL:           gvcsrepo.URL,
			Depth:         commitDepth,
			SingleBranch:  true,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", gvcsrepo.Branch)),
		}
		gvcsrepo.GitRepository, err = git.Clone(limitStorer, repoDirWt, &cloneOpts)
		if err != nil {
			logrus.Warningf("failed to clone the given branch '%s': %v . Will clone the entire repo and try again.", gvcsrepo.Branch, err)
			cloneOpts := git.CloneOptions{
				URL:   gvcsrepo.URL,
				Depth: commitDepth,
			}
			logrus.Infof("Removing previous cloned repository folder and recreating storer: %q", repoPath)

			if err := os.RemoveAll(repoPath); err != nil {
				return "", fmt.Errorf("failed to remove the files/directories at '%s' . error: %w", repoPath, err)
			}
			repoDirWt = osfs.New(repoPath)
			dotGitDir, _ = repoDirWt.Chroot(git.GitDirName)
			fStorer := filesystem.NewStorage(dotGitDir, cache.NewObjectLRUDefault())
			limitStorer := Limit(fStorer, cloneOptions.MaxSize)

			gvcsrepo.GitRepository, err = git.Clone(limitStorer, repoDirWt, &cloneOpts)
			if err != nil {
				return "", fmt.Errorf("failed to perform clone operation using git. Error: %w", err)
			}
			branch := fmt.Sprintf("refs/heads/%s", gvcsrepo.Branch)
			b := plumbing.ReferenceName(branch)
			w, err := gvcsrepo.GitRepository.Worktree()
			if err != nil {
				return "", fmt.Errorf("failed return a worktree for the repostiory. Error: %w", err)
			}
			if err := w.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: b}); err != nil {
				logrus.Warningf("failed to checkout the branch '%s', creating it...", b)
				if err := w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b}); err != nil {
					return "", fmt.Errorf("failed checkout a new branch. Error : %+v", err)
				}
			}
		}
	} else if gvcsrepo.CommitHash != "" {
		commitHash := plumbing.NewHash(gvcsrepo.CommitHash)
		cloneOpts := git.CloneOptions{
			URL: gvcsrepo.URL,
		}
		gvcsrepo.GitRepository, err = git.Clone(limitStorer, repoDirWt, &cloneOpts)
		if err != nil {
			return "", fmt.Errorf("failed to perform clone operation using git with options %+v. Error: %w", cloneOpts, err)
		}
		r, err := git.PlainOpen(repoPath)
		if err != nil {
			return "", fmt.Errorf("failed to open the git repository at the given path '%s' . Error: %w", repoPath, err)
		}
		w, err := r.Worktree()
		if err != nil {
			return "", fmt.Errorf("failed return a worktree for the repostiory %+v. Error: %w", r, err)
		}
		checkoutOpts := git.CheckoutOptions{Hash: commitHash}
		if err := w.Checkout(&checkoutOpts); err != nil {
			return "", fmt.Errorf("failed to checkout commit hash '%s' on work tree. Error: %w", commitHash, err)
		}
	} else if gvcsrepo.Tag != "" {
		cloneOpts := git.CloneOptions{
			URL:           gvcsrepo.URL,
			ReferenceName: plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", gvcsrepo.Tag)),
		}
		gvcsrepo.GitRepository, err = git.Clone(limitStorer, repoDirWt, &cloneOpts)
		if err != nil {
			return "", fmt.Errorf("failed to perform clone operation using git with options %+v. Error: %w", cloneOpts, err)
		}
	} else {
		cloneOpts := git.CloneOptions{
			URL:           gvcsrepo.URL,
			Depth:         commitDepth,
			SingleBranch:  true,
			ReferenceName: "refs/heads/main",
		}
		gvcsrepo.GitRepository, err = git.Clone(limitStorer, repoDirWt, &cloneOpts)
		if err != nil {
			return "", fmt.Errorf("failed to perform clone operation using git with options %+v and %+v. Error: %w", cloneOpts, cloneOptions, err)
		}
	}
	return filepath.Join(repoPath, gvcsrepo.PathWithinRepo), nil

}
