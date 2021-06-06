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

package translator_test

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/containerizer"
	"github.com/konveyor/move2kube/internal/source"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
)

func TestGetServiceOptions(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get services with a non existent directory and empty plan", func(t *testing.T) {
		// Setup
		inputpath := "/this/does/not/exit/foobar/"
		translator := source.Any2KubeTranslator{}
		plan := plantypes.NewPlan()
		want := []plantypes.Service{}
		containerizer.InitContainerizers(inputpath, nil)

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to get the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("get services with empty directory and empty plan", func(t *testing.T) {
		// Setup
		inputpath := t.TempDir()
		translator := source.Any2KubeTranslator{}
		plan := plantypes.NewPlan()
		want := []plantypes.Service{}
		containerizer.InitContainerizers(inputpath, nil)

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to get the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("get services when the directory contains files and directories we can't read", func(t *testing.T) {
		// Setup
		inputpath := t.TempDir()
		subdirpath := filepath.Join(inputpath, "nopermstoread")
		if err := os.Mkdir(subdirpath, 0); err != nil {
			t.Fatal("Failed to create a temporary directory for testing at path", subdirpath, "Error:", err)
		}
		ignorefilepath := filepath.Join(inputpath, common.IgnoreFilename)
		if err := ioutil.WriteFile(ignorefilepath, []byte("foo/"), 0); err != nil {
			t.Fatal("Failed to create a temporary file for testing at path", ignorefilepath, "Error:", err)
		}
		translator := source.Any2KubeTranslator{}
		plan := plantypes.NewPlan()
		want := []plantypes.Service{}
		containerizer.InitContainerizers(inputpath, nil)

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to get the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("get services from a simple nodejs app and empty plan", func(t *testing.T) {
		// Setup
		relInputPath := "../../samples/nodejs"
		inputPath, err := filepath.Abs(relInputPath)
		if err != nil {
			t.Fatalf("Failed to make the input path %q absolute. Error: %q", relInputPath, err)
		}
		translator := source.Any2KubeTranslator{}
		containerizer.InitContainerizers(inputPath, nil)

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		if err := plan.SetRootDir(inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}

		wantPlan, err := plantypes.ReadPlan("testdata/expectedservicesfornodejsapp.yaml")
		if err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}
		want := wantPlan.Spec.Inputs.Services["nodejs"]

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		// Don't compare RepoInfo
		for i := range services {
			services[i].RepoInfo = plantypes.RepoInfo{}
		}

		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to create the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("get services from a simple nodejs app and filled plan", func(t *testing.T) {
		// Setup
		relInputPath := "../../samples/nodejs"
		inputPath, err := filepath.Abs(relInputPath)
		if err != nil {
			t.Fatalf("Failed to make the input path %q absolute. Error: %q", relInputPath, err)
		}
		translator := source.Any2KubeTranslator{}
		containerizer.InitContainerizers(inputPath, nil)

		// services
		svc1 := plantypes.NewService("svc1", "Any2Kube")
		svc1.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = []string{"foo/"}
		svc2 := plantypes.NewService("svc2", "Any2Kube")
		svc2.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = []string{"bar/"}

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		if err := plan.SetRootDir(inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}
		plan.Spec.Inputs.Services = map[string][]plantypes.Service{
			"svc1": {svc1},
			"svc2": {svc2},
		}

		wantPlan, err := plantypes.ReadPlan("testdata/expectedservicesfornodejsapp.yaml")
		if err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}
		want := wantPlan.Spec.Inputs.Services["nodejs"]

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		// Don't compare RepoInfo
		for i := range services {
			services[i].RepoInfo = plantypes.RepoInfo{}
		}

		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to create the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("get services from a simple nodejs app that we already containerized", func(t *testing.T) {
		// Setup
		relInputPath := "../../samples/nodejs"
		inputPath, err := filepath.Abs(relInputPath)
		if err != nil {
			t.Fatalf("Failed to make the input path %q absolute. Error: %q", relInputPath, err)
		}
		translator := source.Any2KubeTranslator{}
		containerizer.InitContainerizers(inputPath, nil)

		// services
		svc1 := plantypes.NewService("svc1", "Any2Kube")
		svc1.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = []string{inputPath}

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		if err := plan.SetRootDir(inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}
		plan.Spec.Inputs.Services = map[string][]plantypes.Service{
			"svc1": {svc1},
		}

		want := []plantypes.Service{}

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)

		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to create the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("test m2kignore can ignore a directory but include its subdirectories", func(t *testing.T) {
		// 1. Ignore a directory, but include all subdirectories

		// Setup
		relInputPath := "testdata/nodejsappwithm2kignorecase1"
		inputPath, err := filepath.Abs(relInputPath)
		if err != nil {
			t.Fatalf("Failed to make the input path %q absolute. Error: %q", relInputPath, err)
		}
		translator := source.Any2KubeTranslator{}
		containerizer.InitContainerizers(inputPath, nil)

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		if err := plan.SetRootDir(inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}

		wantPlan, err := plantypes.ReadPlan("testdata/expectedservicesfornodejsappwithm2kignorecase1.yaml")
		if err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}
		want := wantPlan.Spec.Inputs.Services["includeme"]

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		// Don't compare RepoInfo
		for i := range services {
			services[i].RepoInfo = plantypes.RepoInfo{}
		}

		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to create the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("test m2kignore can be used to ignore everything but a very specific subdirectory", func(t *testing.T) {
		// Setup
		relInputPath := "testdata/javamavenappwithm2kignorecase2"
		inputPath, err := filepath.Abs(relInputPath)
		if err != nil {
			t.Fatalf("Failed to make the input path %q absolute. Error: %q", relInputPath, err)
		}
		translator := source.Any2KubeTranslator{}
		containerizer.InitContainerizers(inputPath, nil)

		plan := plantypes.NewPlan()
		plan.Name = "java-maven-app"
		if err := plan.SetRootDir(inputPath); err != nil {
			t.Fatalf("Failed to set the root directory of the plan to path %q Error: %q", inputPath, err)
		}

		wantPlan, err := plantypes.ReadPlan("testdata/expectedservicesforjavamavenappwithm2kignorecase2.yaml")
		if err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}
		want := wantPlan.Spec.Inputs.Services["java-maven"]

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		// Don't compare RepoInfo
		for i := range services {
			services[i].RepoInfo = plantypes.RepoInfo{}
		}

		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to create the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("test m2kignore can include a directory but ignore all subdirectories", func(t *testing.T) {
		// 2. Include a directory, ignore all subdirectories or a subset of subdirectories
		// TODO: Note that while m2kignore might work as expected, the buildpacks do not.
		// The CNB buildpacks when run on a directory will ALWAYS look inside all of its subdirectories as well.

		// Setup
		// We create the following directory structure:
		// .
		inputpath := t.TempDir()
		// ./includeme/
		// ./includeme/excludeme/
		subdirpath := filepath.Join(inputpath, "includeme")
		subsubdirpath := filepath.Join(subdirpath, "excludeme")
		if err := os.MkdirAll(subsubdirpath, common.DefaultDirectoryPermission); err != nil {
			t.Fatal("Failed to create a temporary directory for testing at path", subsubdirpath, "Error:", err)
		}
		// .m2kignore
		testdatapath := "testdata/m2kignoreforignorecontents"
		ignorerules, err := ioutil.ReadFile(testdatapath)
		if err != nil {
			t.Fatal("Failed to read the testdata at", testdatapath, "Error:", err)
		}
		ignorefilepath := filepath.Join(inputpath, common.IgnoreFilename)
		if err := ioutil.WriteFile(ignorefilepath, ignorerules, common.DefaultFilePermission); err != nil {
			t.Fatal("Failed to create a temporary file for testing at path", ignorefilepath, "Error:", err)
		}
		// ./includeme/excludeme/index.php
		fpath := filepath.Join(subsubdirpath, "package.json")
		if err := ioutil.WriteFile(fpath, []byte("this is ' invalid json"), common.DefaultFilePermission); err != nil {
			t.Fatal("Failed to create a temporary file for testing at path", fpath, "Error:", err)
		}

		translator := source.Any2KubeTranslator{}
		plan := plantypes.NewPlan()
		want := []plantypes.Service{}
		containerizer.InitContainerizers(inputpath, nil)

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to get the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})

	t.Run("test multiple hierarchical m2kignores", func(t *testing.T) {
		// TODO: Note that while m2kignore might work as expected, the buildpacks do not.
		// The CNB buildpacks when run on a directory will ALWAYS look inside all of its subdirectories as well.
		// The behaviour of the CNB buildpacks makes it virtually impossible to test this scenario.
		// This test can really only be checked through vscode debugging to make sure the correct directories are being ignored.

		// Setup
		// We create the following directory structure:
		/*
			.
			├── .m2kignore
			├── a
			│   └── a
			│       ├── .m2kignore
			│       ├── a
			│       │   ├── a
			│       │   ├── b
			│       │   ├── c
			│       │   └── d
			│       └── b
			│           ├── a
			│           └── b
			├── b
			│   ├── .m2kignore
			│   └── a
			│       ├── a
			│       └── b
			└── c
			    └── a
		*/
		tempdir := t.TempDir()
		testdatapath := "testdata/testmultiplem2kignores.tar"

		tarbytes, err := ioutil.ReadFile(testdatapath)
		if err != nil {
			t.Fatalf("Failed to read the test data at path %q Error: %q", testdatapath, err)
		}

		tarstring := base64.StdEncoding.EncodeToString(tarbytes)
		err = common.UnTarString(tarstring, tempdir)
		if err != nil {
			t.Fatalf("Failed to untar the test data into path %q Error: %q", tempdir, err)
		}

		inputpath := filepath.Join(tempdir, "testmultiplem2kignores")
		translator := source.Any2KubeTranslator{}
		plan := plantypes.NewPlan()
		want := []plantypes.Service{}

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !cmp.Equal(services, want) {
			t.Fatalf("Failed to get the services properly. Difference:\n%s", cmp.Diff(want, services))
		}
	})
}

func TestTranslate(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get intermediate representation with no services and an empty plan", func(t *testing.T) {
		// Setup
		translator := source.Any2KubeTranslator{}
		services := []plantypes.Service{}
		plan := plantypes.NewPlan()
		want := irtypes.NewIR(plan)
		containerizer.InitContainerizers(plan.Spec.Inputs.RootDir, nil)

		// Test
		ir, err := translator.Translate(services, plan)
		if err != nil {
			t.Fatal("Failed to get the intermediate representation. Error:", err)
		}
		if !cmp.Equal(ir, want) {
			t.Fatalf("Failed to get the intermediate representation properly. Difference:\n%s", cmp.Diff(want, ir))
		}
	})

	t.Run("get intermediate representation with some services and an empty plan", func(t *testing.T) {
		// Setup
		translator := source.Any2KubeTranslator{}

		// Input
		services := []plantypes.Service{}
		testdataservices := "testdata/datafortestingtranslate/servicesfromnodejsapp.yaml"
		if err := common.ReadYaml(testdataservices, &services); err != nil {
			t.Fatalf("Failed to read the testdata at path %q Error: %q", testdataservices, err)
		}
		plan := plantypes.NewPlan()
		containerizer.InitContainerizers(plan.Spec.Inputs.RootDir, nil)

		// Output
		testdatapath := "testdata/datafortestingtranslate/expectedirfornodejsapp.yaml"

		want := irtypes.IR{}
		if err := common.ReadYaml(testdatapath, &want); err != nil {
			t.Fatalf("Failed to read the test data at path %q Error: %q", testdatapath, err)
		}

		// Test
		ir, err := translator.Translate(services, plan)
		if err != nil {
			t.Fatal("Failed to get the intermediate representation. Error:", err)
		}
		if !cmp.Equal(ir, want, cmpopts.EquateEmpty()) {
			t.Fatalf("Failed to get the intermediate representation properly. Differences:\n%s", cmp.Diff(want, ir, cmpopts.EquateEmpty()))
		}
	})
}
