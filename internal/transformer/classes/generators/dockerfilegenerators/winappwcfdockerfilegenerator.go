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

package dockerfilegenerators

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/apex/log"
	"github.com/konveyor/move2kube/environment"
	plantypes "github.com/konveyor/move2kube/types/plan"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

// WinAppWCFDockerfileGenerator implements the Transformer interface
type WinAppWCFDockerfileGenerator struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// Init Initializes the transformer
func (t *WinAppWCFDockerfileGenerator) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *WinAppWCFDockerfileGenerator) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *WinAppWCFDockerfileGenerator) BaseDirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *WinAppWCFDockerfileGenerator) DirectoryDetect(dir string) (namedServices map[string]plantypes.Service, unnamedServices []plantypes.Transformer, err error) {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Error while trying to read directory: %s", err)
		return nil, nil, err
	}
	for _, de := range dirEntries {
		ext := filepath.Ext(de.Name())
		if ext != csProj {
			continue
		}
		xmlFile, err := os.Open(filepath.Join(dir, de.Name()))
		if err != nil {
			logrus.Errorf("Could not open the csproj file: %s", err)
			continue
		}

		defer xmlFile.Close()

		byteValue, err := ioutil.ReadAll(xmlFile)
		if err != nil {
			logrus.Errorf("Could not read the csproj file: %s", err)
			continue
		}

		configuration := ConfigurationDotNET{}
		xml.Unmarshal(byteValue, &configuration)
		metadata.version = configuration.PropertyGroup.TargetFrameworkVersion
		metadata.path, err = filepath.Rel(dir, csprojFile)
		if err != nil {
			fmt.Println(err)
		}
		metadata.appName = strings.TrimSuffix(filepath.Base(csprojFile), filepath.Ext(csprojFile))
		break
	}
	if IsDotNet4x(metadata.version) {
		cfgFiles, err := GetFilesByExt(sourcePath, []string{config})
		if err != nil {
			fmt.Println(err)
		}
		isAppConfigPresent := false
		for _, cfgFile := range cfgFiles {
			if filepath.Base(cfgFile) == "App.config" {
				appConfigFile, err := os.Open(cfgFile)
				if err != nil {
					fmt.Println(err)
				}

				defer appConfigFile.Close()

				byteValue, _ := ioutil.ReadAll(appConfigFile)
				appCfg := AppConfig{}
				xml.Unmarshal(byteValue, &appCfg)

				if !IsWCF(appCfg) {
					return false, metadata
				}
				isAppConfigPresent = true
				for _, addKey := range appCfg.AppCfgSettings.AddList {
					if parsedUrl, err := url.ParseRequestURI(addKey.Value); err == nil && parsedUrl.Scheme != "" && parsedUrl.Host != "" {
						_, port, _ := net.SplitHostPort(parsedUrl.Host)
						if err == nil {
							metadata.ports = append(metadata.ports, port)
						}
					}
				}
			}
		}

		if !isAppConfigPresent {
			return false, metadata
		}

		metadata.baseImageVersion = defaultBaseImageVersion

		return true, metadata
	}

	return false, metadata

	isDotNET, configInfo := DetectDotNET(dir)
	if !isDotNET {
		return nil, nil, nil
	}

	namedServices = map[string]plantypes.Service{
		configInfo.appName: []plantypes.Transformer{{
			Mode:              t.Config.Spec.Mode,
			ArtifactTypes:     []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {dir},
			},
			Configs: map[string]interface{}{
				artifacts.DockerfileTemplateConfigConfigType: map[string]interface{}{
					"Ports":            configInfo.ports,
					"baseImageVersion": configInfo.baseImageVersion,
					"appName":          configInfo.appName,
				},
			},
		}},
	}
	return namedServices, nil, nil
}

// Transform transforms the artifacts
func (t *WinAppWCFDockerfileGenerator) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	return transform(t.Config, t.Env, newArtifacts)
}

const (
	csProj                  = ".csproj"
	config                  = ".config"
	defaultBaseImageVersion = "4.8"
)

type Metadata struct {
	version          string
	ports            []string
	appName          string
	path             string
	baseImageVersion string
}

type AppConfig struct {
	XMLName        xml.Name       `xml:"configuration"`
	Startup        StartUp        `xml:"startup"`
	AppCfgSettings AppCfgSettings `xml:"appSettings"`
}

type StartUp struct {
	XMLName          xml.Name         `xml:"startup"`
	SupportedRuntime SupportedRuntime `xml:"supportedRuntime"`
}

type SupportedRuntime struct {
	XMLName xml.Name `xml:"supportedRuntime"`
	Version string   `xml:"version,attr"`
	Sku     string   `xml:"sku,attr"`
}

type AppCfgSettings struct {
	XMLName xml.Name  `xml:"appSettings"`
	AddList []AddKeys `xml:"add"`
}

type AddKeys struct {
	XMLName xml.Name `xml:"add"`
	Key     string   `xml:"key,attr"`
	Value   string   `xml:"value,attr"`
}

type ConfigurationDotNET struct {
	XMLName       xml.Name        `xml:"Project"`
	Sdk           string          `xml:"Sdk,attr"`
	PropertyGroup []PropertyGroup `xml:"PropertyGroup"`
}

type PropertyGroup struct {
	XMLName                xml.Name `xml:"PropertyGroup"`
	Condition              string   `xml:"Condition,attr"`
	TargetFrameworkVersion string   `xml:"TargetFrameworkVersion"`
}

//GetFilesByExt returns files by extension
func GetFilesByExt(inputPath string, exts []string) ([]string, error) {
	var files []string
	if info, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Warnf("Error in walking through files due to : %q", err)
		return nil, err
	} else if !info.IsDir() {
		log.Warnf("The path %q is not a directory.", inputPath)
	}
	err := filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil && path == inputPath { // if walk for root search path return gets error
			// then stop walking and return this error
			return err
		}
		if err != nil {
			log.Warnf("Skipping path %q due to error: %q", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		fext := filepath.Ext(path)
		for _, ext := range exts {
			if fext == ext {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files due to : %q", err)
		return files, err
	}
	log.Debugf("No of files with %s ext identified : %d", exts, len(files))
	return files, nil
}

func IsDotNet4x(version string) bool {
	r, _ := regexp.Compile("v4.*")
	return r.MatchString(version)
}

func IsWCF(appCfg AppConfig) bool {
	return len(appCfg.Startup.SupportedRuntime.Version) > 0
}

//DetectDotNETreturns .NET version if detects .NET
func DetectDotNET(sourcePath string) (bool, Metadata) {
	var metadata Metadata
	csprojFiles, err := GetFilesByExt(sourcePath, []string{csProj})
	if err != nil {
		fmt.Println(err)
	}
	for _, csprojFile := range csprojFiles {
		xmlFile, err := os.Open(csprojFile)
		if err != nil {
			fmt.Println(err)
		}

		defer xmlFile.Close()

		byteValue, _ := ioutil.ReadAll(xmlFile)

		configuration := ConfigurationDotNET{}
		xml.Unmarshal(byteValue, &configuration)
		metadata.version = configuration.PropertyGroup.TargetFramework
		metadata.path, err = filepath.Rel(sourcePath, csprojFile)
		if err != nil {
			fmt.Println(err)
		}
		metadata.appName = strings.TrimSuffix(filepath.Base(csprojFile), filepath.Ext(csprojFile))
	}
	if IsDotNet4x(metadata.version) {
		cfgFiles, err := GetFilesByExt(sourcePath, []string{config})
		if err != nil {
			fmt.Println(err)
		}
		isAppConfigPresent := false
		for _, cfgFile := range cfgFiles {
			if filepath.Base(cfgFile) == "App.config" {
				appConfigFile, err := os.Open(cfgFile)
				if err != nil {
					fmt.Println(err)
				}

				defer appConfigFile.Close()

				byteValue, _ := ioutil.ReadAll(appConfigFile)
				appCfg := AppConfig{}
				xml.Unmarshal(byteValue, &appCfg)

				if !IsWCF(appCfg) {
					return false, metadata
				}
				isAppConfigPresent = true
				for _, addKey := range appCfg.AppCfgSettings.AddList {
					if parsedUrl, err := url.ParseRequestURI(addKey.Value); err == nil && parsedUrl.Scheme != "" && parsedUrl.Host != "" {
						_, port, _ := net.SplitHostPort(parsedUrl.Host)
						if err == nil {
							metadata.ports = append(metadata.ports, port)
						}
					}
				}
			}
		}

		if !isAppConfigPresent {
			return false, metadata
		}

		metadata.baseImageVersion = defaultBaseImageVersion

		return true, metadata
	}

	return false, metadata
}
