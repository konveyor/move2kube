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
	"regexp"
	"strconv"
	"strings"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/types/qaengine/qagrpc"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
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
	// MultilineInputSolutionFormType allows multiple user input
	MultilineInputSolutionFormType SolutionFormType = "MultiLineInput"
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
	ID        string                  `yaml:"id" json:"id"`
	Type      SolutionFormType        `yaml:"type,omitempty" json:"type,omitempty"`
	Desc      string                  `yaml:"description,omitempty" json:"description,omitempty"`
	Hints     []string                `yaml:"hints,omitempty" json:"hints,omitempty"`
	Options   []string                `yaml:"options,omitempty" json:"options,omitempty"`
	Default   interface{}             `yaml:"default,omitempty" json:"default,omitempty"`
	Answer    interface{}             `yaml:"answer,omitempty" json:"answer,omitempty"`
	Validator func(interface{}) error `yaml:"-" json:"-"`
}

// NewProblem creates a new problem object from a GRPC problem
func NewProblem(p *qagrpc.Problem) (prob Problem, err error) {
	defaults, err := ArrayToInterface(p.Default, SolutionFormType(p.Type))
	if err != nil {
		logrus.Errorf("Unable to convert defaults : %s", err)
		return prob, err
	}
	pp := Problem{
		ID:      p.Id,
		Type:    SolutionFormType(p.Type),
		Desc:    p.Description,
		Hints:   p.Hints,
		Options: p.Options,
		Default: defaults,
	}
	if p.Pattern != "" {
		reg, err := regexp.Compile(p.Pattern)
		if err != nil {
			return pp, fmt.Errorf("not a valid regex pattern : Error : %s", err)
		}
		pp.Validator = func(ans interface{}) error {
			a, ok := ans.(string)
			if !ok {
				return fmt.Errorf("expected input to be type String, got %T. Value : %+v", ans, ans)
			}
			if !reg.MatchString(a) {
				return fmt.Errorf("pattern does not match : %s", a)
			}
			return nil
		}
	}
	return pp, nil
}

// InterfaceToArray converts the answer interface to array
func InterfaceToArray(ansI interface{}, problemType SolutionFormType) (ans []string, err error) {
	if ansI == nil {
		return nil, fmt.Errorf("the answer is nil")
	}
	switch problemType {
	case InputSolutionFormType, PasswordSolutionFormType, MultilineInputSolutionFormType, SelectSolutionFormType:
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

// ArrayToInterface converts the answer array to interface
func ArrayToInterface(ans []string, problemType SolutionFormType) (ansI interface{}, err error) {
	if ansI == nil {
		return nil, nil
	}
	switch problemType {
	case InputSolutionFormType, PasswordSolutionFormType, MultilineInputSolutionFormType, SelectSolutionFormType:
		if len(ans) == 0 {
			return "", nil
		}
		return ans[0], nil
	case ConfirmSolutionFormType:
		if len(ans) == 0 {
			return false, nil
		}
		return cast.ToBoolE(ans[0])
	case MultiSelectSolutionFormType:
		return ans, nil
	default:
		return nil, fmt.Errorf("unsupported QA problem type %+v", problemType)
	}
}

// SetAnswer sets the answer
func (p *Problem) SetAnswer(ansI interface{}, validate bool) error {
	logrus.Trace("Problem.SetAnswer start")
	defer logrus.Trace("Problem.SetAnswer end")
	if ansI == nil {
		return fmt.Errorf("the answer is nil")
	}
	if validate && p.Validator != nil {
		if err := p.Validator(ansI); err != nil {
			return &ValidationError{Reason: err.Error()}
		}
	}
	switch p.Type {
	case InputSolutionFormType, PasswordSolutionFormType, MultilineInputSolutionFormType, SelectSolutionFormType:
		ans, ok := ansI.(string)
		if !ok {
			return fmt.Errorf("expected answer to be string. Actual value %+v is of type %T", ansI, ansI)
		}
		if p.Type == SelectSolutionFormType {
			if !common.IsPresent(p.Options, ans) && !common.IsPresent(p.Options, OtherAnswer) {
				return fmt.Errorf("the answer '%s' has no matching value in the options", ans)
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
			if !common.IsPresent(p.Options, a) {
				logrus.Debugf("the answer '%s' has no matching value in the options. Ignoring", a)
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
func NewSelectProblem(probid, desc string, hints []string, def string, opts []string, validator func(interface{}) error) (Problem, error) {
	var answer interface{}
	if len(opts) == 1 {
		answer = opts[0]
	}
	return Problem{
		ID:        probid,
		Desc:      desc,
		Hints:     hints,
		Type:      SelectSolutionFormType,
		Default:   def,
		Options:   opts,
		Answer:    answer,
		Validator: validator,
	}, nil
}

// NewMultiSelectProblem creates a new instance of multiselect problem
func NewMultiSelectProblem(probid, desc string, hints []string, def []string, opts []string, validator func(interface{}) error) (Problem, error) {
	var answer interface{}
	if len(opts) == 0 {
		answer = []string{}
	}
	return Problem{
		ID:        probid,
		Type:      MultiSelectSolutionFormType,
		Desc:      desc,
		Hints:     hints,
		Options:   opts,
		Default:   def,
		Answer:    answer,
		Validator: validator,
	}, nil
}

// NewConfirmProblem creates a new instance of confirm problem
func NewConfirmProblem(probid, desc string, hints []string, def bool, validator func(interface{}) error) (Problem, error) {
	return Problem{
		ID:        probid,
		Type:      ConfirmSolutionFormType,
		Desc:      desc,
		Hints:     hints,
		Options:   nil,
		Default:   def,
		Answer:    nil,
		Validator: validator,
	}, nil
}

// NewInputProblem creates a new instance of input problem
func NewInputProblem(probid, desc string, hints []string, def string, validator func(interface{}) error) (Problem, error) {
	return Problem{
		ID:        probid,
		Type:      InputSolutionFormType,
		Desc:      desc,
		Hints:     hints,
		Options:   nil,
		Default:   def,
		Answer:    nil,
		Validator: validator,
	}, nil
}

// NewMultilineInputProblem creates a new instance of multiline input problem
func NewMultilineInputProblem(probid, desc string, hints []string, def string, validator func(interface{}) error) (Problem, error) {
	return Problem{
		ID:        probid,
		Type:      MultilineInputSolutionFormType,
		Desc:      desc,
		Hints:     hints,
		Options:   nil,
		Default:   def,
		Answer:    nil,
		Validator: validator,
	}, nil
}

// NewPasswordProblem creates a new instance of password problem
func NewPasswordProblem(probid, desc string, hints []string, validator func(interface{}) error) (p Problem, err error) {
	return Problem{
		ID:        probid,
		Type:      PasswordSolutionFormType,
		Desc:      desc,
		Hints:     hints,
		Options:   nil,
		Default:   nil,
		Answer:    nil,
		Validator: validator,
	}, nil
}
