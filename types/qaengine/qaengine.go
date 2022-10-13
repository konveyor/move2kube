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

/*
Package qaengine contains the types used for the question answering part of the CLI.
*/
package qaengine

import "fmt"

// Store helps store answers
type Store interface {
	Load() error
	GetSolution(Problem) (Problem, error)

	Write() error
	AddSolution(p Problem) error
}

// ValidationError is the error while validating answer in QA Engine
type ValidationError struct {
	Reason string
}

func (v *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", v.Reason)
}
