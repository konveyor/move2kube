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
	"fmt"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/qaengine"
	"github.com/konveyor/move2kube/internal/starlark/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
)

var (
	questions = map[string]types.MapT{}
)

// Question answer functions

// AskStaticQuestion asks a static question
func AskStaticQuestion(questionObjI interface{}) error {
	log.Trace("start myStaticAskQuestion")
	defer log.Trace("end myStaticAskQuestion")
	questionObj, ok := questionObjI.(types.MapT)
	if !ok {
		return fmt.Errorf("Expected questions to be of map type. Actual value is %+v of type %T", questionObjI, questionObjI)
	}
	qakeyI, ok := questionObj["key"]
	if !ok {
		return fmt.Errorf("The key 'key' is missing from the question object %+v", questionObj)
	}
	qakey, ok := qakeyI.(string)
	if !ok {
		return fmt.Errorf("The key 'key' is not a string. The question object %+v", questionObj)
	}
	if _, ok := questionObj["type"]; !ok {
		questionObj["type"] = "string"
	}
	questions[qakey] = questionObj
	return nil
}

// AnswerFn answers a static question
func AnswerFn(qakey string) (interface{}, error) {
	log.Trace("start myAnswerFn")
	defer log.Trace("end myAnswerFn")
	questionObj, ok := questions[qakey]
	if !ok {
		return nil, fmt.Errorf("There is no question with the key: %s", qakey)
	}
	return askQuestion(questionObj)
}

// AskDynamicQuestion asks a dynamic question
func AskDynamicQuestion(questionObjI interface{}) (interface{}, error) {
	questionObj, ok := questionObjI.(types.MapT)
	if !ok {
		return nil, fmt.Errorf("Expected question to be of map type. Actual value is %+v of type %T", questionObjI, questionObjI)
	}
	if _, ok := questionObj["type"]; !ok {
		questionObj["type"] = "string"
	}
	return askQuestion(questionObj)
}

func askQuestion(questionObj types.MapT) (interface{}, error) {
	log.Trace("start commonStuff")
	defer log.Trace("end commonStuff")
	qakeyI, ok := questionObj["key"]
	if !ok {
		return nil, fmt.Errorf("The key 'key' is missing from the question object %+v", questionObj)
	}
	qakey, ok := qakeyI.(string)
	if !ok {
		return nil, fmt.Errorf("The key 'key' is not a string. The question object %+v", questionObj)
	}
	quesTypeI, ok := questionObj["type"]
	if !ok {
		return nil, fmt.Errorf("The key 'type' is missing from the question object %+v", questionObj)
	}
	quesType, ok := quesTypeI.(string)
	if !ok {
		return nil, fmt.Errorf("The key 'type' is not a string. The question object %+v", questionObj)
	}
	descI, ok := questionObj["description"]
	if !ok {
		return nil, fmt.Errorf("The key 'description' is missing from the question object %+v", questionObj)
	}
	desc, ok := descI.(string)
	if !ok {
		return nil, fmt.Errorf("The key 'description' is not a string. The question object %+v", questionObj)
	}
	hints := []string{}
	hintsI, ok := questionObj["hints"]
	if ok {
		hintIs, ok := hintsI.([]interface{})
		if !ok {
			return nil, fmt.Errorf("The key 'hints' is not an array. The question object %+v", questionObj)
		}
		for _, hintI := range hintIs {
			hint, ok := hintI.(string)
			if !ok {
				return nil, fmt.Errorf("The hints are not all strings. The question object %+v", questionObj)
			}
			hints = append(hints, hint)
		}
	}

	// add prefix
	// log.Infof("IMPO %+v %+v %+v %+v", qakey, desc, hints, defaultAnswer)
	// answers[qakey] = fmt.Sprintf("wip: [%s]", qakey)
	qakey = common.BaseKey + common.Delim + qakey

	switch quesType {
	case "string", "multiline":
		defaultAnswer := ""
		defaultAnswerI, ok := questionObj["default"]
		if ok {
			defaultAnswer, ok = defaultAnswerI.(string)
			if !ok {
				return nil, fmt.Errorf("The key 'default' is not a string. The question object %+v", questionObj)
			}
		}
		if quesType == "multiline" {
			return qaengine.FetchMultilineAnswer(qakey, desc, hints, defaultAnswer), nil
		}
		return qaengine.FetchStringAnswer(qakey, desc, hints, defaultAnswer), nil
	case "password":
		return qaengine.FetchPasswordAnswer(qakey, desc, hints), nil
	case "bool":
		defaultAnswer := ""
		defaultAnswerI, ok := questionObj["default"]
		if ok {
			defaultAnswer, ok = defaultAnswerI.(string)
			if !ok {
				return nil, fmt.Errorf("The key 'default' is not a string. The question object %+v", questionObj)
			}
		}
		defaultAnswerBool, err := cast.ToBoolE(defaultAnswer)
		if err != nil {
			return nil, fmt.Errorf("The default answer is not a boolean. The question object %+v", questionObj)
		}
		return qaengine.FetchBoolAnswer(qakey, desc, hints, defaultAnswerBool), nil
	case "select", "multiselect":
		options := []string{}
		optionsI, ok := questionObj["options"]
		if !ok {
			return nil, fmt.Errorf("The key 'options' is missing from multiselect answer. The question object %+v", questionObj)
		}
		optionIs, ok := optionsI.([]interface{})
		if !ok {
			return nil, fmt.Errorf("The key 'options' is not an array. The question object %+v", questionObj)
		}
		for _, optionI := range optionIs {
			option, ok := optionI.(string)
			if !ok {
				return nil, fmt.Errorf("The hints are not all strings. The question object %+v", questionObj)
			}
			options = append(options, option)
		}
		if quesType == "select" {
			defaultAnswer := ""
			defaultAnswerI, ok := questionObj["default"]
			if ok {
				defaultAnswer, ok = defaultAnswerI.(string)
				if !ok {
					return nil, fmt.Errorf("The key 'default' is not a string. The question object %+v", questionObj)
				}
			}
			return qaengine.FetchSelectAnswer(qakey, desc, hints, defaultAnswer, options), nil
		}
		defaultAnswer := []string{}
		defaultAnswerI, ok := questionObj["default"]
		if ok {
			defaultAnswerIs, ok := defaultAnswerI.([]interface{})
			if !ok {
				return nil, fmt.Errorf("The key 'default' is not a array. The question object %+v", questionObj)
			}
			for _, defI := range defaultAnswerIs {
				def, ok := defI.(string)
				if !ok {
					return nil, fmt.Errorf("The default answers are not all strings. The question object %+v", questionObj)
				}
				defaultAnswer = append(defaultAnswer, def)
			}
		}
		return qaengine.FetchMultiSelectAnswer(qakey, desc, hints, defaultAnswer, options), nil
	}
	return nil, fmt.Errorf("Unknown question type %s . Question object is: %+v", quesType, questionObj)
}
