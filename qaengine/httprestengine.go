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

//import (
//	"encoding/json"
//	"fmt"
//	"net"
//	"net/http"
//	"strings"
//
//	"github.com/gorilla/mux"
//	"github.com/konveyor/move2kube/common"
//	"github.com/konveyor/move2kube/common/deepcopy"
//	qatypes "github.com/konveyor/move2kube/types/qaengine"
//	"github.com/phayes/freeport"
//	"github.com/sirupsen/logrus"
//	"github.com/spf13/cast"
//)
//
//// HTTPRESTEngine handles qa using HTTP REST services
//type HTTPRESTEngine struct {
//	port           int
//	currentProblem qatypes.Problem
//	problemChan    chan qatypes.Problem
//	answerChan     chan qatypes.Problem
//}
//
//const (
//	problemsURLPrefix        = "/problems"
//	currentProblemURLPrefix  = problemsURLPrefix + "/current"
//	currentSolutionURLPrefix = currentProblemURLPrefix + "/solution"
//)
//
//// NewHTTPRESTEngine creates a new instance of Http REST engine
//func NewHTTPRESTEngine(qaport int) Engine {
//	return &HTTPRESTEngine{
//		port:           qaport,
//		currentProblem: qatypes.Problem{ID: "", Answer: ""},
//		problemChan:    make(chan qatypes.Problem),
//		answerChan:     make(chan qatypes.Problem),
//	}
//}
//
//// StartEngine starts the QA Engine
//func (h *HTTPRESTEngine) StartEngine() error {
//	if h.port == 0 {
//		var err error
//		h.port, err = freeport.GetFreePort()
//		if err != nil {
//			return fmt.Errorf("unable to find a free port : %s", err)
//		}
//	}
//	// Create the REST router.
//	r := mux.NewRouter()
//	r.HandleFunc(currentProblemURLPrefix, h.getQuestionHandler).Methods("GET")
//	r.HandleFunc(currentSolutionURLPrefix, h.postSolutionHandler).Methods("POST")
//
//	http.Handle("/", r)
//	qaportstr := cast.ToString(h.port)
//
//	listener, err := net.Listen("tcp", ":"+qaportstr)
//	if err != nil {
//		return fmt.Errorf("unable to listen on port %d : %s", h.port, err)
//	}
//	go func(listener net.Listener) {
//		err := http.Serve(listener, nil)
//		if err != nil {
//			logrus.Fatalf("Unable to start qa server : %s", err)
//		}
//	}(listener)
//	logrus.Info("Started QA engine on: localhost:" + qaportstr)
//	return nil
//}
//
//// IsInteractiveEngine returns true if the engine interacts with the user
//func (*HTTPRESTEngine) IsInteractiveEngine() bool {
//	return true
//}
//
//// FetchAnswer fetches the answer using a REST service
//func (h *HTTPRESTEngine) FetchAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
//	logrus.Trace("HTTPRESTEngine.FetchAnswer start")
//	defer logrus.Trace("HTTPRESTEngine.FetchAnswer end")
//	if err := ValidateProblem(prob); err != nil {
//		return prob, fmt.Errorf("the QA problem object is invalid. Error: %w", err)
//	}
//	if prob.Answer != nil {
//		return prob, nil
//	}
//	logrus.Debugf("Passing problem to HTTP REST QA Engine ID: '%s' desc: '%s'", prob.ID, prob.Desc)
//	h.problemChan <- prob
//	logrus.Debugf("sent the current question into the problem channel: %+v", prob)
//	prob = <-h.answerChan
//	logrus.Debugf("received a solution from the problem channel: %+v", prob)
//	if prob.Answer == nil {
//		return prob, fmt.Errorf("failed to resolve the QA problem: %+v", prob)
//	}
//	if prob.Type != qatypes.MultiSelectSolutionFormType {
//		return prob, nil
//	}
//	otherAnsPresent := false
//	ans, err := common.ConvertInterfaceToSliceOfStrings(prob.Answer)
//	if err != nil {
//		return prob, fmt.Errorf("failed to convert the answer from an interface to a slice of strings. Error: %w", err)
//	}
//	newAns := []string{}
//	for _, a := range ans {
//		if a == qatypes.OtherAnswer {
//			otherAnsPresent = true
//		} else {
//			newAns = append(newAns, a)
//		}
//	}
//	if otherAnsPresent {
//		multilineAns := ""
//		multilineProb := deepcopy.DeepCopy(prob).(qatypes.Problem)
//		multilineProb.Type = qatypes.MultilineInputSolutionFormType
//		multilineProb.Default = ""
//		h.problemChan <- multilineProb
//		multilineProb = <-h.answerChan
//		multilineAns = multilineProb.Answer.(string)
//		for _, lineAns := range strings.Split(multilineAns, "\n") {
//			lineAns = strings.TrimSpace(lineAns)
//			if lineAns != "" {
//				newAns = common.AppendIfNotPresent(newAns, lineAns)
//			}
//		}
//	}
//	prob.Answer = newAns
//	return prob, nil
//}
//
//// getQuestionHandler blocks until it gets a question and returns it as json.
//func (h *HTTPRESTEngine) getQuestionHandler(w http.ResponseWriter, r *http.Request) {
//	logrus.Trace("problemHandler start")
//	defer logrus.Trace("problemHandler end")
//	logrus.Debug("Looking for a problem fron HTTP REST service")
//	// if currently problem is resolved
//	if h.currentProblem.Answer != nil || h.currentProblem.ID == "" {
//		// Pick the next problem off the channel
//		h.currentProblem = <-h.problemChan
//	}
//	logrus.Debugf("QA Engine serves problem id: '%s' desc: '%s'", h.currentProblem.ID, h.currentProblem.Desc)
//	// Send the problem to the request.
//	w.Header().Set("Content-Type", "application/json")
//	w.WriteHeader(http.StatusOK)
//	if err := json.NewEncoder(w).Encode(h.currentProblem); err != nil {
//		logrus.Errorf("failed to encode the current problem as json and send the response. Error: %q", err)
//		return
//	}
//}
//
//// postSolutionHandler accepts solution for the current question.
//func (h *HTTPRESTEngine) postSolutionHandler(w http.ResponseWriter, r *http.Request) {
//	logrus.Trace("solutionHandler start")
//	defer logrus.Trace("solutionHandler end")
//	logrus.Debugf("QA Engine reading solution: %+v", r.Body)
//	// Read out the solution
//	var prob qatypes.Problem
//	if err := json.NewDecoder(r.Body).Decode(&prob); err != nil {
//		newErr := fmt.Errorf("failed to decode the request body as solution json. Error: %w", err)
//		http.Error(w, newErr.Error(), http.StatusBadRequest)
//		logrus.Error(newErr.Error())
//		return
//	}
//	logrus.Debugf("QA Engine received the solution: %+v", prob)
//	if h.currentProblem.ID != prob.ID {
//		err := fmt.Errorf("the solution's problem ID doesn't match the current problem. Expected: '%s' Actual '%s'", h.currentProblem.ID, prob.ID)
//		http.Error(w, err.Error(), http.StatusNotAcceptable)
//		logrus.Error(err.Error())
//		return
//	}
//	if err := h.currentProblem.SetAnswer(prob.Answer, true); err != nil {
//		newErr := fmt.Errorf("failed to set the given solution as the answer. Error: %w", err)
//		http.Error(w, newErr.Error(), http.StatusNotAcceptable)
//		logrus.Error(newErr.Error())
//		return
//	}
//	logrus.Debugf("QA Engine set the given solution as the answer: %+v", h.currentProblem)
//	w.WriteHeader(http.StatusNoContent)
//	go func() { h.answerChan <- h.currentProblem }()
//}
