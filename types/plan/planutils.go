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

package plan

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/common/deepcopy"
	"github.com/konveyor/move2kube/common/pathconverters"
	"github.com/konveyor/move2kube/common/vcs"
	"github.com/sirupsen/logrus"
)

// ReadPlan decodes the plan from yaml converting relative paths to absolute.
func ReadPlan(path string, sourceDir string) (Plan, error) {
	plan := Plan{}
	var err error
	absSourceDir := ""
	if err = common.ReadMove2KubeYaml(path, &plan); err != nil {
		return plan, fmt.Errorf("failed to load the plan file at path '%s' . Error: %w", path, err)
	}
	if sourceDir != "" {
		plan.Spec.SourceDir = sourceDir
	}
	if plan.Spec.SourceDir != "" {
		remoteSrcPath, err := vcs.GetClonedPath(plan.Spec.SourceDir, common.RemoteSourcesFolder, false)
		if err != nil {
			return plan, fmt.Errorf("failed to clone the repo. Error: %w", err)
		}
		if remoteSrcPath != "" {
			plan.Spec.SourceDir = remoteSrcPath
		}
		absSourceDir, err = filepath.Abs(plan.Spec.SourceDir)
		if err != nil {
			return plan, fmt.Errorf("failed to convert sourceDir to full path. Error: %w", err)
		}
	}
	if err = pathconverters.MakePlanPathsAbsolute(&plan, absSourceDir, common.TempPath); err != nil {
		return plan, err
	}
	plan.Spec.SourceDir = absSourceDir
	return plan, nil
}

// WritePlan encodes the plan to yaml converting absolute paths to relative.
func WritePlan(path string, plan Plan) error {
	inputFSPath := plan.Spec.SourceDir
	remoteSrcPath, err := vcs.GetClonedPath(plan.Spec.SourceDir, common.RemoteSourcesFolder, false)
	if err != nil {
		return fmt.Errorf("failed to clone the repo. error: %w", err)
	}
	if remoteSrcPath != "" {
		inputFSPath = remoteSrcPath
	}
	newPlan := deepcopy.DeepCopy(plan).(Plan)
	if err := pathconverters.ChangePaths(&newPlan, map[string]string{inputFSPath: "", common.TempPath: ""}); err != nil {
		return fmt.Errorf("failed to convert plan to use relative paths. Error: %w", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		logrus.Errorf("Unable to get current working dir : %s", err)
		return err
	}
	if remoteSrcPath == "" && plan.Spec.SourceDir != "" {
		if newPlan.Spec.SourceDir, err = filepath.Rel(wd, plan.Spec.SourceDir); err != nil {
			return err
		}
	}
	return common.WriteYaml(path, newPlan)
}
