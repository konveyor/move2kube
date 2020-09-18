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
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

// Engine defines interface for qa engines
type Engine interface {
	StartEngine() error
	FetchAnswer(prob qatypes.Problem) (ans qatypes.Problem, err error)
}

var (
	engines    []Engine
	writeCache qatypes.Cache
)

// StartEngine starts the QA Engines
func StartEngine(qaskip bool, qaport int, qadisablecli bool) {
	if qaskip {
		e := NewDefaultEngine()
		err := e.StartEngine()
		if err != nil {
			log.Errorf("Ignoring engine %T due to error : %s", e, err)
		} else {
			engines = append(engines, e)
		}
	} else if !qadisablecli {
		e := NewCliEngine()
		err := e.StartEngine()
		if err != nil {
			log.Errorf("Ignoring engine %T due to error : %s", e, err)
		} else {
			engines = append(engines, e)
		}
	} else {
		e := NewHTTPRESTEngine(qaport)
		err := e.StartEngine()
		if err != nil {
			log.Errorf("Ignoring engine %T due to error : %s", e, err)
		} else {
			engines = append(engines, e)
		}
	}
}

// AddCaches adds cache responders
func AddCaches(cacheFiles []string) {
	cengines := []Engine{}
	for _, cacheFile := range cacheFiles {
		e := NewCacheEngine(cacheFile)
		err := e.StartEngine()
		if err != nil {
			log.Errorf("Ignoring engine %T due to error : %s", e, err)
		} else {
			cengines = append(cengines, e)
		}
	}
	engines = append(cengines, engines...)
}

// FetchAnswer fetches the answer for the question
func FetchAnswer(prob qatypes.Problem) (ans qatypes.Problem, err error) {
	for _, e := range engines {
		ans, err = e.FetchAnswer(prob)
		if err != nil {
			log.Warnf("Error while fetching answer using engine %s : %s", e, err)
		} else if ans.Resolved {
			break
		}
	}
	if !ans.Resolved {
		for {
			ans, err = engines[len(engines)-1].FetchAnswer(prob)
			if err != nil {
				log.Fatalf("Unable to get answer to %s : %s", ans.Desc, err)
			}
			if ans.Resolved {
				break
			}
		}
	}
	if err == nil && ans.Resolved {
		writeCache.AddProblemSolutionToCache(ans)
	} else if err != nil {
		log.Errorf("Unable to fetch answer : %s", err)
	}
	return ans, err
}

// SetWriteCache sets the write cache
func SetWriteCache(cacheFile string) error {
	dirpath := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dirpath, common.DefaultDirectoryPermission); err != nil {
		// Create the qacache directory if it is missing
		log.Errorf("Failed to create the qacache directory at path %q. Error: %q", dirpath, err)
		return err
	}

	writeCache = qatypes.NewCache(cacheFile)
	return writeCache.Write()
}
