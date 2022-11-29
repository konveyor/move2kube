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

package parameterizer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/transformer/kubernetes/k8sschema"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
)

var (
	arrayIndexRegex      = regexp.MustCompile(`^\[(\d+)\]$`)
	complexSubKeyRegex   = regexp.MustCompile(`^\[(\w+:)?(\w+)(=.+)?\]$`)
	stripHelmQuotesRegex = regexp.MustCompile(`'({{.+}})'`)
)

// RT has Key, Value and Matches
type RT struct {
	Key     []string
	Value   interface{}
	Matches map[string]string
}

func isNormal(k string) bool {
	return !strings.Contains(k, "[") || arrayIndexRegex.MatchString(k)
}

// GetAll returns all the keys that matched and all corresponding values
func GetAll(key string, resource interface{}) ([]RT, error) {
	results := []RT{}
	subKeys := GetSubKeys(key)
	currentResult := RT{}
	err := getRecurse(subKeys, 0, resource, currentResult, &results)
	return results, err
}

// getRecurse recurses on the value and finds all matches for the key
func getRecurse(subKeys []string, subKeyIdx int, value interface{}, currentResult RT, results *[]RT) error {
	if subKeyIdx >= len(subKeys) {
		kc := make([]string, len(currentResult.Key))
		copy(kc, currentResult.Key)
		currentResult.Key = kc
		currentResult.Value = value
		*results = append(*results, currentResult)
		return nil
	}
	subKey := subKeys[subKeyIdx]
	if isNormal(subKey) {
		valueMap, ok := value.(map[string]interface{})
		if ok {
			value, ok = valueMap[subKey]
			if ok {
				currentResult.Key = append(currentResult.Key, subKey)
				return getRecurse(subKeys, subKeyIdx+1, value, currentResult, results)
			}
			return fmt.Errorf("failed to find the subkey %s in the map %+v", subKey, valueMap)
		}
		valueArr, ok := value.([]interface{})
		if ok {
			idx, ok := getIndex(subKey)
			if !ok {
				return fmt.Errorf("failed to interpret the subkey %s as an index to the slice %+v", subKey, valueArr)
			}
			if idx >= len(valueArr) {
				return fmt.Errorf("the index %d is out of range for the slice %+v", idx, valueArr)
			}
			value = valueArr[idx]
			currentResult.Key = append(currentResult.Key, subKey)
			return getRecurse(subKeys, subKeyIdx+1, value, currentResult, results)
		}
		return fmt.Errorf("the value is not a map or slice. Actual value %+v is of type %T", value, value)
	}
	// subkey like [containerName:name=nginx]
	if !complexSubKeyRegex.MatchString(subKey) {
		return fmt.Errorf("the subkey %s is invalid", subKey)
	}
	subMatches := complexSubKeyRegex.FindAllStringSubmatch(subKey, -1)
	if len(subMatches) != 1 {
		return fmt.Errorf("expected there to be 1 match. Actual no. of matches %d matches: %+v", len(subMatches), subMatches)
	}
	if len(subMatches[0]) != 4 {
		return fmt.Errorf("expected there to be 4 submatches. Actual no. of submatches %d submatches: %+v", len(subMatches[0]), subMatches[0])
	}
	matchName, matchKey, matchValue := subMatches[0][1], subMatches[0][2], subMatches[0][3]
	if matchName == "" {
		matchName = matchKey
	} else {
		matchName = strings.TrimSuffix(matchName, ":")
	}
	if matchValue != "" {
		matchValue = strings.TrimPrefix(matchValue, "=")
	}
	valueArr, ok := value.([]interface{})
	if !ok {
		return fmt.Errorf("expected a slice of objects. actual value is %+v of type %T", value, value)
	}
	if len(valueArr) == 0 {
		return nil
	}
	for arrIdx, valueMapI := range valueArr {
		valueMap, ok := valueMapI.(map[string]interface{})
		if !ok {
			return fmt.Errorf("expected all the elements of the slice to be object. actual value is %+v of %T", valueMapI, valueMapI)
		}
		actualMatchValueI, ok := valueMap[matchKey]
		if !ok {
			continue
		}
		actualMatchValue, ok := actualMatchValueI.(string)
		if !ok {
			return fmt.Errorf("expected the value to be a string. Actual value is %+v of type %T", actualMatchValueI, actualMatchValueI)
		}
		if matchValue != "" && matchValue != actualMatchValue {
			continue
		}
		if currentResult.Matches == nil {
			currentResult.Matches = map[string]string{}
		}
		orig := currentResult.Matches
		copy := map[string]string{}
		for k, v := range orig {
			copy[k] = v
		}
		copy[matchName] = actualMatchValue
		currentResult.Matches = copy
		origKey := currentResult.Key
		currentResult.Key = append(origKey, "["+cast.ToString(arrIdx)+"]")
		if err := getRecurse(subKeys, subKeyIdx+1, valueArr[arrIdx], currentResult, results); err != nil {
			return err
		}
		currentResult.Matches = orig
		currentResult.Key = origKey
	}
	return nil
}

// get returns the value at the key in the config
/*
func get(key string, config interface{}) (value interface{}, ok bool) {
	subKeys := GetSubKeys(key)
	value = config
	for _, subKey := range subKeys {
		valueMap, ok := value.(map[string]interface{})
		if ok {
			value, ok = valueMap[subKey]
			if ok {
				continue
			}
			return value, false
		}
		valueArr, ok := value.([]interface{})
		if ok {
			idx, ok := getIndex(subKey)
			if ok && idx < len(valueArr) {
				value = valueArr[idx]
				continue
			}
		}
		return value, false
	}
	return value, true
}*/

// set updates the value at the key in the config with the new value
func set(key string, newValue, config interface{}) error {
	if key == "" {
		return fmt.Errorf("the key is an empty string")
	}
	subKeys := GetSubKeys(key)
	if len(subKeys) == 0 {
		return fmt.Errorf("no sub keys found for the key %s", key)
	}
	value := config
	for _, subKey := range subKeys[:len(subKeys)-1] {
		valueMap, ok := value.(map[string]interface{})
		if ok {
			value, ok = valueMap[subKey]
			if ok {
				continue
			}
			return fmt.Errorf("the sub key %s is not present in the map %+v", subKey, valueMap)
		}
		valueArr, ok := value.([]interface{})
		if ok {
			idx, ok := getIndex(subKey)
			if ok && idx < len(valueArr) {
				value = valueArr[idx]
				continue
			}
			return fmt.Errorf("the sub key %s is not a valid index into the array %+v", subKey, valueArr)
		}
		return fmt.Errorf("the sub key %s cannot be matched because we reached a scalar value %+v", subKey, value)
	}
	subKey := subKeys[len(subKeys)-1]
	if valueMap, ok := value.(map[string]interface{}); ok {
		if _, ok := valueMap[subKey]; ok {
			valueMap[subKey] = newValue
			return nil
		}
		return fmt.Errorf("the sub key %s is not present in the map %+v", subKey, valueMap)
	}
	if valueArr, ok := value.([]interface{}); ok {
		idx, ok := getIndex(subKey)
		if ok && idx < len(valueArr) {
			valueArr[idx] = newValue
			return nil
		}
		return fmt.Errorf("the sub key %s is not a valid index into the array %+v", subKey, valueArr)
	}
	return fmt.Errorf("expected a map or array type. Actual value is %+v of type %T", value, value)
}

// setCreatingNew updates the value at the key in the config with the new value
func setCreatingNew(key string, newValue interface{}, config map[string]interface{}) error {
	if key == "" {
		return fmt.Errorf("the key is an empty string")
	}
	subKeys := GetSubKeys(key)
	if len(subKeys) == 0 {
		return fmt.Errorf("no sub keys found for the key %s", key)
	}
	lastIdx := len(subKeys) - 1
	var value interface{}
	var ok bool
	for _, subKey := range subKeys[:lastIdx] {
		value, ok = config[subKey]
		if !ok {
			// sub key doesn't exist
			newMap := map[string]interface{}{}
			config[subKey] = newMap
			config = newMap
			continue
		}
		valueMap, ok := value.(map[string]interface{})
		if ok {
			config = valueMap
			continue
		}
		// sub key exists but corresponding value is not a map
		newMap := map[string]interface{}{}
		config[subKey] = newMap
		config = newMap
	}
	lastSubKey := subKeys[lastIdx]
	config[lastSubKey] = newValue
	return nil
}

// GetSubKeys returns the parts of a key.
// Example aaa.bbb."ccc ddd".eee.fff -> {"aaa", "bbb", "ccc ddd", "eee", "fff"}
func GetSubKeys(key string) []string {
	unStrippedSubKeys := common.SplitOnDotExpectInsideQuotes(key) // assuming delimiter is dot
	subKeys := []string{}
	for _, unStrippedSubKey := range unStrippedSubKeys {
		subKeys = append(subKeys, common.StripQuotes(unStrippedSubKey))
	}
	return subKeys
}

func getIndex(key string) (int, bool) {
	matches := arrayIndexRegex.FindSubmatch([]byte(key))
	if matches == nil {
		return 0, false
	}
	idx, err := cast.ToIntE(string(matches[1]))
	if err != nil || idx < 0 {
		return 0, false
	}
	return idx, true
}

// writeResourceAppendToFile is like WriteResource but appends to the file
func writeResourceAppendToFile(k8sResource k8sschema.K8sResourceT, outputPath string) error {
	logrus.Trace("start WriteResourceAppendToFile")
	defer logrus.Trace("end WriteResourceAppendToFile")
	yamlBytes, err := yaml.Marshal(k8sResource)
	if err != nil {
		logrus.Error("Error while Encoding object")
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), common.DefaultDirectoryPermission); err != nil {
		logrus.Fatalf("Failed to create the output directory at path %s Error: %q", filepath.Dir(outputPath), err)
	}
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, common.DefaultFilePermission)
	if err != nil {
		return fmt.Errorf("failed to open the file at path %s for creating/appending. Error: %q", outputPath, err)
	}
	defer f.Close()
	if _, err := f.Write([]byte("\n---\n" + string(yamlBytes) + "\n...\n")); err != nil {
		return fmt.Errorf("failed to write to the file at path %s . Error: %q", outputPath, err)
	}
	return f.Close()
}

// writeResourceStripQuotesAndAppendToFile is like WriteResource but strips quotes around Helm templates and appends to file
func writeResourceStripQuotesAndAppendToFile(k8sResource k8sschema.K8sResourceT, outputPath string) error {
	logrus.Trace("start WriteResourceStripQuotesAndAppendToFile")
	defer logrus.Trace("end WriteResourceStripQuotesAndAppendToFile")
	yamlBytes, err := yaml.Marshal(k8sResource)
	if err != nil {
		logrus.Error("Error while Encoding object")
		return err
	}
	strippedYamlBytes := stripHelmQuotesRegex.ReplaceAll(yamlBytes, []byte("$1"))
	if err := os.MkdirAll(filepath.Dir(outputPath), common.DefaultDirectoryPermission); err != nil {
		logrus.Fatalf("Failed to create the output directory at path %s Error: %q", filepath.Dir(outputPath), err)
	}
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, common.DefaultFilePermission)
	if err != nil {
		return fmt.Errorf("failed to open the file at path %s for creating/appending. Error: %q", outputPath, err)
	}
	defer f.Close()
	if _, err := f.Write([]byte("\n---\n" + string(strippedYamlBytes) + "\n...\n")); err != nil {
		return fmt.Errorf("failed to write to the file at path %s . Error: %q", outputPath, err)
	}
	return f.Close()
}

// CollectParamsFromPath returns parameterizers found in a directory
func CollectParamsFromPath(parameterizersDir string) (map[string][]ParameterizerT, error) {
	logrus.Trace("CollectParamsFromPath start")
	defer logrus.Trace("CollectParamsFromPath end")
	yamlPaths, err := common.GetFilesByExt(parameterizersDir, []string{".yaml", ".yml"})
	if err != nil {
		return nil, err
	}
	params := map[string][]ParameterizerT{}
	for _, yamlPath := range yamlPaths {
		var paramFile ParameterizerFileT
		if err := common.ReadMove2KubeYamlStrict(yamlPath, &paramFile, ParameterizerKind); err == nil {
			logrus.Debugf("found paramterizer yaml at path %s", yamlPath)
			params[paramFile.ObjectMeta.Name] = paramFile.Spec.Parameterizers
		}
	}
	return params, nil
}
