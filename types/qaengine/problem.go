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

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/types/qaengine/qagrpc"
	"github.com/sirupsen/logrus"
)

// SolutionFormType is the type that defines different types of solutions possible
type SolutionFormType string

const (
	// SelectSolutionFormType defines a single select solution type
	SelectSolutionFormType SolutionFormType = "Select"
	// MultiSelectSolutionFormType defines a multi-select solution type
	MultiSelectSolutionFormType SolutionFormType = "MultiSelect"
	// InputSolutionFormType allows single line user input
	InputSolutionFormType SolutionFormType = "Input"
	// MultilineSolutionFormType allows multiple user input
	MultilineSolutionFormType SolutionFormType = "MultiLine"
	// PasswordSolutionFormType allows password entry
	PasswordSolutionFormType SolutionFormType = "Password"
	// ConfirmSolutionFormType allows yes/no answers
	ConfirmSolutionFormType SolutionFormType = "Confirm"
)

const (
	// OtherAnswer - Use as one of the answers, when there is a option to enter the answer in Select Question Type
	OtherAnswer = "Other (specify custom option)"
)

// Problem defines the QA problem
type Problem struct {
	ID      string           `yaml:"id" json:"id"`
	Type    SolutionFormType `yaml:"type,omitempty" json:"type,omitempty"`
	Desc    string           `yaml:"description,omitempty" json:"description,omitempty"`
	Hints   []string         `yaml:"hints,omitempty" json:"hints,omitempty"`
	Options []string         `yaml:"options,omitempty" json:"options,omitempty"`
	Default interface{}      `yaml:"default,omitempty" json:"default,omitempty"`
	Answer  interface{}      `yaml:"answer,omitempty" json:"answer,omitempty"`
}

func NewProblem(p *qagrpc.Problem) (prob Problem, err error) {
	defaults, err := ArrayToInterface(p.Default, SolutionFormType(p.Type))
	if err != nil {
		logrus.Errorf("Unable to convert defaults : %s", err)
		return prob, err
	}
	return Problem{
		ID:      p.Id,
		Type:    SolutionFormType(p.Type),
		Desc:    p.Description,
		Hints:   p.Hints,
		Options: p.Options,
		Default: defaults,
	}, nil
}

func InterfaceToArray(ansI interface{}, problemType SolutionFormType) (ans []string, err error) {
	if ansI == nil {
		return nil, fmt.Errorf("the answer is nil")
	}
	switch problemType {
	case InputSolutionFormType, PasswordSolutionFormType, MultilineSolutionFormType, SelectSolutionFormType:
		ans, ok := ansI.(string)
		if !ok {
			return nil, fmt.Errorf("expected answer to be string. Actual value %+v is of type %T", ansI, ansI)
		}
		return []string{ans}, nil
	case ConfirmSolutionFormType:
		ans, ok := ansI.(bool)
		if !ok {
			return nil, fmt.Errorf("expected answer to be bool. Actual value %+v is of type %T", ansI, ansI)
		}
		return []string{strconv.FormatBool(ans)}, nil
	case MultiSelectSolutionFormType:
		ans, err := common.ConvertInterfaceToSliceOfStrings(ansI)
		if err != nil {
			return nil, fmt.Errorf("expected answer to be an array of strings. Error: %q", err)
		}
		return ans, nil
	default:
		return nil, fmt.Errorf("unsupported QA problem type %+v", problemType)
	}
}

func ArrayToInterface(ans []string, problemType SolutionFormType) (ansI interface{}, err error) {
	if ansI == nil {
		return nil, nil
	}
	switch problemType {
	case InputSolutionFormType, PasswordSolutionFormType, MultilineSolutionFormType, SelectSolutionFormType:
		if len(ans) == 0 {
			return "", nil
		}
		return ans[0], nil
	case ConfirmSolutionFormType:
		if len(ans) == 0 {
			return false, nil
		}
		return strconv.ParseBool(ans[0])
	case MultiSelectSolutionFormType:
		return ans, nil
	default:
		return nil, fmt.Errorf("unsupported QA problem type %+v", problemType)
	}
}

// SetAnswer sets the answer
func (p *Problem) SetAnswer(ansI interface{}) error {
	if ansI == nil {
		return fmt.Errorf("the answer is nil")
	}
	switch p.Type {
	case InputSolutionFormType, PasswordSolutionFormType, MultilineSolutionFormType, SelectSolutionFormType:
		ans, ok := ansI.(string)
		if !ok {
			return fmt.Errorf("expected answer to be string. Actual value %+v is of type %T", ansI, ansI)
		}
		if p.Type == SelectSolutionFormType {
			if !common.IsStringPresent(p.Options, ans) {
				return fmt.Errorf("no matching value in options for %s", ans)
			}
		}
		p.Answer = ans
	case ConfirmSolutionFormType:
		ans, ok := ansI.(bool)
		if !ok {
			return fmt.Errorf("expected answer to be bool. Actual value %+v is of type %T", ansI, ansI)
		}
		p.Answer = ans
	case MultiSelectSolutionFormType:
		ans, err := common.ConvertInterfaceToSliceOfStrings(ansI)
		if err != nil {
			return fmt.Errorf("expected answer to be an array of strings. Error: %q", err)
		}
		p.Answer = ans
		filteredAns := []string{}
		for _, a := range ans {
			if !common.IsStringPresent(p.Options, a) {
				logrus.Debugf("No matching value in options for %s. Ignoring.", a)
				continue
			}
			filteredAns = append(filteredAns, a)
		}
		p.Answer = filteredAns
		logrus.Debugf("Answering multiselect question %s with %+v", p.ID, p.Answer)
	default:
		return fmt.Errorf("unsupported QA problem type %+v", p.Type)
	}
	return nil
}

// Matches checks if the problems are same
func (p *Problem) matches(np Problem) bool {
	return p.Type == np.Type && p.matchString(p.Desc, np.Desc)
}

// Compares str1 with str2 in a case-insensitive manner
// Tries to compile str1 as a regex and check for full match
func (p *Problem) matchString(str1 string, str2 string) bool {
	if strings.EqualFold(str1, str2) {
		return true
	}
	r, err := regexp.MatchString(str1, str2)
	if err != nil {
		logrus.Debugf("Unable to compile string %s : %s", str1, err)
		return false
	}
	return r
}

// NewSelectProblem creates a new instance of select problem
func NewSelectProblem(probid, desc string, hints []string, def string, opts []string) (Problem, error) {
	var answer interface{}
	if len(opts) == 1 {
		answer = opts[0]
	}
	return Problem{
		ID:    probid,
		Desc:  desc,
		Hints: hints,
		Type:  SelectSolutionFormType, Default: def, Options: opts, Answer: answer,
	}, nil
}

// NewMultiSelectProblem creates a new instance of multiselect problem
func NewMultiSelectProblem(probid, desc string, hints []string, def []string, opts []string) (Problem, error) {
	var answer interface{}
	if len(opts) == 0 {
		answer = []string{}
	}
	return Problem{
		ID:      probid,
		Type:    MultiSelectSolutionFormType,
		Desc:    desc,
		Hints:   hints,
		Options: opts,
		Default: def,
		Answer:  answer,
	}, nil
}

// NewConfirmProblem creates a new instance of confirm problem
func NewConfirmProblem(probid, desc string, hints []string, def bool) (Problem, error) {
	return Problem{
		ID:      probid,
		Type:    ConfirmSolutionFormType,
		Desc:    desc,
		Hints:   hints,
		Options: nil,
		Default: def,
		Answer:  nil,
	}, nil
}

// NewInputProblem creates a new instance of input problem
func NewInputProblem(probid, desc string, hints []string, def string) (Problem, error) {
	return Problem{
		ID:      probid,
		Type:    InputSolutionFormType,
		Desc:    desc,
		Hints:   hints,
		Options: nil,
		Default: def,
		Answer:  nil,
	}, nil
}

// NewMultilineInputProblem creates a new instance of multiline input problem
func NewMultilineInputProblem(probid, desc string, hints []string, def string) (Problem, error) {
	return Problem{
		ID:      probid,
		Type:    MultilineSolutionFormType,
		Desc:    desc,
		Hints:   hints,
		Options: nil,
		Default: def,
		Answer:  nil,
	}, nil
}

// NewPasswordProblem creates a new instance of password problem
func NewPasswordProblem(probid, desc string, hints []string) (p Problem, err error) {
	return Problem{
		ID:      probid,
		Type:    PasswordSolutionFormType,
		Desc:    desc,
		Hints:   hints,
		Options: nil,
		Default: nil,
		Answer:  nil,
	}, nil
}
