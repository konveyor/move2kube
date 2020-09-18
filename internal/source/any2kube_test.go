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

package source_test

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"

	common "github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/source"
	irtypes "github.com/konveyor/move2kube/internal/types"
	plantypes "github.com/konveyor/move2kube/types/plan"
	yaml "gopkg.in/yaml.v3"
)

func TestGetServiceOptions(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get services with a non existent directory and empty plan", func(t *testing.T) {
		// Setup
		inputpath := "this/does/not/exit/foobar/"
		translator := source.Any2KubeTranslator{}
		plan := plantypes.NewPlan()
		want := []plantypes.Service{}

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to get the services properly. Expected:", want, "actual:", services)
		}
	})

	t.Run("get services with empty directory and empty plan", func(t *testing.T) {
		// Setup
		inputpath := t.TempDir()
		translator := source.Any2KubeTranslator{}
		plan := plantypes.NewPlan()
		want := []plantypes.Service{}

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to get the services properly. Expected:", want, "actual:", services)
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

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to get the services properly. Expected:", want, "actual:", services)
		}
	})

	t.Run("get services from a simple nodejs app and empty plan", func(t *testing.T) {
		// Setup
		inputPath := "../../samples/nodejs"
		translator := source.Any2KubeTranslator{}

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		plan.Spec.Inputs.RootDir = inputPath

		want := []plantypes.Service{}
		if err := common.ReadYaml("testdata/expectedservicesfornodejsapp.yaml", &want); err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to create the services properly. Expected:", want, "Actual", services)
		}
	})

	t.Run("get services from a simple nodejs app and filled plan", func(t *testing.T) {
		// Setup
		inputPath := "../../samples/nodejs"
		translator := source.Any2KubeTranslator{}

		// services
		svc1 := plantypes.NewService("svc1", "Any2Kube")
		svc1.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = []string{"foo/"}
		svc2 := plantypes.NewService("svc2", "Any2Kube")
		svc2.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = []string{"bar/"}

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		plan.Spec.Inputs.RootDir = inputPath
		plan.Spec.Inputs.Services = map[string][]plantypes.Service{
			"svc1": {svc1},
			"svc2": {svc2},
		}

		want := []plantypes.Service{}
		if err := common.ReadYaml("testdata/expectedservicesfornodejsapp.yaml", &want); err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to create the services properly. Expected:", want, "Actual", services)
		}
	})

	t.Run("get services from a simple nodejs app that we already containerized", func(t *testing.T) {
		// Setup
		inputPath := "../../samples/nodejs"
		translator := source.Any2KubeTranslator{}

		// services
		svc1 := plantypes.NewService("svc1", "Any2Kube")
		svc1.SourceArtifacts[plantypes.SourceDirectoryArtifactType] = []string{"."}

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		plan.Spec.Inputs.RootDir = inputPath
		plan.Spec.Inputs.Services = map[string][]plantypes.Service{
			"svc1": {svc1},
		}

		want := []plantypes.Service{}
		// if err := common.ReadYaml("testdata/expectedservicesfornodejsapp.yaml", &want); err != nil {
		// 	t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		// }

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to create the services properly. Expected:", want, "Actual", services)
		}
	})

	t.Run("test m2kignore can ignore a directory but include its subdirectories", func(t *testing.T) {
		// 1. Ignore a directory, but include all subdirectories

		// Setup
		inputPath := "testdata/nodejsappwithm2kignorecase1"
		translator := source.Any2KubeTranslator{}

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		plan.Spec.Inputs.RootDir = inputPath

		want := []plantypes.Service{}
		if err := common.ReadYaml("testdata/expectedservicesfornodejsappwithm2kignorecase1.yaml", &want); err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to create the services properly. Expected:", want, "Actual", services)
		}
	})

	t.Run("test m2kignore can be used to ignore everything but a very specific subdirectory", func(t *testing.T) {
		// Setup
		inputPath := "testdata/javamavenappwithm2kignorecase2"
		translator := source.Any2KubeTranslator{}

		plan := plantypes.NewPlan()
		plan.Name = "nodejs-app"
		plan.Spec.Inputs.RootDir = inputPath

		want := []plantypes.Service{}
		if err := common.ReadYaml("testdata/expectedservicesforjavamavenappwithm2kignorecase2.yaml", &want); err != nil {
			t.Fatal("Failed to read the expected output services from yaml. Error:", err)
		}

		// Test
		services, err := translator.GetServiceOptions(inputPath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		// if err := common.WriteYaml("testdata/hmm.yaml", services); err != nil {
		// 	t.Fatal("error", err)
		// }
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to create the services properly. Expected:", want, "Actual", services)
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

		// Test
		services, err := translator.GetServiceOptions(inputpath, plan)
		if err != nil {
			t.Fatal("Failed to get the services. Error:", err)
		}
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to get the services properly. Expected:", want, "actual:", services)
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
		if !reflect.DeepEqual(services, want) {
			t.Fatal("Failed to get the services properly. Expected:", want, "actual:", services)
		}
	})
}

func TestTranslate(t *testing.T) {
	log.SetLevel(log.DebugLevel)

	t.Run("get intermediate representation with no services and an empty plan", func(t *testing.T) {
		// Setup
		// inputpath := "this/does/not/exit/foobar/"
		translator := source.Any2KubeTranslator{}
		services := []plantypes.Service{}
		plan := plantypes.NewPlan()
		want := irtypes.NewIR(plan)

		// Test
		ir, err := translator.Translate(services, plan)
		if err != nil {
			t.Fatal("Failed to get the intermediate representation. Error:", err)
		}
		if !reflect.DeepEqual(ir, want) {
			t.Fatal("Failed to get the intermediate representation properly. Expected:", want, "actual:", ir)
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

		// Output
		testdataoutput := "testdata/datafortestingtranslate/expectedirfornodejsapp.yaml"
		wantbytes, err := ioutil.ReadFile(testdataoutput)
		if err != nil {
			t.Fatalf("Failed to read the testdata at path %q Error: %q", testdataoutput, err)
		}
		wantyaml := string(wantbytes)

		// Test
		ir, err := translator.Translate(services, plan)
		if err != nil {
			t.Fatal("Failed to get the intermediate representation. Error:", err)
		}
		irbytes, err := yaml.Marshal(ir)
		if err != nil {
			t.Fatal("Failed to marshal the intermediate representation to yaml for comparison. Error:", err)
		}
		iryaml := string(irbytes)
		if iryaml != wantyaml {
			t.Fatal("Failed to get the intermediate representation properly. Expected:", wantyaml, "actual:", iryaml)
		}
	})
}
