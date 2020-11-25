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
	"strings"
	"sync"

	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
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
	// MultilineSolutionFormType allows multiple user input
	MultilineSolutionFormType SolutionFormType = "MultiLine"
	// PasswordSolutionFormType allows password entry
	PasswordSolutionFormType SolutionFormType = "Password"
	// ConfirmSolutionFormType allows yes/no answers
	ConfirmSolutionFormType SolutionFormType = "Confirm"
)

var (
	lastAssignedProblemID   = 0             // keep track of IDs
	problemIDIncrementMutex = &sync.Mutex{} // manage incrementing of problem ids atomically
)

// Problem defines the QA problem
type Problem struct {
	ID       int          `yaml:"-" json:"id"`
	Desc     string       `yaml:"description" json:"description"`
	Context  []string     `yaml:"context,omitempty" json:"context,omitempty"`
	Solution SolutionForm `yaml:"solution" json:"solution,omitempty"`
	Resolved bool         `yaml:"resolved,omitempty" json:"resolved,omitempty"`
}

// SolutionForm defines the solution
type SolutionForm struct {
	Type    SolutionFormType `yaml:"type" json:"type"`
	Default []string         `yaml:"default,omitempty" json:"default,omitempty"`
	Options []string         `yaml:"options,omitempty" json:"options,omitempty"`
	Answer  []string         `yaml:"answer" json:"answer"`
}

// SetAnswer sets the answer
func (p *Problem) SetAnswer(answer []string) error {
	if p.Solution.Type != MultiSelectSolutionFormType && len(answer) == 0 {
		return fmt.Errorf("The answer slice is empty")
	}
	if p.Solution.Type != MultiSelectSolutionFormType && len(answer) > 1 {
		return fmt.Errorf("The question type is not multiselect, but there are multiple answers")
	}
	if p.Solution.Type == SelectSolutionFormType || p.Solution.Type == MultiSelectSolutionFormType {
		success := true
		p.Solution.Answer = []string{}
		for _, a := range answer {
			if !common.IsStringPresent(p.Solution.Options, a) {
				log.Warnf("No matching value in options for %s. Ignoring.", a)
				success = false
				continue
			}
			p.Solution.Answer = append(p.Solution.Answer, a)
		}
		if !success {
			return fmt.Errorf("Unknown options selected")
		}
		p.Resolved = true
		return nil
	}
	if p.Solution.Type == ConfirmSolutionFormType {
		_, err := cast.ToBoolE(answer[0])
		if err != nil {
			log.Warnf("Error while parsing answer for confirm question type : %s", err)
			return err
		}
	}
	p.Solution.Answer = []string{answer[0]}
	p.Resolved = true
	return nil
}

// GetSliceAnswer returns a slice as answer if the solution type supports it
func (p *Problem) GetSliceAnswer() (ans []string, err error) {
	if !p.Resolved {
		return ans, fmt.Errorf("Problem yet to be resolved")
	}
	if p.Solution.Type != MultiSelectSolutionFormType {
		return p.Solution.Answer, fmt.Errorf("This question type does not support this answer type")
	}
	return p.Solution.Answer, nil
}

// GetBoolAnswer returns a bool as answer if the solution type supports it
func (p *Problem) GetBoolAnswer() (ans bool, err error) {
	if !p.Resolved {
		return ans, fmt.Errorf("Problem yet to be resolved")
	}
	if p.Solution.Type != ConfirmSolutionFormType {
		return false, fmt.Errorf("This question type does not support this answer type")
	}
	if len(p.Solution.Answer) != 1 {
		return false, fmt.Errorf("No answer available")
	}
	ans, err = cast.ToBoolE(p.Solution.Answer[0])
	if err != nil {
		return false, err
	}
	return ans, nil
}

// GetStringAnswer returns a string as answer if the solution type supports it
func (p *Problem) GetStringAnswer() (ans string, err error) {
	if !p.Resolved {
		return ans, fmt.Errorf("Problem yet to be resolved")
	}
	if p.Solution.Type == MultiSelectSolutionFormType || p.Solution.Type == ConfirmSolutionFormType {
		return "", fmt.Errorf("This question type does not support this answer type")
	}
	if len(p.Solution.Answer) != 1 {
		return "", fmt.Errorf("Wrong number of answers")
	}
	return p.Solution.Answer[0], nil
}

// Matches checks if the problems are same
func (p *Problem) matches(np Problem) bool {
	if !p.matchString(p.Desc, np.Desc) || p.Solution.Type != np.Solution.Type {
		return false
	}
	return true
}

// Compares str1 with str2 in a case-insensitive manner
// Tries to compile str1 as a regex and check for full match
func (p *Problem) matchString(str1 string, str2 string) bool {
	if strings.EqualFold(str1, str2) {
		return true
	}
	r, err := regexp.MatchString(str1, str2)
	if err != nil {
		log.Debugf("Unable to compile string %s : %s", str1, err)
		return false
	}
	return r
}

func newProblem(t SolutionFormType, desc string, context []string, def []string, opts []string) (p Problem, err error) {
	resolved := false
	answer := []string{}
	if desc == "" {
		return p, fmt.Errorf("Empty Description")
	}
	switch t {
	case MultiSelectSolutionFormType:
		if len(opts) == 0 {
			resolved = true
		}
		if len(def) > 0 {
			for _, d := range def {
				if !common.IsStringPresent(opts, d) {
					return p, fmt.Errorf("Default value [%s] not present in options [%+v]", d, opts)
				}
			}
		}
	case SelectSolutionFormType:
		if len(opts) == 0 {
			return p, fmt.Errorf("Atleast one option is required for question %s", desc)
		}
		if len(opts) == 1 {
			answer = opts
			resolved = true
		}
		if len(def) > 1 {
			log.Warnf("Only one default is allowed for question %s. Setting default as first value %s", desc, def)
			def = []string{def[0]}
		}
		if len(def) == 0 {
			def = []string{opts[0]}
		} else {
			if !common.IsStringPresent(opts, def[0]) {
				return p, fmt.Errorf("Default value [%s] not present in options [%+v]", def[0], opts)
			}
		}
	case ConfirmSolutionFormType:
		if len(opts) > 0 {
			log.Warnf("Options is not required for confirm question type : %s", desc)
		}
		if len(def) > 1 {
			log.Warnf("Only one default is allowed for question %s.", desc)
		}
		if len(def) == 0 {
			def = []string{cast.ToString(false)}
		} else {
			_, err := cast.ToBoolE(def[0])
			if err != nil {
				log.Warnf("Unable to parse default value %s. Setting as false", def[0])
				def = []string{cast.ToString(false)}
			}
			def = []string{def[0]}
		}
	case InputSolutionFormType, MultilineSolutionFormType:
		if len(def) > 1 {
			log.Warnf("Only one default value supported for %s. Ignoring others.", desc)
			def = []string{def[0]}
		}
		if len(opts) > 0 {
			log.Warnf("Options not supported for %s. Ignoring options.", desc)
			opts = []string{}
		}
	case PasswordSolutionFormType:
		if len(def) > 0 {
			log.Warnf("Default not supported for %s. Ignoring default.", desc)
			def = []string{}
		}
		if len(opts) > 0 {
			log.Warnf("Options not supported for %s. Ignoring options.", desc)
			opts = []string{}
		}
	}
	return Problem{
		ID:       getProblemID(),
		Desc:     desc,
		Context:  context,
		Solution: SolutionForm{Type: t, Default: def, Options: opts, Answer: answer},
		Resolved: resolved,
	}, nil
}

// getProblemID returns a new problem id
func getProblemID() int {
	problemIDIncrementMutex.Lock()
	lastAssignedProblemID++
	currID := lastAssignedProblemID
	problemIDIncrementMutex.Unlock()
	return currID
}

// NewSelectProblem creates a new instance of select problem
func NewSelectProblem(desc string, context []string, def string, opts []string) (p Problem, err error) {
	return newProblem(SelectSolutionFormType, desc, context, []string{def}, opts)
}

// NewMultiSelectProblem creates a new instance of multiselect problem
func NewMultiSelectProblem(desc string, context []string, def []string, opts []string) (p Problem, err error) {
	return newProblem(MultiSelectSolutionFormType, desc, context, def, opts)
}

// NewConfirmProblem creates a new instance of confirm problem
func NewConfirmProblem(desc string, context []string, def bool) (p Problem, err error) {
	return newProblem(ConfirmSolutionFormType, desc, context, []string{cast.ToString(def)}, []string{})
}

// NewInputProblem creates a new instance of input problem
func NewInputProblem(desc string, context []string, def string) (p Problem, err error) {
	return newProblem(InputSolutionFormType, desc, context, []string{def}, []string{})
}

// NewMultilineInputProblem creates a new instance of multiline input problem
func NewMultilineInputProblem(desc string, context []string, def string) (p Problem, err error) {
	return newProblem(MultilineSolutionFormType, desc, context, []string{def}, []string{})
}

// NewPasswordProblem creates a new instance of password problem
func NewPasswordProblem(desc string, context []string) (p Problem, err error) {
	return newProblem(PasswordSolutionFormType, desc, context, []string{}, []string{})
}
