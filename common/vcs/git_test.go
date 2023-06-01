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
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/common"
)

func TestIsGitCommitHash(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"abcd1234ef56789012abcd1234ef56789012abcd", true},         // valid git commit hash
		{"1234ef56789012abcd1234ef56789012abcd", false},            // not valid because less than 40 characters
		{"ABCD1234EF56789012ABCD1234EF56789012ABCD", true},         // valid git commit hash
		{"ABCD1234EF56789012ABCD1234EF56789012ABCD1234dfd", false}, // not valid because more than 40 characters
		{"12345678901234567890123456789012345678901G", false},      // not valid because contains non-hexadecimal character
		{"", false}, // not valid because empty string
		{"1234abcd1234abcd1234abcd1234abcd1234abcd", true}, // valid git commit hash
		{"1234ABCD1234ABCD1234ABCD1234ABCD1234ABCD", true}, // valid git commit hash
	}

	for _, testCase := range testCases {
		result := isGitCommitHash(testCase.input)
		if result != testCase.expected {
			t.Errorf("for input %s, %v is expected but got %v.", testCase.input, testCase.expected, result)
		}
	}
}

func TestIsGitTag(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"v1", true},        // valid tag
		{"v2.3", true},      // valid tag
		{"v3.4.5", true},    // valid tag
		{"v1.0.0", true},    // valid tag
		{"v1.2.3.4", false}, // invalid tag because more than three levels
		{"v1.2.", false},    // invalid tag (trailing dot)
		{"1.2.3", false},    // invalid tag because 'v' prefix is missing
		{"v1..3", false},    // invalid tag because empty version level
	}

	for _, testCase := range testCases {
		result := isGitTag(testCase.input)
		if result != testCase.expected {
			t.Errorf("for input %s, %v is expected but got %v.", testCase.input, testCase.expected, result)
		}
	}
}

func TestGetGitRepoStruct(t *testing.T) {
	testCases := []struct {
		inputURL                 string
		expectedError            error
		expectedGitVCSRepoStruct *GitVCSRepo
	}{
		{
			inputURL:      "git+https://github.com/konveyor/move2kube.git:/samples/docker-compose",
			expectedError: nil,
			expectedGitVCSRepoStruct: &GitVCSRepo{
				InputURL:       "git+https://github.com/konveyor/move2kube.git:/samples/docker-compose",
				PathWithinRepo: "/samples/docker-compose",
				GitRepoPath:    "konveyor/move2kube.git",
				URL:            "https://github.com/konveyor/move2kube.git",
				Branch:         "",
				CommitHash:     "",
				Tag:            "",
			},
		},
	}

	for _, testCase := range testCases {
		gitVCSRepoStruct, err := getGitRepoStruct(testCase.inputURL)
		if err != nil {
			if testCase.expectedError == nil {
				t.Errorf("Input URL: %s, Expected no error, Got error: %v", testCase.inputURL, err)
			} else if err.Error() != testCase.expectedError.Error() {
				t.Errorf("Input URL: %s, Expected error: %v, Got error: %v", testCase.inputURL, testCase.expectedError, err)
			}
		} else {
			if testCase.expectedError != nil {
				t.Errorf("Input URL: %s, Expected error: %v, Got no error", testCase.inputURL, testCase.expectedError)
			} else if diff := cmp.Diff(gitVCSRepoStruct, testCase.expectedGitVCSRepoStruct); diff != "" {
				t.Errorf("Input URL: %s, Expected GitRepo: %+v, Got GitRepo: %+v, Diff: %s", testCase.inputURL, testCase.expectedGitVCSRepoStruct, gitVCSRepoStruct, diff)
			}
		}
	}
}

func TestIsGitVCS(t *testing.T) {
	validURL := "git+https://github.com/konveyor/move2kube.git:/samples/docker-compose"
	invalidURL := "https://github.com/konveyor/move2kube.git:/samples/docker-compose"

	validGitVCS := isGitVCS(validURL)
	if !validGitVCS {
		t.Errorf("Expected %v to be a valid Git VCS URL, but it was not.", validURL)
	}

	invalidGitVCS := isGitVCS(invalidURL)
	if invalidGitVCS {
		t.Errorf("Expected %v to be an invalid Git VCS URL, but it was not.", invalidURL)
	}
}

func TestClone(t *testing.T) {
	// Test case - clone a valid vcs url with overwrite true
	gitURL := "git+https://github.com/konveyor/move2kube.git"
	repo, err := getGitRepoStruct(gitURL)
	if err != nil {
		t.Errorf("failed to get git repo struct for the given git URL %s. Error : %+v", gitURL, err)
	}
	overwrite := true
	tempPath, err := filepath.Abs(common.RemoteTempPath)
	if err != nil {
		t.Errorf("failed to get absolute path of %s. Error : %+v", common.RemoteTempPath, err)
	}
	folderName := "test-clone"
	cloneOpts := VCSCloneOptions{CommitDepth: 1, Overwrite: overwrite, CloneDestinationPath: filepath.Join(tempPath, folderName)}
	clonedPath, err := repo.Clone(cloneOpts)
	if err != nil {
		t.Errorf("failed to clone the git repo. Error : %+v", err)
	}

	// Test case 2 - Repository already exists with overwrite true
	gitURL = "git+https://github.com/konveyor/move2kube.git"
	repo, err = getGitRepoStruct(gitURL)
	if err != nil {
		t.Errorf("failed to get git repo struct for the given git URL %s. Error : %+v", gitURL, err)
	}
	overwrite = false
	tempPath, err = filepath.Abs(common.RemoteTempPath)
	if err != nil {
		t.Errorf("failed to get absolute path of %s. Error : %+v", common.RemoteTempPath, err)
	}
	folderName = "test-clone"
	cloneOpts = VCSCloneOptions{CommitDepth: 1, Overwrite: overwrite, CloneDestinationPath: filepath.Join(tempPath, folderName)}
	clonedPathWithoutOverwrite, err := repo.Clone(cloneOpts)
	if err != nil {
		t.Errorf("failed to clone the git repo. Error : %+v", err)
	}
	if clonedPath != clonedPathWithoutOverwrite {
		t.Errorf("cloned paths did not match with overwrite false. cloned path %s, cloned path without overwrite: %s", clonedPath, clonedPathWithoutOverwrite)
	}

}

func TestIsGitBranch(t *testing.T) {
	testCases := []struct {
		branchName string
		expected   bool
	}{
		{
			branchName: "main",
			expected:   true,
		},
		{
			branchName: "123branchName",
			expected:   false,
		},
		{
			branchName: "feature/branchName#1",
			expected:   false,
		},
		{
			branchName: "feature/branchName_1",
			expected:   true,
		},
		{
			branchName: "",
			expected:   false,
		},
		{
			branchName: "$develop",
			expected:   false,
		},
	}

	for _, testCase := range testCases {
		isValidBranchName := isGitBranch(testCase.branchName)
		if isValidBranchName != testCase.expected {
			t.Errorf("failed branch %s isValid = %t, but got = %t\n", testCase.branchName, testCase.expected, isValidBranchName)
		}
	}

}
