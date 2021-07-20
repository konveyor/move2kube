/*
 *  Copyright IBM Corporation 2020, 2021
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

package lib

import (
	"context"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/transformer"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
)

// Transform transforms the artifacts and writes output
func Transform(ctx context.Context, plan plantypes.Plan, outputPath string) {
	logrus.Debugf("Temp Dir : %s", common.TempPath)
	logrus.Infof("Starting Plan Transformation")
	err := transformer.Transform(plan, outputPath)
	if err != nil {
		logrus.Fatalf("Failed to transform the plan. Error: %q", err)
	}
	logrus.Infof("Plan Transformation done")
}

// Destroy destroys the tranformers
func Destroy() {
	logrus.Debugf("Cleaning up!")
	transformer.Destroy()
}
