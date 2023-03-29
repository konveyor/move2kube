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

package container

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/docker/docker/api/types"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
	"github.com/sirupsen/logrus"
)

func verifystats(textfile1 string, textfile2 string, textfile1Stat []string, textfile2Stat []string) bool {

	statsmatched := true
	info1, err := os.Stat(textfile1)
	if err != nil {
		fmt.Println("Error:", err)
	}

	info2, err := os.Stat(textfile2)
	if err != nil {
		fmt.Println("Error:", err)
	}

	perm1 := info1.Mode().Perm().String()
	perm2 := info2.Mode().Perm().String()

	uid1 := info1.Sys().(*syscall.Stat_t).Uid
	uid2 := info2.Sys().(*syscall.Stat_t).Uid
	uid1str := strconv.FormatUint(uint64(uid1), 10)
	uid2str := strconv.FormatUint(uint64(uid2), 10)

	gid1 := info1.Sys().(*syscall.Stat_t).Gid
	gid2 := info2.Sys().(*syscall.Stat_t).Gid
	gid1str := strconv.FormatUint(uint64(gid1), 10)
	gid2str := strconv.FormatUint(uint64(gid2), 10)

	if textfile1Stat[0] != perm1 || textfile1Stat[1] != uid1str || textfile1Stat[2] != gid1str {
		statsmatched = false
	}

	if textfile2Stat[0] != perm2 || textfile2Stat[1] != uid2str || textfile2Stat[2] != gid2str {
		statsmatched = false
	}
	return statsmatched

}

func TestIsBuilderAvailable(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	t.Run("normal use case", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "quay.io/konveyor/move2kube-api"

		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}
	})

	t.Run("normal use case where we get result from cache", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "quay.io/konveyor/move2kube-api"

		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}
		if !provider.availableImages[image] {
			t.Fatalf("Failed to add the image %q to the list of available images", image)
		}
		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}
	})

	t.Run("check for a non existent image", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "this/doesnotexist:foobar"
		if err := provider.pullImage(image); err == nil {
			t.Fatalf("Should not have succeeded. The image '%s' does not exist", image)
		}
	})

	t.Run("check for creating a container ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"

		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}

		containerID, err := provider.CreateContainer(environmenttypes.Container{Image: image})
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", image, err)
		}
		t.Logf("created container with container id - %s", containerID)
		err = provider.StopAndRemoveContainer(containerID)
		if err != nil {
			t.Fatalf("failed to stop and remove container with base image '%s' . Error: %q", image, err)
		}
	})

	t.Run("Check for running a cmd in a container ", func(t *testing.T) {
		testStdout := "/root"
		testStderr := ""
		testExitCode := 0
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"
		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}

		containerID, err := provider.CreateContainer(environmenttypes.Container{Image: image, KeepAliveCommand: []string{"sleep", "infinity"}})
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", image, err)
		}
		cmd := environmenttypes.Command{"pwd"}
		stdout, stderr, exitCode, err := provider.RunCmdInContainer(containerID, cmd, testStdout, []string{})
		if err != nil {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		if strings.TrimSpace(stdout) != testStdout || strings.TrimSpace(stderr) != testStderr || exitCode != testExitCode {
			t.Fatalf("Output mismatch need %v %v %v, got %v %v %v ", testStdout, testStderr, testExitCode, stdout, stderr, exitCode)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)
		err = provider.StopAndRemoveContainer(containerID)
		if err != nil {
			t.Fatalf("failed to stop and remove container with base image '%s' . Error: %q", image, err)
		}
	})

	t.Run("Check for InspectImage functionality ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"
		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}
		outputInspect, err := provider.InspectImage(image)
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", image, err)
		}

		found := false
		for _, i := range outputInspect.RepoTags {
			if strings.HasSuffix(i, "alpine:latest") {
				found = true
			}
		}
		if !found {
			t.Fatalf("Ispect Repo Tag Mismatch Should be - alpine:latest got - %v", outputInspect.RepoTags)
		}

	})

	t.Run("check for stopping and removing a container ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"

		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}

		containerID, err := provider.CreateContainer(environmenttypes.Container{Image: image})
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", image, err)
		}
		t.Logf("created container with container id - %s", containerID)

		err = provider.StopAndRemoveContainer(containerID)
		if err != nil {
			t.Fatalf("failed to stop and remove container with base image '%s' . Error: %q", image, err)
		}
		containers, err := provider.cli.ContainerList(provider.ctx, types.ContainerListOptions{})
		if err != nil {
			t.Fatalf("failed to list containers . Error: %q", err)
		}
		for _, container := range containers {
			if container.ID == containerID {
				t.Fatalf("failed to Stop and Remove operation failed container %s exists! \n", containerID)
				return
			}
		}
	})

	t.Run("check for coping directories to an image ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"

		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}

		image = "alpine:latest"
		newimage := "alpine:with_dirs"
		paths := make(map[string]string)
		absDir1, err := filepath.Abs("./testdata/dirstocopy/dir1")
		if err == nil {
			fmt.Println("Absolute path is:", absDir1)
		}
		absDir2, err := filepath.Abs("./testdata/dirstocopy/dir2")
		if err == nil {
			fmt.Println("Absolute path is:", absDir2)
		}
		paths[absDir1] = "/dir1"
		paths[absDir2] = "/dir2"

		err = provider.CopyDirsIntoImage(image, newimage, paths)
		if err != nil {
			t.Fatalf("failed to copy dirs into iamge '%s' . Error: %q", image, err)
		}

		containerID, err := provider.CreateContainer(environmenttypes.Container{Image: newimage, KeepAliveCommand: []string{"sleep", "infinity"}})
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", newimage, err)
		}
		t.Logf("created container with container id - %s", containerID)

		cmd := environmenttypes.Command{"stat", "-c", "'%A %u %g'", "dir1/test1.txt"}
		stdout, stderr, exitCode, err := provider.RunCmdInContainer(containerID, cmd, "/", []string{})
		if err != nil || stderr != "" || exitCode != 0 {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)
		outfromcont := stdout[1 : len(stdout)-2]
		textfile1Stat := strings.Split(outfromcont, " ")

		cmd = environmenttypes.Command{"stat", "-c", "'%A %u %g'", "dir2/test2.txt"}
		stdout, stderr, exitCode, err = provider.RunCmdInContainer(containerID, cmd, "/", []string{})
		if err != nil || stderr != "" || exitCode != 0 {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)
		outfromcont = stdout[1 : len(stdout)-2]
		textfile2Stat := strings.Split(outfromcont, " ")

		textfile1 := filepath.Join(absDir1, "test1.txt")
		textfile2 := filepath.Join(absDir2, "test2.txt")

		statsmatched := verifystats(textfile1, textfile2, textfile1Stat, textfile2Stat)

		if !statsmatched {
			t.Fatalf("Text file stats mismatch test failed file 1 stats: %+v file 2 stats: %+v", textfile1Stat, textfile2Stat)
		}

		err = provider.StopAndRemoveContainer(containerID)
		if err != nil {
			t.Fatalf("failed to stop and remove container with base image '%s' . Error: %q", image, err)
		}

	})
	t.Run("check for coping directories to a container ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"

		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}

		containerID, err := provider.CreateContainer(environmenttypes.Container{Image: image, KeepAliveCommand: []string{"sleep", "infinity"}})
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", image, err)
		}
		t.Logf("created container with container id - %s", containerID)

		paths := make(map[string]string)
		absDir1, err := filepath.Abs("./testdata/dirstocopy/dir1")
		if err == nil {
			t.Log("Absolute path is:", absDir1)
		}
		absDir2, err := filepath.Abs("./testdata/dirstocopy/dir2")
		if err == nil {
			t.Log("Absolute path is:", absDir2)
		}
		paths[absDir1] = "/dir1"
		paths[absDir2] = "/dir2"

		err = provider.CopyDirsIntoContainer(containerID, paths)
		if err != nil {
			t.Fatalf("failed to copy dirs into iamge '%s' . Error: %q", image, err)
		}

		cmd := environmenttypes.Command{"stat", "-c", "'%A %u %g'", "dir1/test1.txt"}

		stdout, stderr, exitCode, err := provider.RunCmdInContainer(containerID, cmd, "/", []string{})
		if err != nil || stderr != "" || exitCode != 0 {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)
		outfromcont := stdout[1 : len(stdout)-2]

		textfile1Stat := strings.Split(outfromcont, " ")

		cmd = environmenttypes.Command{"stat", "-c", "'%A %u %g'", "dir2/test2.txt"}
		stdout, stderr, exitCode, err = provider.RunCmdInContainer(containerID, cmd, "/", []string{})
		if err != nil || stderr != "" || exitCode != 0 {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)

		outfromcont = stdout[1 : len(stdout)-2]

		textfile2Stat := strings.Split(outfromcont, " ")

		textfile1 := filepath.Join(absDir1, "test1.txt")
		textfile2 := filepath.Join(absDir2, "test2.txt")

		statsmatched := verifystats(textfile1, textfile2, textfile1Stat, textfile2Stat)

		if !statsmatched {
			t.Fatalf("Text file stats mismatch test failed file 1 stats: %+v file 2 stats: %+v", textfile1Stat, textfile2Stat)
		}

		err = provider.StopAndRemoveContainer(containerID)
		if err != nil {
			t.Fatalf("failed to stop and remove container with base image '%s' . Error: %q", image, err)
		}
	})

	t.Run("check for coping directories from a container ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"

		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}

		containerID, err := provider.CreateContainer(environmenttypes.Container{Image: image, KeepAliveCommand: []string{"sleep", "infinity"}})
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", image, err)
		}
		t.Logf("created container with container id - %s", containerID)

		paths := make(map[string]string)
		absDir1, err := filepath.Abs("./testdata/dirstocopy/dir1")
		if err == nil {
			fmt.Println("Absolute path is:", absDir1)
		}
		absDir2, err := filepath.Abs("./testdata/dirstocopy/dir2")
		if err == nil {
			fmt.Println("Absolute path is:", absDir2)
		}

		paths[absDir1] = "/dir1"
		paths[absDir2] = "/dir2"

		err = provider.CopyDirsIntoContainer(containerID, paths)
		if err != nil {
			t.Fatalf("failed to copy dirs into iamge '%s' . Error: %q", image, err)
		}

		for k := range paths {
			delete(paths, k)
		}

		tempDir, err := ioutil.TempDir("", "mytempdir")
		if err != nil {
			t.Fatal("Error creating temp directory:", err)
		}
		defer os.RemoveAll(tempDir)

		paths["/dir1"] = tempDir
		paths["/dir2"] = tempDir

		err = provider.CopyDirsFromContainer(containerID, paths)
		if err != nil {
			t.Fatal("failed to copy Directories from Container ", err)
		}

		cmd := environmenttypes.Command{"stat", "-c", "'%A %u %g'", "dir1/test1.txt"}

		stdout, stderr, exitCode, err := provider.RunCmdInContainer(containerID, cmd, "/", []string{})
		if err != nil || stderr != "" || exitCode != 0 {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)
		outfromcont := stdout[1 : len(stdout)-2]

		textfile1Stat := strings.Split(outfromcont, " ")

		cmd = environmenttypes.Command{"stat", "-c", "'%A %u %g'", "dir2/test2.txt"}
		stdout, stderr, exitCode, err = provider.RunCmdInContainer(containerID, cmd, "/", []string{})
		if err != nil || stderr != "" || exitCode != 0 {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)

		outfromcont = stdout[1 : len(stdout)-2]

		textfile2Stat := strings.Split(outfromcont, " ")

		file1FromCont := filepath.Join(tempDir, "test1.txt")
		file2FromCont := filepath.Join(tempDir, "test2.txt")

		statsmatched := verifystats(file1FromCont, file2FromCont, textfile1Stat, textfile2Stat)

		if !statsmatched {
			t.Fatalf("Text file stats mismatch test failed file 1 stats: %+v file 2 stats: %+v", textfile1Stat, textfile2Stat)
		}

		err = provider.StopAndRemoveContainer(containerID)
		if err != nil {
			t.Fatalf("failed to stop and remove container with base image '%s' . Error: %q", image, err)
		}
	})

	t.Run("check for Stat of a file ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/alpine:latest"
		if err := provider.pullImage(image); err != nil {
			t.Fatalf("Failed to find the image '%s' locally and/or pull it. Error: %q", image, err)
		}

		containerID, err := provider.CreateContainer(environmenttypes.Container{Image: image, KeepAliveCommand: []string{"sleep", "infinity"}})
		if err != nil {
			t.Fatalf("failed to create container with base image '%s' . Error: %q", image, err)
		}
		t.Logf("created container with container id - %s", containerID)

		cmd := environmenttypes.Command{"touch", "test.txt"}
		stdout, stderr, exitCode, err := provider.RunCmdInContainer(containerID, cmd, "/", []string{})
		if err != nil || stderr != "" || exitCode != 0 {
			t.Fatalf("failed to run the command in the container ID %s . Error: %q", containerID, err)
		}
		t.Logf("stdout = %v, stderr=%v, exitCode=%v", stdout, stderr, exitCode)

		statOut, err := provider.Stat(containerID, "test.txt")
		t.Logf("statOut =  %+v", statOut)

		if statOut.Name() != "test.txt" {
			t.Fatalf("Stat command failed")
		}
		err = provider.StopAndRemoveContainer(containerID)
		if err != nil {
			t.Fatalf("failed to stop and remove container with base image '%s' . Error: %q", image, err)
		}
	})

	t.Run("check for build image ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "alpine:withfile"
		context, err := filepath.Abs("./testdata/buildimage")
		if err != nil {
			t.Fatal("context path is:", context)
		}
		err = provider.BuildImage(image, context, "dockerfile")
		if err != nil {
			t.Fatalf("failed to build image, . ERROR - %q:", err)
		}
		if !provider.availableImages[image] {
			t.Fatalf("Failed to add the image %q to the list of available images", image)
		}
	})

	t.Run("check for remove image ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "alpine123:withfile"
		context, err := filepath.Abs("./testdata/buildimage")
		if err != nil {
			t.Fatal("context path is:", context)
		}
		err = provider.BuildImage(image, context, "dockerfile")
		if err != nil {
			t.Fatalf("failed to build image, . ERROR - %q:", err)
		}
		if !provider.availableImages[image] {
			t.Fatalf("Failed to add the image %q to the list of available images", image)
		}

		err = provider.RemoveImage(image)
		if err != nil {
			t.Fatalf("failed to remove image, . ERROR - %q:", err)
		}
		err = provider.updateAvailableImages()
		if err != nil {
			t.Fatalf("failed to update images, . ERROR - %q:", err)
		}
		if provider.availableImages[image] {
			t.Fatalf("Failed to remove the image %q from the list of available images %+v", image, provider.availableImages)
		}
	})

	t.Run("check for Running a Container ", func(t *testing.T) {
		provider, _ := newDockerEngine()
		image := "docker.io/somerandomimage"
		output, containerStarted, err := provider.RunContainer(image, environmenttypes.Command{"pwd"}, "", "")
		if err == nil {
			t.Fatalf("Test passed: failed to run the image due to incorrect name '%s' as a container. Error: %q", image, err)
		}
		t.Logf("output = %+v containerStarted = %+v", output, containerStarted)
	})

}
