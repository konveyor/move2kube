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

package transformer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/move2kube/common"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
)

func TestProcessPathMappings(t *testing.T) {
	t.Run("no-pathmappings", func(t *testing.T) {
		pms := []transformertypes.PathMapping{}
		sourcePath := ""
		outputPath := ""
		if err := processPathMappings(pms, sourcePath, outputPath, true); err != nil {
			t.Fatalf("failed to process the path mappings. Error: %q", err)
		}
	})
	t.Run("single-default-pathmapping", func(t *testing.T) {
		sourcePath := "./testdata"
		pms := []transformertypes.PathMapping{{
			Type:     transformertypes.DefaultPathMappingType,
			SrcPath:  filepath.Join(sourcePath, "k8s-yamls"),
			DestPath: "yamls",
		}}
		outputPath := t.TempDir()
		if err := processPathMappings(pms, sourcePath, outputPath, true); err != nil {
			t.Fatalf("failed to process the path mappings. Error: %q", err)
		}
		{
			fs, err := os.ReadDir(outputPath)
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			for _, f := range fs {
				if f.Name() != "yamls" {
					t.Fatalf("wrong folder name. f %+v f.Name %s", f, f.Name())
				}
			}
		}
		{
			fs, err := os.ReadDir(filepath.Join(outputPath, "yamls"))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			for _, f := range fs {
				if f.Name() != "deployment.yaml" {
					t.Fatalf("wrong file name. f %+v f.Name %s", f, f.Name())
				}
			}
		}
		{
			expectedContents, err := os.ReadFile(filepath.Join(sourcePath, "k8s-yamls", "deployment.yaml"))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			actualContents, err := os.ReadFile(filepath.Join(outputPath, "yamls", "deployment.yaml"))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			if diff := cmp.Diff(string(actualContents), string(expectedContents)); diff != "" {
				t.Fatalf("wrong file contents. Differences: %s", diff)
			}
		}
	})
	t.Run("single-source-pathmapping", func(t *testing.T) {
		sourcePath := "./testdata"
		pms := []transformertypes.PathMapping{{
			Type:     transformertypes.SourcePathMappingType,
			SrcPath:  "src",
			DestPath: "source",
		}}
		outputPath := t.TempDir()
		if err := processPathMappings(pms, sourcePath, outputPath, true); err != nil {
			t.Fatalf("failed to process the path mappings. Error: %q", err)
		}
		{
			fs, err := os.ReadDir(outputPath)
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			for _, f := range fs {
				if f.Name() != "source" {
					t.Fatalf("wrong folder name. f %+v f.Name %s", f, f.Name())
				}
			}
		}
		expectedFilenames := []string{
			"index.js",
			"package-lock.json",
			"package.json",
		}
		{
			fs, err := os.ReadDir(filepath.Join(outputPath, "source"))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			for i, f := range fs {
				if f.Name() != expectedFilenames[i] {
					t.Fatalf("wrong file name. expected: '%s' actual: '%s' f %+v", expectedFilenames[i], f.Name(), f)
				}
			}
		}
		for _, expectedFilename := range expectedFilenames {
			expectedContents, err := os.ReadFile(filepath.Join(sourcePath, "src", expectedFilename))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			actualContents, err := os.ReadFile(filepath.Join(outputPath, "source", expectedFilename))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			if diff := cmp.Diff(string(actualContents), string(expectedContents)); diff != "" {
				t.Fatalf("wrong file contents for filename '%s' . Differences: %s", expectedFilename, diff)
			}
		}
	})

	t.Run("single-delete-pathmapping", func(t *testing.T) {
		sourcePath := "./testdata/src"
		sourceIndexJSPath := filepath.Join(sourcePath, "index.js")
		outputPath := t.TempDir()
		outputIndexJSPath := filepath.Join(outputPath, "index.js")
		if err := common.CopyFile(outputIndexJSPath, sourceIndexJSPath); err != nil {
			t.Fatalf("Error: %q", err)
		}
		expectedFilenames := []string{"index.js"}
		{
			fs, err := os.ReadDir(outputPath)
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			for i, f := range fs {
				if f.Name() != expectedFilenames[i] {
					t.Fatalf("wrong file name. expected: '%s' actual: '%s' f %+v", expectedFilenames[i], f.Name(), f)
				}
			}
		}
		for _, expectedFilename := range expectedFilenames {
			expectedContents, err := os.ReadFile(filepath.Join(sourcePath, expectedFilename))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			actualContents, err := os.ReadFile(filepath.Join(outputPath, expectedFilename))
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			if diff := cmp.Diff(string(actualContents), string(expectedContents)); diff != "" {
				t.Fatalf("wrong file contents for filename '%s' . Differences: %s", expectedFilename, diff)
			}
		}
		pms := []transformertypes.PathMapping{{
			Type:     transformertypes.DeletePathMappingType,
			DestPath: outputIndexJSPath,
		}}
		if err := processPathMappings(pms, sourcePath, outputPath, true); err != nil {
			t.Fatalf("failed to process the path mappings. Error: %q", err)
		}
		{
			fs, err := os.ReadDir(outputPath)
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			for _, f := range fs {
				if f.Name() != "source" {
					t.Fatalf("wrong folder name. f %+v f.Name %s", f, f.Name())
				}
			}
		}
		{
			fs, err := os.ReadDir(outputPath)
			if err != nil {
				t.Fatalf("Error: %q", err)
			}
			if len(fs) != 0 {
				t.Fatalf("expected an empty output directory. actual: %+v", fs)
			}
		}
	})
}
