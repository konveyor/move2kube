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

	"github.com/AlecAivazis/survey/v2"
	"github.com/konveyor/move2kube/internal/common"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

// CliEngine handles the CLI based qa
type CliEngine struct {
}

// NewCliEngine creates a new instance of cli engine
func NewCliEngine() Engine {
	return new(CliEngine)
}

// StartEngine starts the cli engine
func (*CliEngine) StartEngine() error {
	return nil
}

// IsInteractiveEngine returns true if the engine interacts with the user
func (*CliEngine) IsInteractiveEngine() bool {
	return true
}

// FetchAnswer fetches the answer using cli
func (c *CliEngine) FetchAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	if err := ValidateProblem(prob); err != nil {
		log.Errorf("the QA problem object is invalid. Error: %q", err)
		return prob, err
	}
	switch prob.Solution.Type {
	case qatypes.SelectSolutionFormType:
		return c.fetchSelectAnswer(prob)
	case qatypes.MultiSelectSolutionFormType:
		return c.fetchMultiSelectAnswer(prob)
	case qatypes.ConfirmSolutionFormType:
		return c.fetchConfirmAnswer(prob)
	case qatypes.InputSolutionFormType:
		return c.fetchInputAnswer(prob)
	case qatypes.MultilineSolutionFormType:
		return c.fetchMultilineAnswer(prob)
	case qatypes.PasswordSolutionFormType:
		return c.fetchPasswordAnswer(prob)
	}
	log.Fatalf("unknown QA problem type: %+v", prob)
	return prob, nil
}

// ValidateProblem validates the problem object.
func ValidateProblem(prob qatypes.Problem) error {
	if prob.ID == "" {
		return fmt.Errorf("the QA problem has an empty key: %+v", prob)
	}
	if prob.Desc == "" {
		return fmt.Errorf("the QA problem has an empty description: %+v", prob)
	}
	if prob.Context != nil {
		if _, err := common.ConvertInterfaceToSliceOfStrings(prob.Context); err != nil {
			return fmt.Errorf("expected the hints to be an array of strings for the QA problem: %+v\nError: %q", prob, err)
		}
	}
	switch prob.Solution.Type {
	case qatypes.MultiSelectSolutionFormType:
		if len(prob.Solution.Options) == 0 {
			log.Debugf("the QA multiselect problem has no options specified: %+v", prob)
			if prob.Solution.Default != nil {
				xs, err := common.ConvertInterfaceToSliceOfStrings(prob.Solution.Default)
				if err != nil {
					return fmt.Errorf("the QA multiselect problem has a default which is not an array of strings and has no options specified: %+v", prob)
				}
				if len(xs) > 0 {
					return fmt.Errorf("the QA multiselect problem has a default set but no options specified: %+v", prob)
				}
			}
			return nil
		}
		if prob.Solution.Default != nil {
			defaults, err := common.ConvertInterfaceToSliceOfStrings(prob.Solution.Default)
			if err != nil {
				return fmt.Errorf("expected the defaults to be an array of strings for the QA multiselect problem: %+v\nError: %q", prob, err)
			}
			for _, def := range defaults {
				if !common.IsStringPresent(prob.Solution.Options, def) {
					return fmt.Errorf("one of the defaults [%s] is not present in the options for the QA multiselect problem: %+v", def, prob)
				}
			}
		}
	case qatypes.SelectSolutionFormType:
		if len(prob.Solution.Options) == 0 {
			return fmt.Errorf("the QA select problem has no options specified: %+v", prob)
		}
		if prob.Solution.Default != nil {
			def, ok := prob.Solution.Default.(string)
			if !ok {
				return fmt.Errorf("expected the default to be a string for the QA select problem: %+v", prob)
			}
			if !common.IsStringPresent(prob.Solution.Options, def) {
				return fmt.Errorf("the default [%s] is not present in the options for the QA select problem: %+v", def, prob)
			}
		}
	case qatypes.ConfirmSolutionFormType:
		if len(prob.Solution.Options) > 0 {
			log.Warnf("options are not supported for the QA confirm question type: %+v", prob)
		}
		if prob.Solution.Default != nil {
			if _, ok := prob.Solution.Default.(bool); !ok {
				return fmt.Errorf("expected the default to be a bool for the QA confirm problem: %+v", prob)
			}
		}
	case qatypes.InputSolutionFormType, qatypes.MultilineSolutionFormType, qatypes.PasswordSolutionFormType:
		if len(prob.Solution.Options) > 0 {
			log.Warnf("options are not supported for the QA input/multiline/password question types: %+v", prob)
		}
		if prob.Solution.Default != nil {
			if prob.Solution.Type == qatypes.PasswordSolutionFormType {
				log.Warnf("default is not supported for the QA password question type: %+v", prob)
			} else {
				if _, ok := prob.Solution.Default.(string); !ok {
					return fmt.Errorf("expected the default to be a string for the QA input/multiline problem: %+v", prob)
				}
			}
		}
	default:
		return fmt.Errorf("unknown QA problem type: %+v", prob)
	}
	return nil
}

func (*CliEngine) fetchSelectAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def string
	if prob.Solution.Default != nil {
		def = prob.Solution.Default.(string)
	} else {
		def = prob.Solution.Options[0]
	}
	prompt := &survey.Select{
		Message: getQAMessage(prob),
		Options: prob.Solution.Options,
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		log.Fatalf("Error while asking a question : %s", err)
	}
	prob.Solution.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchMultiSelectAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	ans := []string{}
	prompt := &survey.MultiSelect{
		Message: getQAMessage(prob),
		Options: prob.Solution.Options,
		Default: prob.Solution.Default,
	}
	tickIcon := func(icons *survey.IconSet) { icons.MarkedOption.Text = "[\u2713]" }
	if err := survey.AskOne(prompt, &ans, survey.WithIcons(tickIcon)); err != nil {
		log.Fatalf("Error while asking a question : %s", err)
	}
	prob.Solution.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchConfirmAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def bool
	if prob.Solution.Default != nil {
		def = prob.Solution.Default.(bool)
	}
	prompt := &survey.Confirm{
		Message: getQAMessage(prob),
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		log.Fatalf("Error while asking a question : %s", err)
	}
	prob.Solution.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchInputAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def string
	if prob.Solution.Default != nil {
		def = prob.Solution.Default.(string)
	}
	prompt := &survey.Input{
		Message: getQAMessage(prob),
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		log.Fatalf("Error while asking a question : %s", err)
	}
	prob.Solution.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchMultilineAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def string
	if prob.Solution.Default != nil {
		def = prob.Solution.Default.(string)
	}
	prompt := &survey.Multiline{
		Message: getQAMessage(prob),
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		log.Fatalf("Error while asking a question : %s", err)
	}
	prob.Solution.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchPasswordAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans string
	prompt := &survey.Password{
		Message: getQAMessage(prob),
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		log.Fatalf("Error while asking a question : %s", err)
	}
	prob.Solution.Answer = ans
	return prob, nil
}

func getQAMessage(prob qatypes.Problem) string {
	message := fmt.Sprintf("%s \n", prob.Desc)
	if prob.Context != nil {
		message = fmt.Sprintf("%s \nHints: \n %s\n", prob.Desc, prob.Context)
	}
	return message
}
