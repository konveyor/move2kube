/*
Copyright IBM Corporation 2020, 2021

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

package api

import (
	"github.com/konveyor/move2kube/internal/translator"
	plantypes "github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
)

// Translate translates the artifacts and writes output
func Translate(plan plantypes.Plan, outputPath string) {
	logrus.Infof("Starting Plan Translation")
	err := translator.Translate(plan, outputPath)
	if err != nil {
		logrus.Fatalf("Failed to translate the plan. Error: %q", err)
	}
	logrus.Infof("Plan Translation done")
}
