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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/gorilla/mux"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	"github.com/phayes/freeport"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

const (
	problemsURLPrefix        = "/problems"
	currentProblemURLPrefix  = problemsURLPrefix + "/current"
	currentSolutionURLPrefix = currentProblemURLPrefix + "/solution"
)

// HTTPRESTEngine handles qa using HTTP REST services
type HTTPRESTEngine struct {
	port           int
	currentProblem qatypes.Problem
	problemChan    chan qatypes.Problem
	answerChan     chan qatypes.Problem
}

// NewHTTPRESTEngine creates a new instance of Http REST engine
func NewHTTPRESTEngine(qaport int) Engine {
	e := new(HTTPRESTEngine)
	e.port = qaport
	e.currentProblem = qatypes.Problem{ID: 0, Resolved: true}
	e.problemChan = make(chan qatypes.Problem)
	e.answerChan = make(chan qatypes.Problem)
	return e
}

// StartEngine starts the QA Engine
func (h *HTTPRESTEngine) StartEngine() error {
	if h.port == 0 {
		var err error
		h.port, err = freeport.GetFreePort()
		if err != nil {
			return fmt.Errorf("Unable to find a free port : %s", err)
		}
	}
	// Create the REST router.
	r := mux.NewRouter()
	r.HandleFunc(currentProblemURLPrefix, h.problemHandler).Methods("GET")
	r.HandleFunc(currentSolutionURLPrefix, h.solutionHandler).Methods("POST")

	http.Handle("/", r)
	qaportstr := cast.ToString(h.port)

	listener, err := net.Listen("tcp", ":"+qaportstr)
	if err != nil {
		return fmt.Errorf("Unable to listen on port %d : %s", h.port, err)
	}
	go func(listener net.Listener) {
		err := http.Serve(listener, nil)
		if err != nil {
			log.Fatalf("Unable to start qa server : %s", err)
		}
	}(listener)
	log.Info("Started QA engine on: localhost:" + qaportstr)
	return nil
}

// FetchAnswer fetches the answer using a REST service
func (h *HTTPRESTEngine) FetchAnswer(prob qatypes.Problem) (ans qatypes.Problem, err error) {
	if prob.ID == 0 {
		prob.Resolved = true
	}
	if !prob.Resolved {
		log.Debugf("Passing problem to HTTP REST QA Engine ID: %d, desc: %s", prob.ID, prob.Desc)
		h.problemChan <- prob
		prob = <-h.answerChan
		if !prob.Resolved {
			return prob, fmt.Errorf("Unable to resolve question %s", prob.Desc)
		}
	}
	return prob, nil
}

// problemHandler returns the current problem being handled
func (h *HTTPRESTEngine) problemHandler(w http.ResponseWriter, r *http.Request) {
	log.Debug("Looking for a problem fron HTTP REST service")
	// if currently problem is resolved
	if h.currentProblem.Resolved || h.currentProblem.ID == 0 {
		// Pick the next problem off the channel
		h.currentProblem = <-h.problemChan
	}
	log.Debugf("QA Engine serves problem id: %d, desc: %s", h.currentProblem.ID, h.currentProblem.Desc)
	// Send the problem to the request.
	_ = json.NewEncoder(w).Encode(h.currentProblem)
}

// solutionHandler accepts solution for a single open problem.
func (h *HTTPRESTEngine) solutionHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("QA Engine reading solution: %s", r.Body)
	// Read out the solution
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errstr := fmt.Sprintf("Error in reading posted solution: %s", err)
		http.Error(w, "errstr", http.StatusInternalServerError)
		log.Errorf(errstr)
	}
	var sol []string
	err = json.Unmarshal(body, &sol)
	if err != nil {
		errstr := fmt.Sprintf("Error in un-marshalling solution in QA engine: %s", err)
		http.Error(w, errstr, http.StatusInternalServerError)
		log.Errorf(errstr)
	}
	log.Debugf("QA Engine receives solution: %s", sol)
	err = h.currentProblem.SetAnswer(sol)
	if err != nil {
		errstr := fmt.Sprintf("Unsuitable answer : %s", err)
		http.Error(w, errstr, http.StatusInternalServerError)
		log.Errorf(errstr)
	} else {
		h.answerChan <- h.currentProblem
	}
}
