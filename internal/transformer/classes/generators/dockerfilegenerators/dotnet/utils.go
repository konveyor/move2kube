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
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/types/source/dotnet"
)

// isSilverlight checks if the app is silverlight by looking for silverlight regex patterns
func isSilverlight(configuration dotnet.CSProj) (bool, error) {
	if configuration.ItemGroups == nil || len(configuration.ItemGroups) == 0 {
		return false, fmt.Errorf("No item groups in project file to parse")
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
		return false, fmt.Errorf("No item groups in project file to parse")
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
	solFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("Could not open the solution file: %s", err)
	}
	defer solFile.Close()

	solFileScanner := bufio.NewScanner(solFile)

	r := regexp.MustCompile(dotnet.ProjBlock)

	csr := regexp.MustCompile(dotnet.CsProj)

	projectPaths := make([]string, 0)
	for solFileScanner.Scan() {
		s := solFileScanner.Text()
		if r.MatchString(s) {
			tokens := strings.Split(s, "=")
			if len(tokens[1]) > 0 {
				values := strings.Split(tokens[1], ",")
				for _, v := range values {
					if csr.MatchString(v) {
						projectPaths = append(projectPaths, v)
					}
				}
			}
		}
	}

	if err := solFileScanner.Err(); err != nil {
		return nil, fmt.Errorf("Could not parse the solution file: %s", err)
	}

	for i, c := range projectPaths {
		c = strings.Replace(c, "\"", "", -1)
		c = strings.Replace(c, "\\", "/", -1)
		projectPaths[i] = c
	}

	return projectPaths, nil
}
