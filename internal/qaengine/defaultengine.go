/*
Copyright IBM Corporation 2020

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

package qaengine

import qatypes "github.com/konveyor/move2kube/types/qaengine"

// DefaultEngine returns default values for all questions
type DefaultEngine struct {
}

// NewDefaultEngine creates a new instance of default engine
func NewDefaultEngine() *DefaultEngine {
	ce := new(DefaultEngine)
	return ce
}

// StartEngine starts the default qa engine
func (c *DefaultEngine) StartEngine() error {
	return nil
}

// FetchAnswer fetches the default answers
func (c *DefaultEngine) FetchAnswer(prob qatypes.Problem) (ans qatypes.Problem, err error) {
	err = prob.SetAnswer(prob.Solution.Default)
	return prob, err
}
