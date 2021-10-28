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

package dotnet

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"runtime"
	"strings"

	"github.com/konveyor/move2kube/types/source/dotnet"
)

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

// parseSolutionFile parses the solution file for cs project file paths
func parseSolutionFile(inputPath string) ([]string, error) {
	solFileTxt, err := ioutil.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("could not open the solution file: %s", err)
	}

	projectPaths := make([]string, 0)
	projBlockRegex := regexp.MustCompile(dotnet.ProjBlock)
	l := projBlockRegex.FindAllStringSubmatch(string(solFileTxt), -1)
	for _, path := range l {
		if len(path) == 0 {
			continue
		}
		projectPaths = append(projectPaths, path[0])
	}

	separator := fmt.Sprintf("%c", os.PathSeparator)
	for i, c := range projectPaths {
		c = strings.Trim(c, `"`)
		if runtime.GOOS != "windows" {
			c = strings.ReplaceAll(c, `\`, separator)
		}
		projectPaths[i] = c
	}
	return projectPaths, nil
}
