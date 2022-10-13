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

package qaengine

import (
	"fmt"

	qatypes "github.com/konveyor/move2kube/types/qaengine"
)

// DefaultEngine returns default values for all questions
type DefaultEngine struct {
}

// NewDefaultEngine creates a new instance of default engine
func NewDefaultEngine() *DefaultEngine {
	return new(DefaultEngine)
}

// StartEngine starts the default qa engine
func (*DefaultEngine) StartEngine() error {
	return nil
}

// IsInteractiveEngine returns true if the engine interacts with the user
func (*DefaultEngine) IsInteractiveEngine() bool {
	return false
}

// FetchAnswer fetches the default answers
func (*DefaultEngine) FetchAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	err := prob.SetAnswer(prob.Default)
	if err != nil {
		return prob, err
	}
	if prob.Validator != nil {
		err := prob.Validator(prob.Answer)
		if err != nil {
			return prob, fmt.Errorf("default value is invalid. Error : %s", err)
		}
	}
	return prob, nil
}
