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
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

// CliEngine handles the CLI based qa
type CliEngine struct {
}

// NewCliEngine creates a new instance of cli engine
func NewCliEngine() Engine {
	ce := new(CliEngine)
	return ce
}

// StartEngine starts the cli engine
func (c *CliEngine) StartEngine() error {
	return nil
}

// FetchAnswer fetches the answer using cli
func (c *CliEngine) FetchAnswer(prob qatypes.Problem) (answer qatypes.Problem, err error) {
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
	default:
		log.Fatalf("Unknown type found: %s", prob.Solution.Type)
	}
	return prob, fmt.Errorf("Unknown type found : %s", prob.Solution.Type)
}

func (c *CliEngine) fetchSelectAnswer(prob qatypes.Problem) (answer qatypes.Problem, err error) {
	var ans, d string
	if len(prob.Solution.Default) > 0 {
		d = prob.Solution.Default[0]
	} else {
		d = prob.Solution.Options[0]
	}
	prompt := &survey.Select{
		Message: fmt.Sprintf("%d. %s \nHints: \n %s\n", prob.ID, prob.Desc, prob.Context),
		Options: prob.Solution.Options,
		Default: d,
	}
	err = survey.AskOne(prompt, &ans)
	if err != nil {
		log.Fatalf("Error while asking a question : %s", err)
		return prob, err
	}
	err = prob.SetAnswer([]string{ans})
	return prob, err
}

func (c *CliEngine) fetchMultiSelectAnswer(prob qatypes.Problem) (answer qatypes.Problem, err error) {
	ans := []string{}
	prompt := &survey.MultiSelect{
		Message: fmt.Sprintf("%d. %s \nHints: \n %s\n", prob.ID, prob.Desc, prob.Context),
		Options: prob.Solution.Options,
		Default: prob.Solution.Default,
	}
	err = survey.AskOne(prompt, &ans, survey.WithIcons(func(icons *survey.IconSet) {
		icons.MarkedOption.Text = "[\u2713]"
	}))
	if err != nil {
		log.Fatalf("Error while asking a question : %s", err)
		return prob, err
	}
	err = prob.SetAnswer(ans)
	return prob, err
}

func (c *CliEngine) fetchConfirmAnswer(prob qatypes.Problem) (answer qatypes.Problem, err error) {
	var ans, d bool
	if len(prob.Solution.Default) > 0 {
		d, err = cast.ToBoolE(prob.Solution.Default[0])
		if err != nil {
			log.Warnf("Unable to parse default value : %s", err)
		}
	}
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("%d. %s \nHints: \n %s\n", prob.ID, prob.Desc, prob.Context),
		Default: d,
	}
	err = survey.AskOne(prompt, &ans)
	if err != nil {
		log.Fatalf("Error while asking a question : %s", err)
		return prob, err
	}
	err = prob.SetAnswer([]string{fmt.Sprintf("%v", ans)})
	return prob, err
}

func (c *CliEngine) fetchInputAnswer(prob qatypes.Problem) (answer qatypes.Problem, err error) {
	var ans string
	d := ""
	if len(prob.Solution.Default) > 0 {
		d = prob.Solution.Default[0]
	}
	prompt := &survey.Input{
		Message: fmt.Sprintf("%d. %s \nHints: \n %s\n", prob.ID, prob.Desc, prob.Context),
		Default: d,
	}
	err = survey.AskOne(prompt, &ans)
	if err != nil {
		log.Fatalf("Error while asking a question : %s", err)
		return prob, err
	}
	err = prob.SetAnswer([]string{ans})
	return prob, err
}

func (c *CliEngine) fetchMultilineAnswer(prob qatypes.Problem) (answer qatypes.Problem, err error) {
	var ans string
	d := ""
	if len(prob.Solution.Default) > 0 {
		d = prob.Solution.Default[0]
	}
	prompt := &survey.Multiline{
		Message: fmt.Sprintf("%d. %s \nHints: \n %s\n", prob.ID, prob.Desc, prob.Context),
		Default: d,
	}
	err = survey.AskOne(prompt, &ans)
	if err != nil {
		log.Fatalf("Error while asking a question : %s", err)
		return prob, err
	}
	err = prob.SetAnswer([]string{ans})
	return prob, err
}

func (c *CliEngine) fetchPasswordAnswer(prob qatypes.Problem) (answer qatypes.Problem, err error) {
	var ans string
	prompt := &survey.Password{
		Message: fmt.Sprintf("%d. %s \nHints: \n %s\n", prob.ID, prob.Desc, prob.Context),
	}
	err = survey.AskOne(prompt, &ans)
	if err != nil {
		log.Fatalf("Error while asking a question : %s", err)
		return prob, err
	}
	err = prob.SetAnswer([]string{ans})
	return prob, err
}
