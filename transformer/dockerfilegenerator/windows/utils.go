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

package windows

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	dotnetutils "github.com/konveyor/move2kube/transformer/dockerfilegenerator/dotnet"
	"github.com/konveyor/move2kube/types/source/dotnet"
)

/*
func getJavaPackage(mappingFile string, version string) (pkg string, err error) {
	var javaPackageNamesMapping JavaPackageNamesMapping
	if err := common.ReadMove2KubeYaml(mappingFile, &javaPackageNamesMapping); err != nil {
		logrus.Debugf("Could not load mapping at %s", mappingFile)
		return "", err
	}
	v, ok := javaPackageNamesMapping.Spec.PackageVersions[version]
	if !ok {
		logrus.Infof("Matching java package not found for java version : %s. Going with default.", version)
		return defaultJavaPackage, nil
	}
	return v, nil
}
*/

func parseCSProj(path string) (dotnet.CSProj, error) {
	configuration := dotnet.CSProj{}
	csProjBytes, err := os.ReadFile(path)
	if err != nil {
		return configuration, fmt.Errorf("failed to read the c sharp project file at path %s . Error: %q", path, err)
	}
	if err := xml.Unmarshal(csProjBytes, &configuration); err != nil {
		return configuration, fmt.Errorf("failed to parse the c sharp project file at path %s . Error: %q", path, err)
	}
	return configuration, nil
}

// isSilverlight checks if the app is silverlight by looking for silverlight regex patterns
func isSilverlight(configuration dotnet.CSProj) (bool, error) {
	if configuration.ItemGroups == nil || len(configuration.ItemGroups) == 0 {
		return false, fmt.Errorf("no item groups in project file to parse")
	}
	for _, ig := range configuration.ItemGroups {
		if ig.Contents == nil || len(ig.Contents) == 0 {
			continue
		}
		for _, r := range ig.Contents {
			if dotnet.WebSLLib.MatchString(r.Include) {
				return true, nil
			}
		}
	}
	return false, nil
}

// isWeb checks if the given app is a web app
func isWeb(configuration dotnet.CSProj) (bool, error) {
	if configuration.ItemGroups == nil || len(configuration.ItemGroups) == 0 {
		return false, fmt.Errorf("no item groups in project file to parse")
	}
	for _, ig := range configuration.ItemGroups {
		if ig.References == nil || len(ig.References) == 0 {
			continue
		}
		for _, r := range ig.References {
			if dotnet.WebLib.MatchString(r.Include) {
				return true, nil
			}
		}
	}
	return false, nil
}

// getCSProjPathsFromSlnFile parses the solution file for cs project file paths.
// If "allPaths" is true then every path we find will be returned (not just c sharp project files).
func getCSProjPathsFromSlnFile(inputPath string, allPaths bool) ([]string, error) {
	slnBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the solution file at path %s . Error: %q", inputPath, err)
	}
	csProjPaths := []string{}
	subMatches := dotnet.ProjBlockRegex.FindAllStringSubmatch(string(slnBytes), -1)
	notWindows := runtime.GOOS != "windows"
	for _, subMatch := range subMatches {
		if len(subMatch) == 0 {
			continue
		}
		csProjPath := strings.Trim(subMatch[1], `"`)
		if notWindows {
			csProjPath = strings.ReplaceAll(csProjPath, `\`, string(os.PathSeparator))
		}
		if !allPaths && filepath.Ext(csProjPath) != dotnetutils.CSPROJ_FILE_EXT {
			continue
		}
		csProjPaths = append(csProjPaths, csProjPath)
	}
	return csProjPaths, nil
}
