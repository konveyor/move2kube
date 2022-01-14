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
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/konveyor/move2kube/common"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	"github.com/sirupsen/logrus"
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
		logrus.Errorf("the QA problem object is invalid. Error: %q", err)
		return prob, err
	}
	switch prob.Type {
	case qatypes.SelectSolutionFormType:
		return c.fetchSelectAnswer(prob)
	case qatypes.MultiSelectSolutionFormType:
		return c.fetchMultiSelectAnswer(prob)
	case qatypes.ConfirmSolutionFormType:
		return c.fetchConfirmAnswer(prob)
	case qatypes.InputSolutionFormType:
		return c.fetchInputAnswer(prob)
	case qatypes.MultilineInputSolutionFormType:
		return c.fetchMultilineInputAnswer(prob)
	case qatypes.PasswordSolutionFormType:
		return c.fetchPasswordAnswer(prob)
	}
	logrus.Fatalf("unknown QA problem type: %+v", prob)
	return prob, nil
}

func (*CliEngine) fetchSelectAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def string
	if prob.Default != nil {
		def = prob.Default.(string)
	} else {
		def = prob.Options[0]
	}
	prompt := &survey.Select{
		Message: getQAMessage(prob),
		Options: prob.Options,
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		logrus.Fatalf("Error while asking a question : %s", err)
	}
	prob.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchMultiSelectAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	ans := []string{}
	prompt := &survey.MultiSelect{
		Message: getQAMessage(prob),
		Options: prob.Options,
		Default: prob.Default,
	}
	tickIcon := func(icons *survey.IconSet) { icons.MarkedOption.Text = "[\u2713]" }
	if err := survey.AskOne(prompt, &ans, survey.WithIcons(tickIcon)); err != nil {
		logrus.Fatalf("Error while asking a question : %s", err)
	}
	otherAnsPresent := false
	newAns := []string{}
	for _, a := range ans {
		if a == qatypes.OtherAnswer {
			otherAnsPresent = true
		} else {
			newAns = append(newAns, a)
		}
	}
	if otherAnsPresent {
		multilineAns := ""
		prompt := &survey.Multiline{
			Message: getQAMessage(prob),
			Default: "",
		}
		if err := survey.AskOne(prompt, &multilineAns); err != nil {
			logrus.Fatalf("Error while asking a question : %s", err)
		}
		for _, lineAns := range strings.Split(multilineAns, "\n") {
			lineAns = strings.TrimSpace(lineAns)
			if lineAns != "" {
				if !common.IsStringPresent(newAns, lineAns) {
					newAns = append(newAns, lineAns)
				}
			}
		}
	}
	prob.Answer = newAns
	return prob, nil
}

func (*CliEngine) fetchConfirmAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def bool
	if prob.Default != nil {
		def = prob.Default.(bool)
	}
	prompt := &survey.Confirm{
		Message: getQAMessage(prob),
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		logrus.Fatalf("Error while asking a question : %s", err)
	}
	prob.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchInputAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def string
	if prob.Default != nil {
		def = prob.Default.(string)
	}
	prompt := &survey.Input{
		Message: getQAMessage(prob),
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		logrus.Fatalf("Error while asking a question : %s", err)
	}
	prob.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchMultilineInputAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans, def string
	if prob.Default != nil {
		def = prob.Default.(string)
	}
	prompt := &survey.Multiline{
		Message: getQAMessage(prob),
		Default: def,
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		logrus.Fatalf("Error while asking a question : %s", err)
	}
	prob.Answer = ans
	return prob, nil
}

func (*CliEngine) fetchPasswordAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	var ans string
	prompt := &survey.Password{
		Message: getQAMessage(prob),
	}
	if err := survey.AskOne(prompt, &ans); err != nil {
		logrus.Fatalf("Error while asking a question : %s", err)
	}
	prob.Answer = ans
	return prob, nil
}

func getQAMessage(prob qatypes.Problem) string {
	if prob.Desc == "" {
		prob.Desc = "Default description for question with id: " + prob.ID
	}
	if len(prob.Hints) == 0 {
		return fmt.Sprintf("%s\nID: %s\n", prob.Desc, prob.ID)
	}
	return fmt.Sprintf("%s\nID: %s\nHints:\n[%s]\n", prob.Desc, prob.ID, strings.Join(prob.Hints, ", "))
}
