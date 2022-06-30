/*
 *  Copyright IBM Corporation 2022
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

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/qaengine"
)

type buildOption string

const (
	// NO_BUILD_STAGE don't generate the build stage in Dockerfiles
	NO_BUILD_STAGE buildOption = "no build stage"
	// BUILD_IN_BASE_IMAGE generate the build stage and put it in a separate Dockerfile
	BUILD_IN_BASE_IMAGE buildOption = "build stage in base image"
	// BUILD_IN_EVERY_IMAGE generate the build stage in every Dockerfile
	BUILD_IN_EVERY_IMAGE buildOption = "build stage in every image"
)

// AskUserForDockerfileType asks the user what type of Dockerfiles to generate.
func AskUserForDockerfileType(rootProjectName string) (buildOption, error) {
	quesId := common.JoinQASubKeys(common.ConfigServicesKey, rootProjectName, "dockerfileType")
	desc := fmt.Sprintf("What type of Dockerfiles should be generated for the service '%s'?", rootProjectName)
	options := []string{
		string(NO_BUILD_STAGE),
		string(BUILD_IN_BASE_IMAGE),
		string(BUILD_IN_EVERY_IMAGE),
	}
	def := BUILD_IN_BASE_IMAGE
	hints := []string{
		fmt.Sprintf("[%s] There is no build stage. Dockerfiles will only contain the run stage. The .dll files will need to be built and present in the file system already, for them to get copied into the container.", NO_BUILD_STAGE),
		fmt.Sprintf("[%s] Put the build stage in a separate Dockerfile and create a base image.", BUILD_IN_BASE_IMAGE),
		fmt.Sprintf("[%s] Put the build stage in every Dockerfile to make it self contained. (Warning: This may cause one build per Dockerfile.)", BUILD_IN_EVERY_IMAGE),
	}
	selectedBuildOption := buildOption(qaengine.FetchSelectAnswer(quesId, desc, hints, string(def), options))
	switch selectedBuildOption {
	case NO_BUILD_STAGE, BUILD_IN_BASE_IMAGE, BUILD_IN_EVERY_IMAGE:
		return selectedBuildOption, nil
	}
	return def, fmt.Errorf("user selected an unsupported option for generating Dockerfiles. Actual: %s", selectedBuildOption)
}
