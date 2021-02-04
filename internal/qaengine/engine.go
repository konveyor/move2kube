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
		return fmt.Errorf("Failed to start the engine: %T\n%v\nError: %s", e, e, err)
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
	var err error
	log.Debugf("Fetching answer for problem:\n%v\n", prob)
	if prob.Resolved {
		log.Debugf("Problem already solved.")
		return prob, nil
	}
	for _, e := range engines {
		prob, err = e.FetchAnswer(prob)
		if err != nil {
			log.Debugf("Error while fetching answer using engine %+v Error: %q", e, err)
			continue
		}
		if prob.Resolved {
			prob = changeSelectToInputForOther(prob)
			break
		}
	}
	if err != nil || !prob.Resolved {
		// loop using interactive engine until we get an answer
		lastEngine := engines[len(engines)-1]
		if !lastEngine.IsInteractiveEngine() {
			return prob, fmt.Errorf("Failed to fetch the answer for problem\n%+v\nError: %q", prob, err)
		}
		for err != nil || !prob.Resolved {
			prob, err = lastEngine.FetchAnswer(prob)
			if err != nil {
				log.Errorf("Unable to get answer to %s Error: %q", prob.Desc, err)
				continue
			}
			if prob.Resolved {
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
	if prob.Solution.Type == qatypes.SelectSolutionFormType && len(prob.Solution.Answer) > 0 && prob.Solution.Answer[0] == qatypes.OtherAnswer {
		prob.Solution.Type = qatypes.InputSolutionFormType
		prob.Desc = string(qatypes.InputSolutionFormType) + " " + prob.Desc
		prob.Solution.Answer = []string{}
		prob.Resolved = false
	}
	return prob
}
