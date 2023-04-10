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

package environment

import (
	"net"
	"os"
	"testing"

	"github.com/konveyor/move2kube/qaengine"
	environmenttypes "github.com/konveyor/move2kube/types/environment"
)

func TestPeerContainer(t *testing.T) {

	t.Run("Test for New Peer Container", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image: "quay.io/konveyor/hello-world:latest",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}
		defer container.Destroy()
	})

	t.Run("Test for Reset", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image: "quay.io/konveyor/hello-world:latest",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}

		oldContainer, ok := container.(*PeerContainer)
		if !ok {
			t.Fatalf("Failed to cast into PeerContainer")
		}

		oldContainerID := oldContainer.ContainerInfo.ID

		err = container.Reset()
		if err != nil {
			t.Fatalf("Error resetting builder: %s", err)
		}

		newContainerID := oldContainer.ContainerInfo.ID

		if oldContainerID == newContainerID {
			t.Fatalf("Expected builder to be available after reset, but it was not")
		}
	})

	t.Run("Test for Stat", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		testStatFile := "sys"

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image: "registry.access.redhat.com/ubi8-minimal:8.7-1107",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}
		defer container.Destroy()

		stats, err := container.Stat(testStatFile)
		if err != nil {
			t.Fatalf("Error getting file stats: %s", err)
		}

		t.Logf("stats = %+v", stats)

		if stats.Name() != testStatFile {
			t.Fatalf("Failed to Stat expected: %+v, got: %+v", testStatFile, stats.Name())
		}
	})

	t.Run("Test for  Exec", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image:            "registry.access.redhat.com/ubi8-minimal:8.7-1107",
			KeepAliveCommand: []string{"sleep", "infinity"},
			WorkingDir:       "/",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}
		defer container.Destroy()

		stdout, stderr, exitcode, err := container.Exec(environmenttypes.Command{"echo", "-n", "Hello World"}, []string{})
		if stderr != "" || err != nil || exitcode != 0 {
			t.Fatalf("Error executing command stderr: %v, exitcode: %v, err: %v", stderr, exitcode, err)
		}

		if stdout != "Hello World" {
			t.Errorf("Expected output to be 'hello world\\n', but got '%s'", stdout)
		}
	})
	t.Run("Test for Destroy", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		testStatFile := "sys"

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image:            "registry.access.redhat.com/ubi8-minimal:8.7-1107",
			KeepAliveCommand: []string{"sleep", "infinity"},
			WorkingDir:       "/",
		}
		containerEnv, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}

		if err := containerEnv.Destroy(); err != nil {
			t.Fatalf("Failed to destroy environment: %s", err)
		}

		_, err = containerEnv.Stat(testStatFile)
		if err == nil {
			t.Fatalf("Failed to Destroy Container")
		}

	})

	t.Run("Test for Download", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image:            "registry.access.redhat.com/ubi8-minimal:8.7-1107",
			KeepAliveCommand: []string{"sleep", "infinity"},
			WorkingDir:       "/",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}

		filename := "/tmp"
		outpath, err := container.Download(filename)
		if err != nil {
			t.Fatalf("Failed to download file %s from environment: %s", filename, err)
		}

		if _, err := os.Stat(outpath); os.IsNotExist(err) {
			t.Fatalf("File %s was not downloaded from environment", outpath)
		}

		if err := os.RemoveAll(outpath); err != nil {
			t.Fatalf("Failed to remove downloaded file %s: %s", outpath, err)
		}
	})

	t.Run("Test for Upload", func(t *testing.T) {
		// Initialize the environment
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image:            "registry.access.redhat.com/ubi8-minimal:8.7-1107",
			KeepAliveCommand: []string{"sleep", "infinity"},
			WorkingDir:       "/",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}

		filename := "container/testdata/buildimage"

		envPath, err := container.Upload(filename)
		if err != nil {
			t.Fatalf("Failed to upload file %s to environment: %s", filename, err)
		}

		if _, err := container.Stat(envPath); err != nil {
			t.Fatalf("File %s was not uploaded to environment: %s", envPath, err)
		}
	})

	t.Run("Test for GetContext", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		context := "/"

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image:            "registry.access.redhat.com/ubi8-minimal:8.7-1107",
			KeepAliveCommand: []string{"sleep", "infinity"},
			WorkingDir:       "/",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}

		ctx := container.GetContext()

		if ctx != context {
			t.Fatalf("Incorrect context recieved, expected: %+v, got: %+v", context, ctx)
		}
	})

	t.Run("Test for GetSource", func(t *testing.T) {
		e := qaengine.NewDefaultEngine()
		qaengine.AddEngine(e)

		source := "/workspace"

		envInfo := EnvInfo{
			Name:        "test",
			ProjectName: "test",
			Source:      "/test/source",
		}
		grpcQAReceiver := &net.TCPAddr{}
		containerInfo := environmenttypes.Container{
			Image:            "registry.access.redhat.com/ubi8-minimal:8.7-1107",
			KeepAliveCommand: []string{"sleep", "infinity"},
			WorkingDir:       "/",
		}
		container, err := NewPeerContainer(envInfo, grpcQAReceiver, containerInfo, true)
		if err != nil {
			t.Fatalf("Error creating new peer container: %s", err)
		}

		src := container.GetSource()

		if src != source {
			t.Fatal("Failed to get source of environment")
		}
	})

}
