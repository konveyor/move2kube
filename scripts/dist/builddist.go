//go:build ignore
// +build ignore

/*
 *  Copyright IBM Corporation 2020
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

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	checksumSuffix = ".sha256sum"
)

var (
	// binName is the name of the exectuable
	binName string
	// version is the version of the exectuable
	version string
	// outputDir is the path where the artifacts should be generated.
	outputDir string
)

func sha256sum(source, target string) error {
	file, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("Failed to open the archive at path %q Error %q", source, err)
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("Failed to caculate the checksum for the archive at path %q Error %q", source, err)
	}
	filename := filepath.Base(source)
	hashAndFilename := fmt.Sprintf("%x  %s", hasher.Sum(nil), filename) // Same format as the output of shasum -a 256 myarchive.tar.gz
	if err := os.WriteFile(target, []byte(hashAndFilename), common.DefaultFilePermission); err != nil {
		return fmt.Errorf("Failed to write the checksum to file at path %q Error %q", target, err)
	}
	return file.Close()
}

func createZip(source, target string) error {
	cmd := exec.Command("zip", "-r", target, source)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to create tar archive %q using files from %q. Output: %q Error %q", target, source, string(out), err)
	}
	return nil
}

func createTar(source, target string) error {
	cmd := exec.Command("tar", "-zcf", target, source)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to create tar archive %q using files from %q. Output: %q Error %q", target, source, string(out), err)
	}
	return nil
}

func copy(sourceFiles []string, target string) error {
	args := append([]string{"-r"}, sourceFiles...)
	args = append(args, target)
	cmd := exec.Command("cp", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to copy files from source files %v to target %q Output: %q Error %q", sourceFiles, target, string(out), err)
	}
	return nil
}

func findDistDirs() []string {
	osArchRegex := regexp.MustCompile("^[^-]+-[^-]+$")
	distDirs := []string{}

	err := filepath.Walk(".", func(path string, finfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		if !finfo.IsDir() {
			return fmt.Errorf("Found a non directory file at path %q", path)
		}
		dirName := filepath.Base(path)
		if osArchRegex.MatchString(dirName) {
			distDirs = append(distDirs, path)
		}
		return filepath.SkipDir
	})
	if err != nil {
		logrus.Fatalf("Failed to walk through the current working directory. Error: %q", err)
	}
	if len(distDirs) == 0 {
		logrus.Fatal("Failed to find the directories containing the build output.")
	}
	return distDirs
}

func createArchives(distDirs []string) {
	tempDir := binName
	extraFilesDir := filepath.Join("files", "*")
	extraFiles, err := filepath.Glob(extraFilesDir)
	if err != nil {
		logrus.Fatalf("Failed to get the files in the directory at path %q Error %q", extraFilesDir, err)
	}
	if err := os.MkdirAll(outputDir, common.DefaultDirectoryPermission); err != nil {
		logrus.Fatalf("Failed to make the output directory at path %s Error: %q", outputDir, err)
	}
	logrus.Debugf("Generating output in directory at path %s", outputDir)

	logrus.Debug("tempDir:", tempDir)
	logrus.Debug("extraFiles:", extraFiles)

	for _, distDir := range distDirs {
		logrus.Debug("Remove and remake the temporary directory.")
		if err := os.RemoveAll(tempDir); err != nil {
			logrus.Fatalf("Failed to remove the temporary directory at path %q Error: %q", tempDir, err)
		}
		if err := os.Mkdir(tempDir, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to make the temporary directory at path %q Error: %q", tempDir, err)
		}

		logrus.Debug("Copy the files over.")
		buildArtifacts, err := filepath.Glob(filepath.Join(distDir, "*"))
		if err != nil {
			logrus.Fatalf("Failed to get the files in the build directory at path %q Error %q", distDir, err)
		}
		logrus.Debug("buildArtifacts:", buildArtifacts)
		if err := copy(buildArtifacts, tempDir); err != nil {
			logrus.Fatal(err)
		}
		if err := copy(extraFiles, tempDir); err != nil {
			logrus.Fatal(err)
		}

		logrus.Debug("Name and make the archives.")
		osArch := filepath.Base(distDir)
		tarArchiveName := fmt.Sprintf("%s-%s-%s.tar.gz", binName, version, osArch)
		tarArchivePath := filepath.Join(outputDir, tarArchiveName)
		logrus.Debug("osArch:", osArch)
		logrus.Debug("tarArchivePath:", tarArchivePath)
		if err := createTar(tempDir, tarArchivePath); err != nil {
			logrus.Fatal(err)
		}
		zipArchiveName := fmt.Sprintf("%s-%s-%s.zip", binName, version, osArch)
		zipArchivePath := filepath.Join(outputDir, zipArchiveName)
		logrus.Debug("zipArchivePath:", zipArchivePath)
		if err := createZip(tempDir, zipArchivePath); err != nil {
			logrus.Fatal(err)
		}

		logrus.Debug("Calculate and write the checksums to files.")
		if err := sha256sum(tarArchivePath, filepath.Join(outputDir, tarArchiveName+checksumSuffix)); err != nil {
			logrus.Fatal(err)
		}
		if err := sha256sum(zipArchivePath, filepath.Join(outputDir, zipArchiveName+checksumSuffix)); err != nil {
			logrus.Fatal(err)
		}
	}

	logrus.Debug("Cleanup the temporary directory.")
	if err := os.RemoveAll(tempDir); err != nil {
		logrus.Warnf("Failed to remove the temporary directory at path %q Error: %q", tempDir, err)
	}
}

func createDistributions() {
	logrus.Infof("Creating archive files for distribution.")

	logrus.Debug("BINNAME:", binName)
	logrus.Debug("VERSION:", version)

	logrus.Debug("Find the directories containing the build output.")
	distDirs := findDistDirs()
	logrus.Debug("distDirs:", distDirs)

	logrus.Debug("Create the archives.")
	createArchives(distDirs)

	logrus.Infof("Done!")
}

func main() {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}

	logrus.SetLevel(logrus.DebugLevel)

	rootCmd := &cobra.Command{
		Use:   "go run builddist.go",
		Short: "builddist creates the distribution files.",
		Long:  `Generate the archives and the corresponding checksum files.`,
		Run:   func(_ *cobra.Command, _ []string) { createDistributions() },
	}
	rootCmd.Flags().StringVarP(&binName, "binname", "b", "", "Name of the executable")
	rootCmd.Flags().StringVarP(&version, "version", "v", "", "Version of the executable")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "output", "Version of the executable")
	must(rootCmd.MarkFlagRequired("binname"))
	must(rootCmd.MarkFlagRequired("version"))

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal("Error:", err)
	}
}
