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
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/mikefarah/yq/v4/pkg/yqlib"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	logging "gopkg.in/op/go-logging.v1"
	"gopkg.in/yaml.v3"
)

type mapT = map[string]interface{}

// nullLogBackend is a used to prevent yq from logging to stdout
type nullLogBackend struct{}

// Config stores the answers in a yaml file
type Config struct {
	configFiles   []string
	configStrings []string
	yamlMap       mapT
	writeYamlMap  mapT
	OutputPath    string
}

// Implement the Store interface

// Load loads and merges config files and strings
func (c *Config) Load() (err error) {
	log.Debugf("Config.Load")
	yamlDatas := []string{}
	// config files specified later override earlier config files
	for _, configFile := range c.configFiles {
		yamlData, err := ioutil.ReadFile(configFile)
		if err != nil {
			log.Errorf("Failed to read the config file %s Error: %q", configFile, err)
			continue
		}
		yamlDatas = append(yamlDatas, string(yamlData))
	}
	// config strings override config files
	// config strings specified later override earlier config strings
	for i, configString := range c.configStrings {
		if len(configString) == 0 {
			log.Errorf("The %dth config string is empty", i)
			continue
		}
		if !strings.HasPrefix(configString, common.Delim) {
			// given move2kube.services change to .move2kube.services for yq parser.
			configString = common.Delim + configString
		}
		log.Debugf("config store load configString: [%s]", configString)
		yamlData, err := generateYAMLFromExpression(configString)
		if err != nil {
			log.Errorf("Unable to parse the config string %s Error: %q", configString, err)
			continue
		}
		log.Debugf("after parsing the yamlData is:\n%s", yamlData)
		yamlDatas = append(yamlDatas, yamlData)
	}
	c.yamlMap, err = mergeYAMLDatasIntoMap(yamlDatas)
	c.writeYamlMap = mapT{}
	return err
}

func (c *Config) convertAnswer(p Problem, value interface{}) (Problem, error) {
	if p.Solution.Type != MultiSelectSolutionFormType {
		// value should be a scalar, convert it to string
		valueStr, err := cast.ToStringE(value)
		if err != nil {
			log.Errorf("Failed to cast value %v of type %T to string. Error: %q", value, value, err)
			return p, err
		}
		err = p.SetAnswer([]string{valueStr})
		return p, err
	}
	// value should be an array, convert it to []string
	valueV := reflect.ValueOf(value)
	if !valueV.IsValid() || valueV.IsZero() || valueV.Kind() != reflect.Slice {
		err := fmt.Errorf("Expected to find an array. Actual value %v is of type %T", value, value)
		log.Error(err)
		return p, err
	}
	valueStrs := []string{}
	for i := 0; i < valueV.Len(); i++ {
		v := valueV.Index(i).Interface()
		valueStr, err := cast.ToStringE(v)
		if err != nil {
			log.Errorf("Failed to cast value %v of type %T to string. Error: %q", v, v, err)
			return p, err
		}
		valueStrs = append(valueStrs, valueStr)
	}
	err := p.SetAnswer(valueStrs)
	return p, err
}

func (c *Config) normalGetSolution(p Problem) (Problem, error) {
	key := p.ID
	value, ok := get(key, c.yamlMap)
	if ok {
		return c.convertAnswer(p, value)
	}
	// starting from 2nd last subkey replace with match all selector *
	// Example: Given a.b.c.d.e this matches a.b.c.*.e, then a.b.*.d.e, then a.*.c.d.e
	subKeys := getSubKeys(key)
	for idx := len(subKeys) - 2; idx > 0; idx-- {
		baseKey := strings.Join(subKeys[:idx], common.Delim)
		lastKeySegment := strings.Join(subKeys[idx+1:], common.Delim)
		newKey := baseKey + common.Delim + common.MatchAll + common.Delim + lastKeySegment
		v, ok := get(newKey, c.yamlMap)
		if ok {
			return c.convertAnswer(p, v)
		}
	}
	return p, fmt.Errorf("no answer found in the config for the problem:%+v", p)
}

func (c *Config) specialGetSolution(p Problem) (Problem, error) {
	noAns := fmt.Errorf("no answer found in the config for the problem:%+v", p)
	key := p.ID
	idx := strings.LastIndex(key, common.Special)
	baseKey := key[:idx-len(common.Delim)]
	lastKeySegment := key[idx+len(common.Special)+len(common.Delim):]
	if baseKey == "" {
		return p, noAns
	}
	baseValue, ok := get(baseKey, c.yamlMap)
	if !ok {
		return p, noAns
	}
	baseValueMap, ok := baseValue.(mapT)
	if !ok {
		return p, noAns
	}
	// Given 'a.b.[].d' there should be at least one key 'c' such that 'a.b.c.d' exists.
	// Note that 'c' in 'a.b.c.d' does not have to be a valid option for the question.
	// We take the mere existence of 'a.b.c.d' to indicate that the user wants to skip the question.
	atleastOneKeyHasLastSegment := false
	for k := range baseValueMap {
		newK := baseKey + common.Delim + k + common.Delim + lastKeySegment
		lastKeySegmentValue, ok := get(newK, c.yamlMap)
		if !ok {
			continue
		}
		if _, ok := lastKeySegmentValue.(bool); !ok {
			log.Debugf("Found key %s in the config but the corresponding value is not a boolean. Actual value %v is of type %T", newK, lastKeySegmentValue, lastKeySegmentValue)
			return p, noAns
		}
		atleastOneKeyHasLastSegment = true
		break
	}
	if !atleastOneKeyHasLastSegment {
		return p, noAns
	}
	// found at least one key, so we will try to answer using the defaults
	selectedOptions := []string{}
	for _, option := range p.Solution.Options {
		isOptionSelected := true
		newKey := baseKey + common.Delim + option + common.Delim + lastKeySegment
		if newValue, ok := get(newKey, c.yamlMap); ok {
			isOptionSelected, ok = newValue.(bool)
			if !ok {
				return p, fmt.Errorf("Error occurred in special case for multiselect problems. Expected key %s to have boolean value. Actual value is %v of type %T", newKey, newValue, newValue)
			}
		}
		if isOptionSelected {
			selectedOptions = append(selectedOptions, option)
		}
	}
	err := p.SetAnswer(selectedOptions)
	return p, err
}

// GetSolution reads a solution from the config
func (c *Config) GetSolution(p Problem) (Problem, error) {
	if strings.Contains(p.ID, common.Special) {
		if p.Solution.Type != MultiSelectSolutionFormType {
			return p, fmt.Errorf("cannot use the %s selector with non multi select problems:%+v", common.Special, p)
		}
		return c.specialGetSolution(p)
	}
	return c.normalGetSolution(p)
}

// Write writes the config to disk
func (c *Config) Write() error {
	log.Debugf("Config.Write write the file out")
	return common.WriteYaml(c.OutputPath, c.writeYamlMap)
}

// AddSolution adds a problem to the config
func (c *Config) AddSolution(p Problem) error {
	log.Debugf("Config.AddSolution the problem is:\n%+v", p)
	if p.Solution.Type == PasswordSolutionFormType {
		err := fmt.Errorf("Passwords will not be added to the config")
		log.Debug(err)
		return err
	}
	if !p.Resolved {
		err := fmt.Errorf("Unresolved problem. Not going to be added to config")
		log.Warn(err)
		return err
	}
	if p.Solution.Type != MultiSelectSolutionFormType {
		if len(p.Solution.Answer) == 0 {
			return fmt.Errorf("Cannot add the problem\n%v\nto the config because it does not have an answer", p)
		}
		if len(p.Solution.Answer) > 1 {
			return fmt.Errorf("Cannot add the problem\n%v\nto the config because it is not a multi-select problem but it has more than one answer", p)
		}
		set(p.ID, p.Solution.Answer[0], c.yamlMap)
		set(p.ID, p.Solution.Answer[0], c.writeYamlMap)
		err := c.Write()
		if err != nil {
			log.Errorf("Failed to write to the config file. Error: %q", err)
		}
		return err
	}

	// multi-select problem has 2 cases
	key := p.ID
	idx := strings.LastIndex(key, common.Special)
	if idx < 0 {
		// normal case key1 = [val1, val2, val3, ...]
		set(key, p.Solution.Answer, c.yamlMap)
		set(key, p.Solution.Answer, c.writeYamlMap)
		return nil
	}

	// special case
	baseKey, lastKeySegment := key[:idx-len(common.Delim)], key[idx+len(common.Special)+len(common.Delim):]
	if baseKey == "" {
		return fmt.Errorf("Failed to add the problem\n%+v\nto the config. The base key is empty", p)
	}
	for _, option := range p.Solution.Options {
		isOptionSelected := common.IsStringPresent(p.Solution.Answer, option)
		newKey := baseKey + common.Delim + option + common.Delim + lastKeySegment
		set(newKey, isOptionSelected, c.yamlMap)
		set(newKey, isOptionSelected, c.writeYamlMap)
	}
	err := c.Write()
	if err != nil {
		log.Errorf("Failed to write to the config file. Error: %q", err)
	}
	return err
}

// NewConfig creates a new config instance given config strings and paths to config files
func NewConfig(outputPath string, configStrings, configFiles []string) (config *Config) {
	log.Debug("NewConfig create a new config")
	return &Config{
		configFiles:   configFiles,
		configStrings: configStrings,
		OutputPath:    outputPath,
	}
}

func (*nullLogBackend) Log(logging.Level, int, *logging.Record) error {
	return nil
}

func getPrinterAndEvaluator(buffer *bytes.Buffer) (yqlib.Printer, yqlib.StreamEvaluator) {
	logging.SetBackend(new(nullLogBackend))
	indentLevel := 2
	printer := yqlib.NewPrinter(buffer, false, false, false, indentLevel, false)
	evaluator := yqlib.NewStreamEvaluator()
	return printer, evaluator
}

func generateYAMLFromExpression(expr string) (string, error) {
	log.Debugf("generateYAMLFromExpression parsing the string [%s]", expr)
	logging.SetBackend(new(nullLogBackend))
	b := bytes.Buffer{}
	printer, evaluator := getPrinterAndEvaluator(&b)
	if err := evaluator.EvaluateNew(expr, printer); err != nil {
		return "", err
	}
	return string(b.Bytes()), nil
}

func isMap(x reflect.Value) bool {
	return x.IsValid() && !x.IsZero() && (x.Kind() == reflect.Map || (x.Kind() == reflect.Interface && x.Elem().Kind() == reflect.Map))
}

func mergeRecursively(baseV, overrideV reflect.Value) {
	if !isMap(baseV) || !isMap(overrideV) {
		return
	}
	if baseV.Kind() == reflect.Interface {
		baseV = baseV.Elem()
	}
	if overrideV.Kind() == reflect.Interface {
		overrideV = overrideV.Elem()
	}
	for _, k := range overrideV.MapKeys() {
		v := overrideV.MapIndex(k)
		oldv := baseV.MapIndex(k)
		if !isMap(v) || !isMap(oldv) {
			baseV.SetMapIndex(k, v)
			continue
		}
		mergeRecursively(oldv, v)
	}
}

// merge takes 2 mapTs and merges them together recursively.
func merge(baseI, overrideI interface{}) {
	if baseI == nil || overrideI == nil {
		return
	}
	mergeRecursively(reflect.ValueOf(baseI), reflect.ValueOf(overrideI))
}

// mergeYAMLDatasIntoMap merges multiple yamls together into a map.
// Later yamls will override earlier yamls.
func mergeYAMLDatasIntoMap(yamlDatas []string) (mapT, error) {
	if len(yamlDatas) == 0 {
		return mapT{}, nil
	}
	if len(yamlDatas) == 1 {
		var v mapT
		err := yaml.Unmarshal([]byte(yamlDatas[0]), &v)
		return v, err
	}
	vs := make([]interface{}, len(yamlDatas))
	for i, yamlData := range yamlDatas {
		if err := yaml.Unmarshal([]byte(yamlData), &vs[i]); err != nil {
			log.Errorf("Error on unmarshalling the %dth yaml:\n%v\nError: %q", i, yamlData, err)
			return nil, err
		}
	}
	basev := vs[0]
	for _, v := range vs[1:] {
		merge(basev, v)
	}
	return basev.(mapT), nil
}

func get(key string, config mapT) (value interface{}, ok bool) {
	subKeys := getSubKeys(key)
	for _, subKey := range subKeys {
		value, ok = config[subKey]
		if !ok {
			// partial match
			return value, false
		}
		valueMap, ok := value.(mapT)
		if ok {
			config = valueMap
			continue
		}
		// value is an array or a scalar
		return value, true
	}
	// value is a map
	return value, true
}

func set(key string, newValue interface{}, config mapT) {
	subKeys := getSubKeys(key)
	if len(subKeys) == 1 {
		config[key] = newValue
	}
	// at least 2 sub keys. example: move2kube.key1 = val1
	lastIdx := len(subKeys) - 1
	var value interface{}
	var ok bool
	for _, subKey := range subKeys[:lastIdx] {
		value, ok = config[subKey]
		if !ok {
			// sub key doesn't exist
			newMap := mapT{}
			config[subKey] = newMap
			config = newMap
			continue
		}
		valueMap, ok := value.(mapT)
		if ok {
			config = valueMap
			continue
		}
		// sub key exists but corresponding value is not a map
		newMap := mapT{}
		config[subKey] = newMap
		config = newMap
	}
	lastSubKey := subKeys[lastIdx]
	config[lastSubKey] = newValue
}

func getSubKeys(key string) []string {
	unStrippedSubKeys := common.SplitOnDotExpectInsideQuotes(key) // assuming delimiter is dot
	subKeys := []string{}
	for _, unStrippedSubKey := range unStrippedSubKeys {
		subKeys = append(subKeys, common.StripQuotes(unStrippedSubKey))
	}
	return subKeys
}
