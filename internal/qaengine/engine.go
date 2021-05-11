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
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

// Engine defines interface for qa engines
type Engine interface {
	StartEngine() error
	IsInteractiveEngine() bool
	FetchAnswer(prob qatypes.Problem) (ans qatypes.Problem, err error)
}

var (
	engines     []Engine
	writeStores []qatypes.Store
)

// StartEngine starts the QA Engines
func StartEngine(qaskip bool, qaport int, qadisablecli bool) {
	var e Engine
	if qaskip {
		e = NewDefaultEngine()
	} else if !qadisablecli {
		e = NewCliEngine()
	} else {
		e = NewHTTPRESTEngine(qaport)
	}
	AddEngine(e)
}

// AddEngine appends an engine to the engines slice
func AddEngine(e Engine) {
	if err := e.StartEngine(); err != nil {
		log.Errorf("Ignoring engine %T due to error : %s", e, err)
	} else {
		engines = append(engines, e)
	}
}

// AddEngineHighestPriority adds an engine to the list and sets it at highest priority
func AddEngineHighestPriority(e Engine) error {
	if err := e.StartEngine(); err != nil {
		return fmt.Errorf("failed to start the engine: %T\n%v\nError: %s", e, e, err)
	}
	engines = append([]Engine{e}, engines...)
	return nil
}

// AddCaches adds cache responders.
// Later cache files override earlier cache files.
// [base.yaml, project.yaml, service.yaml]
func AddCaches(cacheFiles []string) {
	common.ReverseInPlace(cacheFiles)
	for _, cacheFile := range cacheFiles {
		e := NewStoreEngineFromCache(cacheFile)
		if err := AddEngineHighestPriority(e); err != nil {
			log.Errorf("Ignoring engine %T due to error : %s", e, err)
			continue
		}
	}
}

// SetupCacheFile adds cache responders
func SetupCacheFile(outputPath string, cacheFiles []string) {
	writeCachePath := filepath.Join(outputPath, common.QACacheFile)
	cache := qatypes.NewCache(writeCachePath)
	cache.Write()
	writeStores = append(writeStores, cache)
	cacheFiles = append(cacheFiles, writeCachePath)
	AddCaches(cacheFiles)
}

// SetupConfigFile adds config responders - should be called only once
func SetupConfigFile(outputPath string, configStrings, configFiles, presets []string) {
	presetPaths := []string{}
	for _, preset := range presets {
		presetPath := filepath.Join(common.AssetsPath, "configs", preset+".yaml")
		presetPaths = append(presetPaths, presetPath)
	}
	configFiles = append(presetPaths, configFiles...)
	writeConfig := qatypes.NewConfig(filepath.Join(outputPath, common.ConfigFile), configStrings, configFiles)
	writeStores = append(writeStores, writeConfig)
	e := &StoreEngine{store: writeConfig}
	if err := AddEngineHighestPriority(e); err != nil {
		log.Errorf("Ignoring engine %T due to error : %s", e, err)
	}
}

// FetchAnswer fetches the answer for the question
func FetchAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	log.Debugf("Fetching answer for problem:\n%v", prob)
	if prob.Solution.Answer != nil {
		log.Debugf("Problem already solved.")
		return prob, nil
	}
	var err error
	for _, e := range engines {
		prob, err = e.FetchAnswer(prob)
		if err != nil {
			log.Debugf("Error while fetching answer using engine %T Error: %q", e, err)
			continue
		}
		if prob.Solution.Answer != nil {
			prob = changeSelectToInputForOther(prob)
			break
		}
	}
	if err != nil || prob.Solution.Answer == nil {
		if err := ValidateProblem(prob); err != nil {
			return prob, fmt.Errorf("the QA problem object is invalid: %+v\nError: %q", prob, err)
		}
		// loop using interactive engine until we get an answer
		lastEngine := engines[len(engines)-1]
		if !lastEngine.IsInteractiveEngine() {
			return prob, fmt.Errorf("failed to fetch the answer for problem\n%+v\nError: %q", prob, err)
		}
		for err != nil || prob.Solution.Answer == nil {
			prob, err = lastEngine.FetchAnswer(prob)
			if err != nil {
				log.Errorf("Unable to get answer to %s Error: %q", prob.Desc, err)
				continue
			}
			if prob.Solution.Answer != nil {
				prob = changeSelectToInputForOther(prob)
			}
		}
	}
	for _, writeStore := range writeStores {
		writeStore.AddSolution(prob)
	}
	return prob, err
}

// WriteStoresToDisk forces all the stores to write their contents out to disk
func WriteStoresToDisk() error {
	var err error
	for _, writeStore := range writeStores {
		cerr := writeStore.Write()
		if cerr != nil {
			if err == nil {
				err = cerr
			} else {
				err = fmt.Errorf("%s : %s", err, cerr)
			}
		}
	}
	return err
}

func changeSelectToInputForOther(prob qatypes.Problem) qatypes.Problem {
	if prob.Solution.Type == qatypes.SelectSolutionFormType && prob.Solution.Answer != nil && prob.Solution.Answer.(string) == qatypes.OtherAnswer {
		newDesc := string(qatypes.InputSolutionFormType) + " " + prob.Desc
		newProb, err := qatypes.NewInputProblem(prob.ID, newDesc, nil, "")
		if err != nil {
			log.Fatalf("failed to change the QA select type problem to input type problem: %+v\nError: %q", prob, err)
		}
		return newProb
	}
	return prob
}

// Convenience functions

// FetchStringAnswer asks a input type question and gets a string as the answer
func FetchStringAnswer(probid, desc string, context []string, def string) string {
	problem, err := qatypes.NewInputProblem(probid, desc, context, def)
	if err != nil {
		log.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Solution.Answer.(string)
	if !ok {
		log.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Solution.Answer, problem.Solution.Answer)
	}
	return answer
}

// FetchBoolAnswer asks a confirm type question and gets a boolean as the answer
func FetchBoolAnswer(probid, desc string, context []string, def bool) bool {
	problem, err := qatypes.NewConfirmProblem(probid, desc, context, def)
	if err != nil {
		log.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Solution.Answer.(bool)
	if !ok {
		log.Fatalf("Answer is not of the correct type. Expected bool. Actual value is %+v of type %T", problem.Solution.Answer, problem.Solution.Answer)
	}
	return answer
}

// FetchSelectAnswer asks a select type question and gets a string as the answer
func FetchSelectAnswer(probid, desc string, context []string, def string, options []string) string {
	problem, err := qatypes.NewSelectProblem(probid, desc, context, def, options)
	if err != nil {
		log.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Solution.Answer.(string)
	if !ok {
		log.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Solution.Answer, problem.Solution.Answer)
	}
	return answer
}

// FetchMultiSelectAnswer asks a multi-select type question and gets a slice of strings as the answer
func FetchMultiSelectAnswer(probid, desc string, context, def, options []string) []string {
	problem, err := qatypes.NewMultiSelectProblem(probid, desc, context, def, options)
	if err != nil {
		log.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, err := common.ConvertInterfaceToSliceOfStrings(problem.Solution.Answer)
	if err != nil {
		log.Fatalf("Answer is not of the correct type. Expected array of strings. Error: %q", err)
	}
	return answer
}

// FetchPasswordAnswer asks a password type question and gets a string as the answer
func FetchPasswordAnswer(probid, desc string, context []string) string {
	problem, err := qatypes.NewPasswordProblem(probid, desc, context)
	if err != nil {
		log.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Solution.Answer.(string)
	if !ok {
		log.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Solution.Answer, problem.Solution.Answer)
	}
	return answer
}

// FetchMultilineAnswer asks a multi-line type question and gets a string as the answer
func FetchMultilineAnswer(probid, desc string, context []string, def string) string {
	problem, err := qatypes.NewMultilineInputProblem(probid, desc, context, def)
	if err != nil {
		log.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		log.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Solution.Answer.(string)
	if !ok {
		log.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Solution.Answer, problem.Solution.Answer)
	}
	return answer
}

// ValidateProblem validates the problem object.
func ValidateProblem(prob qatypes.Problem) error {
	if prob.ID == "" {
		return fmt.Errorf("the QA problem has an empty key: %+v", prob)
	}
	if prob.Desc == "" {
		log.Warnf("the QA problem has an empty description: %+v", prob)
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
