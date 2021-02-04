/*
Copyright IBM Corporation 2020

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package types_test

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"

	"github.com/konveyor/move2kube/internal/types"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	plantypes "github.com/konveyor/move2kube/types/plan"
	core "k8s.io/kubernetes/pkg/apis/core"
)

func TestAddVolume(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("add a volume to an empty service", func(t *testing.T) {
		// Setup
		name1 := "name1"
		v := core.Volume{Name: name1}
		s := types.Service{}
		want := types.Service{}
		want.Volumes = []core.Volume{v}

		// Test
		s.AddVolume(v)
		if !cmp.Equal(s, want) {
			t.Fatalf("Failed to add volume to an empty service properly. Difference:\n%s", cmp.Diff(want, s))
		}
	})

	t.Run("add a new volume with same name to a filled service", func(t *testing.T) {
		// Setup
		name1 := "name1"
		v := core.Volume{Name: name1}
		s := types.Service{}
		s.Volumes = []core.Volume{v}
		want := types.Service{}
		want.Volumes = []core.Volume{v}

		// Test
		s.AddVolume(v)
		if !cmp.Equal(s, want) {
			t.Fatalf("Failed to add a volume with the same name to a filled service properly. Difference:\n%s", cmp.Diff(want, s))
		}
	})
}

func TestNewContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	name1 := "name1"
	new1 := true
	c := types.NewContainer(plantypes.DockerFileContainerBuildTypeValue, name1, new1)
	if len(c.ImageNames) != 1 || c.ImageNames[0] != name1 {
		t.Fatal("Failed to initialize the container properly. Expected image names to be: [", name1, "] Actual:", c.ImageNames)
	}
	if c.New != new1 {
		t.Fatal("Failed to initialize the container properly. Expected New to be:", new1, "Actual:", c.New)
	}
	if c.NewFiles == nil {
		t.Fatal("Failed to initialize the container properly. The map NewFiles is not initialized. Actual:", c.NewFiles)
	}
}

func TestNewContainerFromImageInfo(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get container from image with tags", func(t *testing.T) {
		imginfo1 := collecttypes.NewImageInfo()
		imginfo1.Spec.Tags = []string{"tag1"}
		c := types.NewContainerFromImageInfo(imginfo1)
		if !cmp.Equal(c.ImageNames, imginfo1.Spec.Tags) {
			t.Fatalf("Failed to initialze the image names from the tags properly. Difference between image names:\n%s", cmp.Diff(imginfo1.Spec.Tags, c.ImageNames))
		}
		if !cmp.Equal(c.ExposedPorts, imginfo1.Spec.PortsToExpose) {
			t.Fatalf("Failed to initialze the ports from the image info properly. Difference between ports:\n%s", cmp.Diff(imginfo1.Spec.PortsToExpose, c.ExposedPorts))
		}
		if c.UserID != imginfo1.Spec.UserID {
			t.Fatal("The user ids are not the same. Expected:", imginfo1.Spec.UserID, "Actual:", c.UserID)
		}
		if !cmp.Equal(c.AccessedDirs, imginfo1.Spec.AccessedDirs) {
			t.Fatalf("Failed to initialze the directories from the image info properly. Difference between ports:\n%s", cmp.Diff(imginfo1.Spec.AccessedDirs, c.AccessedDirs))
		}
	})

	t.Run("get container from image without tags", func(t *testing.T) {
		imginfo1 := collecttypes.NewImageInfo()
		c := types.NewContainerFromImageInfo(imginfo1)
		if !cmp.Equal(c.ImageNames, imginfo1.Spec.Tags) {
			t.Fatalf("Failed to initialze the image names from the tags properly. Difference between image names:\n%s", cmp.Diff(imginfo1.Spec.Tags, c.ImageNames))
		}
		if !cmp.Equal(c.ExposedPorts, imginfo1.Spec.PortsToExpose) {
			t.Fatalf("Failed to initialze the ports from the image info properly. Difference between ports:\n%s", cmp.Diff(imginfo1.Spec.PortsToExpose, c.ExposedPorts))
		}
		if c.UserID != imginfo1.Spec.UserID {
			t.Fatal("The user ids are not the same. Expected:", imginfo1.Spec.UserID, "Actual:", c.UserID)
		}
		if !cmp.Equal(c.AccessedDirs, imginfo1.Spec.AccessedDirs) {
			t.Fatalf("Failed to initialze the directories from the image info properly. Difference between ports:\n%s", cmp.Diff(imginfo1.Spec.AccessedDirs, c.AccessedDirs))
		}
	})
}

func TestContainerMerge(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("merge 2 empty containers", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		c1 := types.NewContainer(buildType1, name1, new1)
		name2 := "name2"
		new2 := true
		c2 := types.NewContainer(buildType1, name2, new2)
		want := types.NewContainer(buildType1, name1, new1)

		// Test
		if c1.Merge(c2) || !reflect.DeepEqual(c1, want) { // TODO: If neither container has image name should it return true?
			t.Fatalf("Failed to merge 2 empty containers properly. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})

	t.Run("merge containers that share no image names", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}
		name2 := "name2"
		new2 := true
		c2 := types.NewContainer(buildType1, name2, new2)
		c2.ImageNames = []string{"imgname4", "imgname5", "imgname6"}
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = []string{"imgname1", "imgname2", "imgname3"}

		// Test
		if c1.Merge(c2) || !reflect.DeepEqual(c1, want) {
			t.Fatalf("Should not merge 2 containers having no image names in common. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})

	t.Run("merge containers that share some image names", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}
		name2 := "name2"
		new2 := false
		c2 := types.NewContainer(buildType1, name2, new2)
		c2.ImageNames = []string{"imgname3", "imgname4", "imgname5"}
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = []string{"imgname1", "imgname2", "imgname3", "imgname4", "imgname5"}

		// Test
		if !c1.Merge(c2) || !reflect.DeepEqual(c1, want) {
			t.Fatalf("Failed to merge 2 containers having common image names properly. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})

	t.Run("merge 2 new containers that share some image names and have the same build scripts", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		path1 := "path1"
		contents1 := "contents1"
		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}
		c1.NewFiles[path1] = contents1
		name2 := "name2"
		new2 := true
		c2 := types.NewContainer(buildType1, name2, new2)
		c2.ImageNames = []string{"imgname3", "imgname4", "imgname5"}
		c2.NewFiles[path1] = contents1
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = []string{"imgname1", "imgname2", "imgname3", "imgname4", "imgname5"}
		want.NewFiles[path1] = contents1

		// Test
		if !c1.Merge(c2) || !reflect.DeepEqual(c1, want) {
			t.Fatalf("Failed to merge the 2 containers properly. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})

	t.Run("merge 2 new containers that share some image names and having different build scripts", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		path1 := "path1"
		path2 := "path2"
		contents1 := "contents1"
		contents2 := "contents2"
		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}
		c1.NewFiles[path1] = contents1
		name2 := "name2"
		new2 := true
		c2 := types.NewContainer(buildType1, name2, new2)
		c2.ImageNames = []string{"imgname3", "imgname4", "imgname5"}
		c2.NewFiles[path2] = contents2
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = []string{"imgname1", "imgname2", "imgname3", "imgname4", "imgname5"}
		want.NewFiles[path1] = contents1
		want.NewFiles[path2] = contents2

		// Test
		if !c1.Merge(c2) || !reflect.DeepEqual(c1, want) {
			t.Fatalf("Failed to merge the 2 containers properly. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})

	t.Run("merge 2 new containers that share some image names and having different build scripts for the same key", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		path1 := "path1"
		contents1 := "contents1"
		contents2 := "contents2"
		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}
		c1.NewFiles[path1] = contents1
		name2 := "name2"
		new2 := true
		c2 := types.NewContainer(buildType1, name2, new2)
		c2.ImageNames = []string{"imgname3", "imgname4", "imgname5"}
		c2.NewFiles[path1] = contents2
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = []string{"imgname1", "imgname2", "imgname3", "imgname4", "imgname5"}
		want.NewFiles[path1] = contents1

		// Test
		if !c1.Merge(c2) || !reflect.DeepEqual(c1, want) {
			t.Fatalf("Failed to merge the 2 containers properly. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})

	t.Run("merge 2 new containers that share some image names but have different user ids", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}
		c1.UserID = 1
		name2 := "name2"
		new2 := true
		c2 := types.NewContainer(buildType1, name2, new2)
		c2.ImageNames = []string{"imgname3", "imgname4", "imgname5"}
		c2.UserID = 2
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = []string{"imgname1", "imgname2", "imgname3", "imgname4", "imgname5"}
		want.UserID = 1

		// Test
		if !c1.Merge(c2) || !reflect.DeepEqual(c1, want) {
			t.Fatalf("Failed to merge the 2 containers properly. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})

	t.Run("merge a new container into an old container that share some image names but have different user ids and build scripts", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		path1 := "path1"
		path2 := "path2"
		contents1 := "contents1"
		contents2 := "contents2"

		name1 := "name1"
		new1 := false
		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}
		c1.UserID = 1
		c1.NewFiles[path1] = contents1

		name2 := "name2"
		new2 := true
		c2 := types.NewContainer(buildType1, name2, new2)
		c2.ImageNames = []string{"imgname3", "imgname4", "imgname5"}
		c2.UserID = 2
		c2.NewFiles[path2] = contents2

		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = []string{"imgname1", "imgname2", "imgname3", "imgname4", "imgname5"}
		want.UserID = 2
		want.NewFiles[path2] = contents2

		// Test
		if !c1.Merge(c2) || !reflect.DeepEqual(c1, want) {
			t.Fatalf("Failed to merge the 2 containers properly. Difference:\n%s:", cmp.Diff(want, c1))
		}
	})
}

func TestAddFile(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("add a new script to an empty container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		path1 := "path1/foo/bar"
		contents1 := "contents1"
		c := types.NewContainer(buildType1, name1, new1)
		want := types.NewContainer(buildType1, name1, new1)
		want.NewFiles[path1] = contents1

		// Test
		c.AddFile(path1, contents1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Failed to add the new script to the empty container properly. Difference:\n%s:", cmp.Diff(want, c))
		}
	})

	t.Run("add the same script at the same path to a filled container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		path1 := "path1/foo/bar"
		contents1 := "contents1"
		c := types.NewContainer(buildType1, name1, new1)
		c.NewFiles[path1] = contents1
		want := types.NewContainer(buildType1, name1, new1)
		want.NewFiles[path1] = contents1

		// Test
		c.AddFile(path1, contents1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Adding the same script to the same path should not change the container. Difference:\n%s:", cmp.Diff(want, c))
		}
	})

	t.Run("add a different script at the same path to a filled container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		path1 := "path1/foo/bar"
		contents1 := "contents1"
		contents2 := "contents2"
		c := types.NewContainer(buildType1, name1, new1)
		c.NewFiles[path1] = contents1
		want := types.NewContainer(buildType1, name1, new1)
		want.NewFiles[path1] = contents1

		// Test
		c.AddFile(path1, contents2)
		if !cmp.Equal(c, want) {
			t.Fatalf("Adding a different script to the same path should not change the container. Difference:\n%s:", cmp.Diff(want, c))
		}
	})
}

func TestAddExposedPort(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("add a new port to expose to an empty container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		port1 := 8080
		c := types.NewContainer(buildType1, name1, new1)
		want := types.NewContainer(buildType1, name1, new1)
		want.ExposedPorts = append(want.ExposedPorts, port1)

		// Test
		c.AddExposedPort(port1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Failed to add the new port to the list of exposed ports properly. Difference:\n%s:", cmp.Diff(want, c))
		}
	})

	t.Run("add an already exposed port to a filled container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		port1 := 8080
		c := types.NewContainer(buildType1, name1, new1)
		c.ExposedPorts = append(c.ExposedPorts, port1)
		want := types.NewContainer(buildType1, name1, new1)
		want.ExposedPorts = append(want.ExposedPorts, port1)

		// Test
		c.AddExposedPort(port1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Adding an already exposed port should not change the container. Difference:\n%s:", cmp.Diff(want, c))
		}
	})
}

func TestAddImageName(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("add a new image name to an empty container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		img1 := "img1"
		c := types.NewContainer(buildType1, name1, new1)
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = append(want.ImageNames, img1)

		// Test
		c.AddImageName(img1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Failed to add an image name to an empty container properly. Difference:\n%s:", cmp.Diff(want, c))
		}
	})

	t.Run("add an existing image name to a filled container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		img1 := "img1"
		c := types.NewContainer(buildType1, name1, new1)
		c.ImageNames = append(c.ImageNames, img1)
		want := types.NewContainer(buildType1, name1, new1)
		want.ImageNames = append(want.ImageNames, img1)

		// Test
		c.AddImageName(img1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Adding an existing image name should not change the container. Difference:\n%s:", cmp.Diff(want, c))
		}
	})
}

func TestAddAccessedDirs(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("add a new directory to an empty container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		dir1 := "dir1"
		c := types.NewContainer(buildType1, name1, new1)
		want := types.NewContainer(buildType1, name1, new1)
		want.AccessedDirs = append(want.AccessedDirs, dir1)

		// Test
		c.AddAccessedDirs(dir1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Failed to add a new directory to an empty container properly. Difference:\n%s:", cmp.Diff(want, c))
		}
	})

	t.Run("add an existing directory to a filled container", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		dir1 := "dir1"
		c := types.NewContainer(buildType1, name1, new1)
		c.AccessedDirs = append(c.AccessedDirs, dir1)
		want := types.NewContainer(buildType1, name1, new1)
		want.AccessedDirs = append(want.AccessedDirs, dir1)

		// Test
		c.AddAccessedDirs(dir1)
		if !cmp.Equal(c, want) {
			t.Fatalf("Adding an existing directory should not change the container. Difference:\n%s:", cmp.Diff(want, c))
		}
	})
}

func TestNewIR(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	p := plantypes.NewPlan()
	ir := types.NewIR(p)
	if ir.Containers == nil ||
		ir.Services == nil ||
		ir.Storages == nil ||
		ir.Values.GlobalVariables == nil {
		t.Fatal("Failed to initialize the maps inside IR properly. The IR returned by NewIR:", ir)
	}
}

func TestIRMerge(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("merge 2 empty irs", func(t *testing.T) {
		// Setup
		p1 := plantypes.NewPlan()
		ir1 := types.NewIR(p1)
		p2 := plantypes.NewPlan()
		ir2 := types.NewIR(p2)
		p3 := plantypes.NewPlan()
		want := types.NewIR(p3)
		// Test
		ir1.Merge(ir2)
		if !cmp.Equal(ir1, want) {
			t.Fatalf("Failed to merge the 2 irs properly. Difference:\n%s:", cmp.Diff(want, ir1))
		}
	})

	t.Run("merge 2 irs with different names", func(t *testing.T) {
		// Setup
		name1 := "name1"
		name2 := "name2"
		p1 := plantypes.NewPlan()
		ir1 := types.NewIR(p1)
		ir1.Name = name1
		p2 := plantypes.NewPlan()
		ir2 := types.NewIR(p2)
		ir2.Name = name2
		p3 := plantypes.NewPlan()
		want := types.NewIR(p3)
		want.Name = name1
		// Test
		ir1.Merge(ir2)
		if !cmp.Equal(ir1, want) {
			t.Fatalf("Failed to merge the 2 irs properly. Difference:\n%s:", cmp.Diff(want, ir1))
		}
	})

	t.Run("merge an ir with a name into an ir with an empty name", func(t *testing.T) {
		// Setup
		name1 := "name1"
		p1 := plantypes.NewPlan()
		ir1 := types.NewIR(p1)
		ir1.Name = ""

		p2 := plantypes.NewPlan()
		ir2 := types.NewIR(p2)
		ir2.Name = name1

		p3 := plantypes.NewPlan()
		want := types.NewIR(p3)
		want.Name = name1

		// Test
		ir1.Merge(ir2)
		if !cmp.Equal(ir1, want) {
			t.Fatalf("Failed to merge the 2 irs properly. Difference:\n%s:", cmp.Diff(want, ir1))
		}
	})

	t.Run("merge 2 filled irs", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		contname1 := "contname1"
		new1 := true
		c1 := types.NewContainer(buildType1, contname1, new1)
		c1.ImageNames = []string{"imgname1", "imgname2", "imgname3"}

		s1 := types.Storage{Name: "storage1"}

		svcname1 := "svcname1"
		svc1 := types.Service{Name: svcname1, Replicas: 2}
		svc2 := types.Service{Name: svcname1, Replicas: 4}

		p1 := plantypes.NewPlan()
		ir1 := types.NewIR(p1)
		ir1.Services[svcname1] = svc1

		p2 := plantypes.NewPlan()
		ir2 := types.NewIR(p2)
		ir2.Services[svcname1] = svc2
		ir2.Containers = append(ir2.Containers, c1)
		ir2.Storages = append(ir2.Storages, s1)

		p3 := plantypes.NewPlan()
		want := types.NewIR(p3)
		want.Services[svcname1] = svc2
		want.Containers = append(want.Containers, c1)
		want.Storages = append(want.Storages, s1)

		// Test
		ir1.Merge(ir2)
		if !cmp.Equal(ir1, want) {
			t.Fatalf("Failed to merge the 2 irs properly. Difference:\n%s:", cmp.Diff(want, ir1))
		}
	})
}

func TestStorageMerge(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("merge empty storage into empty storage", func(t *testing.T) {
		s1 := types.Storage{}
		s2 := types.Storage{}
		want := types.Storage{}
		if !s1.Merge(s2) || !reflect.DeepEqual(s1, want) {
			t.Fatalf("Failed to merge 2 empty storages properly. Difference:\n%s:", cmp.Diff(want, s1))
		}
	})

	t.Run("merge storages with different names", func(t *testing.T) {
		s1 := types.Storage{}
		s1.Name = "name1"
		s2 := types.Storage{}
		s2.Name = "name2"
		want := types.Storage{}
		want.Name = "name1"
		if s1.Merge(s2) || !reflect.DeepEqual(s1, want) {
			t.Fatalf("Should not merge 2 storages with different names. Merge should return false. Difference:\n%s:", cmp.Diff(want, s1))
		}
	})

	t.Run("merge filled storage into filled storage", func(t *testing.T) {
		// Setup
		s1 := types.Storage{}
		s1.Content = map[string][]byte{"key1": []byte("val1")}
		s2 := types.Storage{}
		s2.Content = map[string][]byte{"key2": []byte("val2")}
		want := types.Storage{}
		want.Content = map[string][]byte{"key2": []byte("val2")}
		// Test
		if !s1.Merge(s2) || !reflect.DeepEqual(s1, want) {
			t.Fatalf("Failed to merge 2 filled storages properly. Difference:\n%s:", cmp.Diff(want, s1))
		}
	})
}

func TestAddContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("add a new container to an empty IR", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		c := types.NewContainer(buildType1, name1, new1)
		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)
		p2 := plantypes.NewPlan()
		want := types.NewIR(p2)
		want.Containers = append(want.Containers, c)
		// Test
		ir.AddContainer(c)
		if !cmp.Equal(ir, want) {
			t.Fatalf("Failed to add the container properly. Difference:\n%s:", cmp.Diff(want, ir))
		}
	})

	t.Run("add an existing container to a filled IR", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "name1"
		new1 := true
		c1 := types.NewContainer(buildType1, name1, new1)
		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)
		ir.Containers = append(ir.Containers, c1)

		p2 := plantypes.NewPlan()
		want := types.NewIR(p2)
		c2 := types.NewContainer(buildType1, name1, new1)
		want.Containers = append(want.Containers, c2)

		// Test
		ir.AddContainer(c1)
		if !cmp.Equal(ir, want) {
			t.Fatalf("Failed to add the container properly. Difference:\n%s:", cmp.Diff(want, ir))
		}
	})
}

func TestAddStorage(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("add a new storage to an empty IR", func(t *testing.T) {
		// Setup
		s := types.Storage{}

		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)

		p2 := plantypes.NewPlan()
		want := types.NewIR(p2)
		want.Storages = append(want.Storages, s)

		// Test
		ir.AddStorage(s)
		if !cmp.Equal(ir, want) {
			t.Fatalf("Failed to add the storage properly. Difference:\n%s:", cmp.Diff(want, ir))
		}
	})

	t.Run("add a existing storage to an filled IR", func(t *testing.T) {
		// Setup
		s := types.Storage{}

		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)
		ir.Storages = append(ir.Storages, s)

		p2 := plantypes.NewPlan()
		want := types.NewIR(p2)
		want.Storages = append(want.Storages, s)

		// Test
		ir.AddStorage(s)
		if !cmp.Equal(ir, want) {
			t.Fatalf("Failed to add the storage properly. Difference:\n%s:", cmp.Diff(want, ir))
		}
	})
}

func TestGetContainer(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get container for non existent image name from empty ir", func(t *testing.T) {
		imgname1 := "imgname1"
		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)
		if _, ok := ir.GetContainer(imgname1); ok {
			t.Fatal("Should not have found the image name", imgname1, "in an empty ir.")
		}
	})

	t.Run("get container for non existent image name from filled ir", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "contname1"
		new1 := true
		c1 := types.NewContainer(buildType1, name1, new1)
		imgname1 := "imgname1"
		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)
		ir.Containers = append(ir.Containers, c1)

		// Test
		if _, ok := ir.GetContainer(imgname1); ok {
			t.Fatal("Should not have found the non existent image name", imgname1, "in the ir.")
		}
	})

	t.Run("get container for image name from filled ir", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "contname1"
		new1 := true
		imgname1 := "imgname1"

		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = append(c1.ImageNames, imgname1)
		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)
		ir.Containers = append(ir.Containers, c1)

		// Test
		if _, ok := ir.GetContainer(imgname1); !ok {
			t.Fatal("Failed to get the container for the image name", imgname1, "in the ir.")
		}
	})

	t.Run("get container for image url from filled ir", func(t *testing.T) {
		// Setup
		buildType1 := plantypes.DockerFileContainerBuildTypeValue
		name1 := "contname1"
		new1 := true
		registry1 := "registry1.com"
		imgname1 := "imgname1"
		imgurl1 := registry1 + "/namespace/" + imgname1

		c1 := types.NewContainer(buildType1, name1, new1)
		c1.ImageNames = append(c1.ImageNames, imgname1)
		p1 := plantypes.NewPlan()
		ir := types.NewIR(p1)
		ir.Containers = append(ir.Containers, c1)
		ir.Kubernetes.RegistryURL = registry1

		// Test
		if _, ok := ir.GetContainer(imgurl1); !ok {
			t.Fatal("Failed to get the container for the image url", imgurl1, "in the ir.")
		}
	})
}
