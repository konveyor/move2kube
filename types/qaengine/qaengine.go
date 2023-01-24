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

import (
	"encoding/base64"
	"fmt"

	"github.com/sirupsen/logrus"
)

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

// Error returns the error message as a string.
func (v *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", v.Reason)
}

// Serialize transforms certain fields of the problem before it gets written to the store.
func Serialize(p Problem) (Problem, error) {
	logrus.Tracef("Serialize start p: %+v", p)
	defer logrus.Trace("Serialize end")
	switch p.Type {
	case PasswordSolutionFormType:
		if p.Answer == nil {
			return p, fmt.Errorf("the answer for the password type problem is nil")
		}
		answer, ok := p.Answer.(string)
		if !ok {
			return p, fmt.Errorf("expected the answer for the password type problem to be a string. Actual type %T and value %+v", p.Answer, p.Answer)
		}
		p.Answer = base64.StdEncoding.EncodeToString([]byte(answer))
		return p, nil
	default:
		return p, nil
	}
}

// Deserialize transforms certain fields of the problem after it gets read from the store.
func Deserialize(p Problem) (Problem, error) {
	logrus.Tracef("Deserialize start p: %+v", p)
	defer logrus.Trace("Deserialize end")
	switch p.Type {
	case PasswordSolutionFormType:
		if p.Answer == nil {
			return p, fmt.Errorf("the answer for the password type problem is nil")
		}
		answer, ok := p.Answer.(string)
		if !ok {
			return p, fmt.Errorf("expected the answer for the password type problem to be a string. Actual type %T and value %+v", p.Answer, p.Answer)
		}
		ansBytes, err := base64.StdEncoding.DecodeString(answer)
		if err != nil {
			return p, fmt.Errorf("failed to base64 decode the answer for the password type problem. Error: %w", err)
		}
		p.Answer = string(ansBytes)
		return p, nil
	default:
		return p, nil
	}
}
