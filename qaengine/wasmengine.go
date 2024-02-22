/*
 *  Copyright IBM Corporation 2024
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
	"encoding/json"
	"fmt"
	"strings"
	"unsafe"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/common/deepcopy"
	qatypes "github.com/konveyor/move2kube-wasm/types/qaengine"
	"github.com/sirupsen/logrus"
)

// WasmEngine handles qa using HTTP REST services
type WasmEngine struct {
	port           int
	currentProblem qatypes.Problem
	problemChan    chan qatypes.Problem
	answerChan     chan qatypes.Problem
}

const (
	// maxOutputLength TODO: this is hardcoded since we can't export myAllocate yet
	maxOutputLength uint32 = 65536
)

//go:wasmimport mym2kmodule ask_question
func ask_question(
	questionPtr unsafe.Pointer,
	questionLen uint32,
	outPtr unsafe.Pointer,
) (outLen int32)

// AskQuestion asks a question using the WASM host function
func AskQuestion(prob qatypes.Problem) (qatypes.Problem, error) {
	quesJsonBytes, err := json.Marshal(prob)
	if err != nil {
		return prob, fmt.Errorf("failed to marshal the question as json. Error: %w", err)
	}
	quesJsonBytesPtr := unsafe.Pointer(&quesJsonBytes[0])
	newArr := make([]byte, maxOutputLength)
	ansJsonBytesPtr := &newArr[0]
	ansJsonBytesLen := ask_question(
		quesJsonBytesPtr,
		uint32(len(quesJsonBytes)),
		unsafe.Pointer(ansJsonBytesPtr),
	)
	if ansJsonBytesLen < 0 {
		return prob, fmt.Errorf("failed to ask the question, wasm host function returned an error")
	}
	ansJsonBytes := unsafe.Slice(ansJsonBytesPtr, ansJsonBytesLen)
	if err := json.Unmarshal(ansJsonBytes, &prob); err != nil {
		return prob, fmt.Errorf("failed to unmarshal the answer as json. Error: %w", err)
	}
	return prob, nil
}

// NewWasmEngine creates a new instance of WASM QA engine
func NewWasmEngine() Engine {
	return &WasmEngine{}
}

// StartEngine starts the QA Engine
func (h *WasmEngine) StartEngine() error {
	return nil
}

// IsInteractiveEngine returns true if the engine interacts with the user
func (*WasmEngine) IsInteractiveEngine() bool {
	return true
}

// FetchAnswer fetches the answer using a WASM host function
func (*WasmEngine) FetchAnswer(prob qatypes.Problem) (qatypes.Problem, error) {
	logrus.Trace("WasmEngine.FetchAnswer start")
	defer logrus.Trace("WasmEngine.FetchAnswer end")
	if err := ValidateProblem(prob); err != nil {
		return prob, fmt.Errorf("the QA problem object is invalid. Error: %w", err)
	}
	if prob.Answer != nil {
		return prob, nil
	}
	logrus.Debugf("Passing problem to WASM QA Engine ID: '%s' desc: '%s'", prob.ID, prob.Desc)
	logrus.Debugf("sent the current question to the wasm host function: %+v", prob)
	prob, err := AskQuestion(prob)
	if err != nil {
		return prob, fmt.Errorf("failed to ask a question. Error: %w", err)
	}
	logrus.Debugf("received a solution from the wasm host function: %+v", prob)
	if prob.Answer == nil {
		return prob, fmt.Errorf("failed to resolve the QA problem: %+v", prob)
	}
	if prob.Type != qatypes.MultiSelectSolutionFormType {
		return prob, nil
	}
	otherAnsPresent := false
	ans, err := common.ConvertInterfaceToSliceOfStrings(prob.Answer)
	if err != nil {
		return prob, fmt.Errorf("failed to convert the answer from an interface to a slice of strings. Error: %w", err)
	}
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
		multilineProb := deepcopy.DeepCopy(prob).(qatypes.Problem)
		multilineProb.Type = qatypes.MultilineInputSolutionFormType
		multilineProb.Default = ""
		// h.problemChan <- multilineProb
		multilineProb, err := AskQuestion(multilineProb)
		if err != nil {
			return multilineProb, fmt.Errorf("failed to ask the 'other' question. Error: %w", err)
		}
		// multilineProb = <-h.answerChan
		multilineAns = multilineProb.Answer.(string)
		for _, lineAns := range strings.Split(multilineAns, "\n") {
			lineAns = strings.TrimSpace(lineAns)
			if lineAns != "" {
				newAns = common.AppendIfNotPresent(newAns, lineAns)
			}
		}
	}
	prob.Answer = newAns
	return prob, nil
}
