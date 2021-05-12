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

package transformations

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	"github.com/konveyor/move2kube/internal/starlark/gettransformdata"
	"github.com/konveyor/move2kube/internal/starlark/types"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
)

// Question answer functions

// AskDynamicQuestion asks a dynamic question
func AskDynamicQuestion(questionObjI interface{}) (interface{}, error) {
	log.Trace("start AskDynamicQuestion")
	defer log.Trace("end AskDynamicQuestion")

	questionObj, ok := questionObjI.(types.MapT)
	if !ok {
		return nil, fmt.Errorf("expected question to be of map type. Actual value is %+v of type %T", questionObjI, questionObjI)
	}
	return askQuestion(questionObj)
}

func convertMapTToProblem(questionObj types.MapT) (qatypes.Problem, error) {
	log.Trace("start convertMapTToProblem")
	defer log.Trace("end convertMapTToProblem")

	prob := qatypes.Problem{}
	quesBytes, err := json.Marshal(questionObj)
	if err != nil {
		return prob, fmt.Errorf("failed to marshal the question object into json: %+v\nError: %q", questionObj, err)
	}
	if err := json.Unmarshal(quesBytes, &prob); err != nil {
		return prob, fmt.Errorf("failed to unmarshal the question json into a QA problem struct: %s\nError: %q", string(quesBytes), err)
	}

	// key
	if prob.ID == "" {
		return prob, fmt.Errorf("the key 'id' is missing from the question object %+v", questionObj)
	}
	if !strings.HasPrefix(prob.ID, common.BaseKey) {
		prob.ID = common.BaseKey + common.Delim + prob.ID
	}

	// type
	if _, ok := questionObj["type"]; !ok {
		prob.Type = qatypes.InputSolutionFormType
	}

	return prob, nil
}

func askQuestion(questionObj types.MapT) (interface{}, error) {
	log.Trace("start askQuestion")
	defer log.Trace("end askQuestion")

	prob, err := convertMapTToProblem(questionObj)
	if err != nil {
		log.Errorf("failed to make a QA problem object using the given information: %+v Error: %q", questionObj, err)
		return nil, err
	}
	resolved, err := qaengine.FetchAnswer(prob)
	if err != nil {
		log.Fatalf("failed to ask the question. Error: %q", err)
	}
	return resolved.Answer, nil
}

// GetTransformsFromPathsUsingDefaults returns starlark transforms using this package's QA handlers
func GetTransformsFromPathsUsingDefaults(transformPaths []string) ([]types.TransformT, error) {
	return gettransformdata.GetTransformsFromPaths(transformPaths, AskDynamicQuestion)
}
