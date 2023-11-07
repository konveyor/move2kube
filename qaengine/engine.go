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
	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/common/download"
	qatypes "github.com/konveyor/move2kube-wasm/types/qaengine"
	"github.com/sirupsen/logrus"
	"path/filepath"
)

// Engine defines interface for qa engines
type Engine interface {
	StartEngine() error
	IsInteractiveEngine() bool
	FetchAnswer(prob qatypes.Problem) (ans qatypes.Problem, err error)
}

var (
	engines       []Engine
	stores        []qatypes.Store
	defaultEngine = NewDefaultEngine()
)

// StartEngine starts the QA Engines
func StartEngine(qaskip bool, qaport int, qadisablecli bool) {
	var e Engine
	if qaskip {
		e = NewDefaultEngine()
	}
	//TODO: WASI
	// else if !qadisablecli {
	//	e = NewCliEngine()
	//} else {
	//	e = NewHTTPRESTEngine(qaport)
	//}
	AddEngine(e)
}

// AddEngine appends an engine to the engines slice
func AddEngine(e Engine) {
	if err := e.StartEngine(); err != nil {
		logrus.Errorf("Ignoring engine %T due to error : %s", e, err)
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
func AddCaches(cacheFiles ...string) {
	common.ReverseInPlace(cacheFiles)
	for _, cacheFile := range cacheFiles {
		e := NewStoreEngineFromCache(cacheFile, false)
		if err := AddEngineHighestPriority(e); err != nil {
			logrus.Errorf("Ignoring engine %T due to error : %s", e, err)
			continue
		}
	}
}

// SetupWriteCacheFile adds write cache
func SetupWriteCacheFile(writeCachePath string, persistPasswords bool) {
	cache := qatypes.NewCache(writeCachePath, persistPasswords)
	cache.Write()
	stores = append(stores, cache)
	AddCaches(writeCachePath)
}

// SetupConfigFile adds config responders - should be called only once
func SetupConfigFile(writeConfigFile string, configStrings, configFiles, presets []string, persistPasswords bool) {
	presetPaths := []string{}
	for _, preset := range presets {
		presetPath := filepath.Join(common.AssetsPath, "built-in", "presets", preset+".yaml")
		presetPaths = append(presetPaths, presetPath)
	}
	for i, configFile := range configFiles {
		if download.IsRemotePath(configFile) {
			downloadedPath := download.GetDownloadedPath(configFile, common.RemoteCustomizationsFolder, true)
			if downloadedPath != "" {
				configFiles[i] = downloadedPath
			}
		}
	}
	configFiles = append(presetPaths, configFiles...)
	writeConfig := qatypes.NewConfig(writeConfigFile, configStrings, configFiles, persistPasswords)
	if writeConfigFile != "" {
		stores = append(stores, writeConfig)
	}
	e := &StoreEngine{store: writeConfig}
	if err := AddEngineHighestPriority(e); err != nil {
		logrus.Errorf("Ignoring engine %T due to error : %s", e, err)
	}
}

// FetchAnswer fetches the answer for the question
func FetchAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	logrus.Trace("FetchAnswer start")
	defer logrus.Trace("FetchAnswer end")
	logrus.Debugf("Fetching answer for the problem: %#v", prob)
	if prob.Answer != nil {
		logrus.Debugf("Problem already solved.")
		return prob, nil
	}
	var err error
	logrus.Debug("looping through the engines to try and fetch the answer")
	for _, engine := range engines {
		logrus.Debugf("engine '%T'", engine)
		if prob.Desc == "" && engine.IsInteractiveEngine() {
			return defaultEngine.FetchAnswer(prob)
		}
		prob, err = engine.FetchAnswer(prob)
		if err != nil {
			if _, ok := err.(*qatypes.ValidationError); ok {
				logrus.Errorf("failed to fetch the answer using the engine '%T' . Error: %q", engine, err)
				continue
			}
			logrus.Debugf("failed to fetch the answer using the engine '%T' . Error: %q", engine, err)
			continue
		}
		if prob.Answer != nil {
			prob = changeSelectToInputForOther(prob)
			break
		}
	}
	if err != nil || prob.Answer == nil {
		logrus.Debugf("the answer is nil: '%+v' or there was an error: %q , checking if the problem is valid", prob.Answer, err)
		if err := ValidateProblem(prob); err != nil {
			return prob, fmt.Errorf("the QA problem object is invalid: %+v . Error: %w", prob, err)
		}
		logrus.Debug("loop using interactive engine until we get an answer")
		lastEngine := engines[len(engines)-1]
		if !lastEngine.IsInteractiveEngine() {
			logrus.Debug("there is no interactive engine")
			return prob, fmt.Errorf("failed to fetch the answer for problem: %+v . Error: %w", prob, err)
		}
		for err != nil || prob.Answer == nil {
			prob, err = lastEngine.FetchAnswer(prob)
			if err != nil {
				logrus.Errorf("failed to fetch the answer for the problem: '%s' , trying again. Error: %q", prob.Desc, err)
				continue
			}
			if prob.Answer != nil {
				prob = changeSelectToInputForOther(prob)
			}
		}
	}
	for _, store := range stores {
		store.AddSolution(prob)
	}
	return prob, err
}

// WriteStoresToDisk forces all the stores to write their contents out to disk
func WriteStoresToDisk() error {
	var err error
	for _, store := range stores {
		cerr := store.Write()
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
	if prob.Type == qatypes.SelectSolutionFormType && prob.Answer != nil && prob.Answer.(string) == qatypes.OtherAnswer {
		newDesc := string(qatypes.InputSolutionFormType) + " " + prob.Desc
		newProb, err := qatypes.NewInputProblem(prob.ID, newDesc, nil, "", prob.Validator)
		if err != nil {
			logrus.Fatalf("failed to change the QA select type problem to input type problem: %+v\nError: %q", prob, err)
		}
		return newProb
	}
	return prob
}

// Convenience functions

// FetchStringAnswer asks a input type question and gets a string as the answer
func FetchStringAnswer(probid, desc string, context []string, def string, validator func(interface{}) error) string {
	problem, err := qatypes.NewInputProblem(probid, desc, context, def, validator)
	if err != nil {
		logrus.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		logrus.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Answer.(string)
	if !ok {
		logrus.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Answer, problem.Answer)
	}
	return answer
}

// FetchBoolAnswer asks a confirm type question and gets a boolean as the answer
func FetchBoolAnswer(probid, desc string, context []string, def bool, validator func(interface{}) error) bool {
	problem, err := qatypes.NewConfirmProblem(probid, desc, context, def, validator)
	if err != nil {
		logrus.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		logrus.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Answer.(bool)
	if !ok {
		logrus.Fatalf("Answer is not of the correct type. Expected bool. Actual value is %+v of type %T", problem.Answer, problem.Answer)
	}
	return answer
}

// FetchSelectAnswer asks a select type question and gets a string as the answer
func FetchSelectAnswer(probid, desc string, context []string, def string, options []string, validator func(interface{}) error) string {
	problem, err := qatypes.NewSelectProblem(probid, desc, context, def, options, validator)
	if err != nil {
		logrus.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		logrus.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Answer.(string)
	if !ok {
		logrus.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Answer, problem.Answer)
	}
	return answer
}

// FetchMultiSelectAnswer asks a multi-select type question and gets a slice of strings as the answer
func FetchMultiSelectAnswer(probid, desc string, context, def, options []string, validator func(interface{}) error) []string {
	problem, err := qatypes.NewMultiSelectProblem(probid, desc, context, def, options, validator)
	if err != nil {
		logrus.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		logrus.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, err := common.ConvertInterfaceToSliceOfStrings(problem.Answer)
	if err != nil {
		logrus.Fatalf("Answer is not of the correct type. Expected array of strings. Error: %q", err)
	}
	return answer
}

// FetchPasswordAnswer asks a password type question and gets a string as the answer
func FetchPasswordAnswer(probid, desc string, context []string, validator func(interface{}) error) string {
	problem, err := qatypes.NewPasswordProblem(probid, desc, context, validator)
	if err != nil {
		logrus.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		logrus.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Answer.(string)
	if !ok {
		logrus.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Answer, problem.Answer)
	}
	return answer
}

// FetchMultilineInputAnswer asks a multi-line type question and gets a string as the answer
func FetchMultilineInputAnswer(probid, desc string, context []string, def string, validator func(interface{}) error) string {
	problem, err := qatypes.NewMultilineInputProblem(probid, desc, context, def, validator)
	if err != nil {
		logrus.Fatalf("Unable to create problem. Error: %q", err)
	}
	problem, err = FetchAnswer(problem)
	if err != nil {
		logrus.Fatalf("Unable to fetch answer. Error: %q", err)
	}
	answer, ok := problem.Answer.(string)
	if !ok {
		logrus.Fatalf("Answer is not of the correct type. Expected string. Actual value is %+v of type %T", problem.Answer, problem.Answer)
	}
	return answer
}

// ValidateProblem validates the problem object.
func ValidateProblem(prob qatypes.Problem) error {
	if prob.ID == "" {
		return fmt.Errorf("the QA problem has an empty key: %+v", prob)
	}
	if prob.Desc == "" {
		logrus.Warnf("the QA problem has an empty description: %+v", prob)
	}
	if prob.Hints != nil {
		if _, err := common.ConvertInterfaceToSliceOfStrings(prob.Hints); err != nil {
			return fmt.Errorf("expected the hints to be an array of strings for the QA problem: %+v\nError: %q", prob, err)
		}
	}
	switch prob.Type {
	case qatypes.MultiSelectSolutionFormType:
		if len(prob.Options) == 0 {
			logrus.Debugf("the QA multiselect problem has no options specified: %+v", prob)
			if prob.Default != nil {
				xs, err := common.ConvertInterfaceToSliceOfStrings(prob.Default)
				if err != nil {
					return fmt.Errorf("the QA multiselect problem has a default which is not an array of strings and has no options specified: %+v", prob)
				}
				if len(xs) > 0 {
					return fmt.Errorf("the QA multiselect problem has a default set but no options specified: %+v", prob)
				}
			}
			return nil
		}
		if prob.Default != nil {
			defaults, err := common.ConvertInterfaceToSliceOfStrings(prob.Default)
			if err != nil {
				return fmt.Errorf("expected the defaults to be an array of strings for the QA multiselect problem: %+v\nError: %q", prob, err)
			}
			for _, def := range defaults {
				if !common.IsPresent(prob.Options, def) {
					return fmt.Errorf("one of the defaults [%s] is not present in the options for the QA multiselect problem: %+v", def, prob)
				}
			}
		}
	case qatypes.SelectSolutionFormType:
		if len(prob.Options) == 0 {
			return fmt.Errorf("the QA select problem has no options specified: %+v", prob)
		}
		if prob.Default != nil {
			def, ok := prob.Default.(string)
			if !ok {
				return fmt.Errorf("expected the default to be a string for the QA select problem: %+v", prob)
			}
			if !common.IsPresent(prob.Options, def) {
				return fmt.Errorf("the default [%s] is not present in the options for the QA select problem: %+v", def, prob)
			}
		}
	case qatypes.ConfirmSolutionFormType:
		if len(prob.Options) > 0 {
			logrus.Warnf("options are not supported for the QA confirm question type: %+v", prob)
		}
		if prob.Default != nil {
			if _, ok := prob.Default.(bool); !ok {
				return fmt.Errorf("expected the default to be a bool for the QA confirm problem: %+v", prob)
			}
		}
	case qatypes.InputSolutionFormType, qatypes.MultilineInputSolutionFormType, qatypes.PasswordSolutionFormType:
		if len(prob.Options) > 0 {
			logrus.Warnf("options are not supported for the QA input/multiline/password question types: %+v", prob)
		}
		if prob.Default != nil {
			if prob.Type == qatypes.PasswordSolutionFormType {
				logrus.Warnf("default is not supported for the QA password question type: %+v", prob)
			} else {
				if _, ok := prob.Default.(string); !ok {
					return fmt.Errorf("expected the default to be a string for the QA input/multiline problem: %+v", prob)
				}
			}
		}
	default:
		return fmt.Errorf("unknown QA problem type: %+v", prob)
	}
	return nil
}
