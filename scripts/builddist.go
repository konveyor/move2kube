// +build !excludedist

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

package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
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
	hash := string(hasher.Sum(nil))
	filename := filepath.Base(target)

	err = common.WriteTemplateToFile(`{{.Hash}} {{.Filename}}`, struct {
		Hash     string
		Filename string
	}{
		Hash:     hash,
		Filename: filename,
	}, target, common.DefaultFilePermission)
	if err != nil {
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

func main() {
	log.SetLevel(log.DebugLevel)
	binName := os.Args[1]
	version := os.Args[2]
	checksumSuffix := ".sha256sum"
	join := filepath.Join

	log.Infof("Creating archive files for distribution.")
	log.Debug("BINNAME:", binName)
	log.Debug("VERSION:", version)

	currDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get the current working directory. Error: %q", err)
	}

	log.Debug("Find the directories containing the build output.")
	osArchRegex := regexp.MustCompile("^[^-]+-[^-]+$")
	distDirs := []string{}

	err = filepath.Walk(currDir, func(path string, finfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == currDir {
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
		log.Fatalf("Failed to create the archive files. Error: %q", err)
	}
	if len(distDirs) == 0 {
		log.Fatal("Failed to find the directories containing the build output.")
	}
	log.Debug("distDirs:", distDirs)

	log.Debug("Create the archives.")
	tempDir := join(currDir, binName)
	extraFiles, err := filepath.Glob(join(currDir, "files", "*"))
	if err != nil {
		log.Fatalf("Failed to get the files in the directory at path %q Error %q", currDir, err)
	}

	log.Debug("tempDir:", tempDir)
	log.Debug("extraFiles:", extraFiles)

	for _, distDir := range distDirs {
		log.Debug("Remove and remake the temporary directory.")
		if err := os.RemoveAll(tempDir); err != nil {
			log.Fatalf("Failed to remove the temporary directory at path %q Error: %q", tempDir, err)
		}
		if err := os.Mkdir(tempDir, common.DefaultDirectoryPermission); err != nil {
			log.Fatalf("Failed to make the temporary directory at path %q Error: %q", tempDir, err)
		}

		log.Debug("Copy the files over.")
		buildArtifacts, err := filepath.Glob(join(distDir, "*"))
		if err != nil {
			log.Fatalf("Failed to get the files in the build directory at path %q Error %q", distDir, err)
		}
		log.Debug("buildArtifacts:", buildArtifacts)
		if err := copy(buildArtifacts, tempDir); err != nil {
			log.Fatal(err)
		}
		if err := copy(extraFiles, tempDir); err != nil {
			log.Fatal(err)
		}

		log.Debug("Name and make the archives.")
		osArch := filepath.Base(distDir)
		tarArchiveName := fmt.Sprintf("%s-%s-%s.tar.gz", binName, version, osArch)
		tarArchivePath := join(currDir, tarArchiveName)
		if err := createTar(tempDir, tarArchivePath); err != nil {
			log.Fatal(err)
		}
		zipArchiveName := fmt.Sprintf("%s-%s-%s.zip", binName, version, osArch)
		zipArchivePath := join(currDir, zipArchiveName)
		if err := createZip(tempDir, zipArchivePath); err != nil {
			log.Fatal(err)
		}

		log.Debug("Calculate and write the checksums to files.")
		if err := sha256sum(tarArchivePath, tarArchivePath+checksumSuffix); err != nil {
			log.Fatal(err)
		}
		if err := sha256sum(zipArchivePath, zipArchivePath+checksumSuffix); err != nil {
			log.Fatal(err)
		}
	}

	log.Debug("Cleanup the temporary directory.")
	if err := os.RemoveAll(tempDir); err != nil {
		log.Errorf("Failed to remove the temporary directory at path %q Error: %q", tempDir, err)
	}

	log.Infof("Done!")
}
