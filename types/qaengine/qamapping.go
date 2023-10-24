/*
 *  Copyright IBM Corporation 2023
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

package qaengine

import (
	"strings"

	"github.com/gobwas/glob"
	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

// QAMapping defines the structure of the QA Category mapping file
type QAMapping struct {
	Categories []QACategory `yaml:"categories"`
}

// QACategory represents a category of QA engine questions
type QACategory struct {
	Name      string   `yaml:"name"`
	Enabled   bool     `yaml:"enabled"`
	Questions []string `yaml:"questions"`
}

// GetProblemCategories returns the list of categories a particular question belongs to
func GetProblemCategories(targetProbId string, additionalCategories []string) []string {
	categories := additionalCategories

	for category, probIds := range common.QACategoryMap {
		for _, probId := range probIds {
			// if the problem ID contains a *, interpret it as a glob
			if strings.Contains(probId, "*") {
				g, err := glob.Compile(probId)
				if err != nil {
					logrus.Errorf("invalid problem ID glob: %s\n", probId)
					continue
				}
				if g.Match(targetProbId) {
					categories = append(categories, category)
				}
			} else if targetProbId == probId {
				categories = append(categories, category)
			}
		}

	}

	if len(categories) == 0 {
		categories = append(categories, "default")
	}

	return categories
}
