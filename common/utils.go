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

package common

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/crc64"
	"io"
	"math"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-git/go-git/v5"
	"github.com/konveyor/move2kube/types"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/xrash/smetrics"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//GetFilesByExt returns files by extension
func GetFilesByExt(inputPath string, exts []string) ([]string, error) {
	var files []string
	if info, err := os.Stat(inputPath); os.IsNotExist(err) {
		logrus.Warnf("Error in walking through files due to : %q", err)
		return nil, err
	} else if !info.IsDir() {
		logrus.Warnf("The path %q is not a directory.", inputPath)
	}
	err := filepath.WalkDir(inputPath, func(path string, info os.DirEntry, err error) error {
		if err != nil && path == inputPath { // if walk for root search path return gets error
			// then stop walking and return this error
			return err
		}
		if err != nil {
			logrus.Warnf("Skipping path %q due to error: %q", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			for _, dirRegExp := range DefaultIgnoreDirRegexps {
				if dirRegExp.Match([]byte(filepath.Base(path))) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		fext := filepath.Ext(path)
		for _, ext := range exts {
			if fext == ext {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		logrus.Warnf("Error in walking through files due to : %q", err)
		return files, err
	}
	logrus.Debugf("No of files with %s ext identified : %d", exts, len(files))
	return files, nil
}

// GetFilesByExtInCurrDir returns the files present in current directory which have one of the specified extensions
func GetFilesByExtInCurrDir(inputPath string, exts []string) ([]string, error) {
	var files []string
	if info, err := os.Stat(inputPath); os.IsNotExist(err) {
		logrus.Warnf("Error in walking through files due to : %q", err)
		return nil, err
	} else if !info.IsDir() {
		fext := filepath.Ext(inputPath)
		for _, ext := range exts {
			if fext == ext {
				return []string{inputPath}, nil
			}
		}
		return []string{}, nil
	}
	dirEntries, err := os.ReadDir(inputPath)
	if err != nil {
		logrus.Warnf("Error while trying to read directory : %s", err)
		return []string{}, err
	}
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		fext := filepath.Ext(de.Name())
		for _, ext := range exts {
			if fext == ext {
				files = append(files, filepath.Join(inputPath, de.Name()))
			}
		}
	}
	return files, nil
}

//GetFilesByName returns files by name
func GetFilesByName(inputPath string, names []string, nameRegexes []string) ([]string, error) {
	var files []string
	if info, err := os.Stat(inputPath); os.IsNotExist(err) {
		logrus.Warnf("Error in walking through files due to : %q", err)
		return files, err
	} else if !info.IsDir() {
		logrus.Warnf("The path %q is not a directory.", inputPath)
	}
	compiledNameRegexes := []*regexp.Regexp{}
	for _, nameRegex := range nameRegexes {
		compiledNameRegex, err := regexp.Compile(nameRegex)
		if err != nil {
			logrus.Errorf("Could not compile regular expression (%s): %s. Ignoring regex during search", err, nameRegex)
			continue
		}
		compiledNameRegexes = append(compiledNameRegexes, compiledNameRegex)
	}
	err := filepath.WalkDir(inputPath, func(path string, info os.DirEntry, err error) error {
		if err != nil && path == inputPath { // if walk for root search path return gets error
			// then stop walking and return this error
			return err
		}
		if err != nil {
			logrus.Warnf("Skipping path %q due to error: %q", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			for _, dirRegExp := range DefaultIgnoreDirRegexps {
				if dirRegExp.Match([]byte(filepath.Base(path))) {
					return filepath.SkipDir
				}
			}
			return nil
		}
		fname := filepath.Base(path)
		for _, name := range names {
			if name == fname {
				files = append(files, path)
				return nil
			}
		}
		for _, compiledNameRegex := range compiledNameRegexes {
			if compiledNameRegex.MatchString(fname) {
				files = append(files, path)
				return nil
			}
		}
		return nil
	})
	if err != nil {
		logrus.Warnf("Error in walking through files due to : %s", err)
		return files, err
	}
	logrus.Debugf("No of files with %s names identified : %d", names, len(files))
	return files, nil
}

// GetFilesInCurrentDirectory returns the name of the file present in the current directory which matches the pattern
func GetFilesInCurrentDirectory(path string, fileNames, fileNameRegexes []string) (files []string, err error) {
	files = []string{}
	currFileNames := []string{}
	if info, err := os.Stat(path); os.IsNotExist(err) {
		logrus.Warnf("Error in walking through files due to : %q", err)
		return nil, err
	} else if !info.IsDir() {
		currFileNames = append(currFileNames, path)
	} else {
		dirhandle, err := os.Open(path)
		if err != nil {
			logrus.Errorf("Unable to open directory: %s", err)
			return nil, err
		}
		defer dirhandle.Close()
		currFileNames, err = dirhandle.Readdirnames(0) // 0 to read all files and folders
		if err != nil {
			logrus.Errorf("Unable to read dir names in %s : %s", path, err)
			return nil, err
		}
	}
	compiledNameRegexes := []*regexp.Regexp{}
	for _, nameRegex := range fileNameRegexes {
		compiledNameRegex, err := regexp.Compile(nameRegex)
		if err != nil {
			logrus.Errorf("Could not compile regular expression (%s): %s. Ignoring regex during search", err, nameRegex)
			continue
		}
		compiledNameRegexes = append(compiledNameRegexes, compiledNameRegex)
	}
	for _, fName := range currFileNames {
		for _, name := range fileNames {
			if name == fName {
				files = append(files, filepath.Join(path, fName))
			}
		}
		for _, compiledNameRegex := range compiledNameRegexes {
			if compiledNameRegex.MatchString(fName) {
				files = append(files, filepath.Join(path, fName))
			}
		}
	}
	return files, nil
}

//YamlAttrPresent returns YAML attributes
func YamlAttrPresent(path string, attr string) (bool, interface{}) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		logrus.Warnf("Error in reading yaml file %s: %s. Skipping", path, err)
		return false, nil
	}
	var fileContents map[string]interface{}
	err = yaml.Unmarshal(yamlFile, &fileContents)
	if err != nil {
		logrus.Warnf("Error in unmarshalling yaml file %s: %s. Skipping", path, err)
		return false, nil
	}
	if value, ok := fileContents[attr]; ok {
		logrus.Debugf("%s file has %s attribute", path, attr)
		return true, value
	}
	return false, nil
}

// GetImageNameAndTag splits an image full name and returns the image name and tag
func GetImageNameAndTag(image string) (string, string) {
	parts := strings.Split(image, "/")
	imageAndTag := strings.Split(parts[len(parts)-1], ":")
	imageName := imageAndTag[0]
	var tag string
	if len(imageAndTag) == 1 {
		// no tag, assume latest
		tag = "latest"
	} else {
		tag = imageAndTag[1]
	}

	return imageName, tag
}

// ObjectToYamlBytes encodes an object to yaml
func ObjectToYamlBytes(data interface{}) ([]byte, error) {
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		logrus.Errorf("Failed to encode the object to yaml. Error: %q", err)
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		logrus.Errorf("Failed to close the yaml encoder. Error: %q", err)
		return nil, err
	}
	return b.Bytes(), nil
}

// WriteYaml writes encodes object as yaml and writes it to a file
func WriteYaml(outputPath string, data interface{}) error {
	yamlBytes, err := ObjectToYamlBytes(data)
	if err != nil {
		logrus.Errorf("Failed to encode the object as a yaml string. Error: %q", err)
		return err
	}
	return os.WriteFile(outputPath, yamlBytes, DefaultFilePermission)
}

// ReadYaml reads an yaml into an object
func ReadYaml(file string, data interface{}) error {
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		logrus.Debugf("Error in reading yaml file %s: %s.", file, err)
		return err
	}
	err = yaml.Unmarshal(yamlFile, data)
	if err != nil {
		logrus.Debugf("Error in unmarshalling yaml file %s: %s.", file, err)
		return err
	}
	rv := reflect.ValueOf(data)
	if rv.Kind() == reflect.Ptr && rv.Elem().Kind() == reflect.Struct {
		rv = rv.Elem()
		fv := rv.FieldByName("APIVersion")
		if fv.IsValid() {
			if fv.Kind() == reflect.String {
				val := strings.TrimSpace(fv.String())
				if strings.HasPrefix(val, types.SchemeGroupVersion.Group) && !strings.HasSuffix(val, types.SchemeGroupVersion.Version) {
					logrus.Warnf("The application file (%s) was generated using a different version than (%s)", val, types.SchemeGroupVersion.String())
				}
			}
		}
	}
	return nil
}

// ReadMove2KubeYaml reads move2kube specific yaml files (like m2k.plan) into an struct.
// It checks if apiVersion to see if the group is move2kube and also reports if the
// version is different from the expected version.
func ReadMove2KubeYaml(path string, out interface{}) error {
	yamlData, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("Failed to read the yaml file at path %s Error: %q", path, err)
		return err
	}
	yamlMap := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(yamlData), yamlMap); err != nil {
		logrus.Debugf("Error occurred while unmarshalling yaml file at path %s Error: %q", path, err)
		return err
	}
	groupVersionI, ok := yamlMap["apiVersion"]
	if !ok {
		err := fmt.Errorf("did not find apiVersion in the yaml file at path %s", path)
		logrus.Debug(err)
		return err
	}
	groupVersionStr, ok := groupVersionI.(string)
	if !ok {
		err := fmt.Errorf("the apiVersion is not a string in the yaml file at path %s", path)
		logrus.Debug(err)
		return err
	}
	groupVersion, err := schema.ParseGroupVersion(groupVersionStr)
	if err != nil {
		logrus.Debugf("Failed to parse the apiVersion %s Error: %q", groupVersionStr, err)
		return err
	}
	if groupVersion.Group != types.SchemeGroupVersion.Group {
		err := fmt.Errorf("the file at path %s doesn't have the correct group. Expected group %s Actual group %s", path, types.SchemeGroupVersion.Group, groupVersion.Group)
		logrus.Debug(err)
		return err
	}
	if groupVersion.Version != types.SchemeGroupVersion.Version {
		logrus.Warnf("The file at path %s was generated using a different version. File version is %s and move2kube version is %s", path, groupVersion.Version, types.SchemeGroupVersion.Version)
	}
	if err := yaml.Unmarshal(yamlData, out); err != nil {
		logrus.Debugf("Error occurred while unmarshalling yaml file at path %s Error: %q", path, err)
		return err
	}
	return nil
}

// ReadMove2KubeYamlStrict is like ReadMove2KubeYaml but returns an error
// when it finds unknown fields in the yaml
func ReadMove2KubeYamlStrict(path string, out interface{}, kind string) error {
	yamlData, err := os.ReadFile(path)
	if err != nil {
		logrus.Debugf("Failed to read the yaml file at path %s Error: %q", path, err)
		return err
	}
	yamlMap := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(yamlData), yamlMap); err != nil {
		logrus.Debugf("Error occurred while unmarshalling yaml file at path %s Error: %q", path, err)
		return err
	}
	groupVersionI, ok := yamlMap["apiVersion"]
	if !ok {
		err := fmt.Errorf("did not find apiVersion in the yaml file at path %s", path)
		logrus.Debug(err)
		return err
	}
	groupVersionStr, ok := groupVersionI.(string)
	if !ok {
		err := fmt.Errorf("the apiVersion is not a string in the yaml file at path %s", path)
		logrus.Debug(err)
		return err
	}
	groupVersion, err := schema.ParseGroupVersion(groupVersionStr)
	if err != nil {
		logrus.Debugf("Failed to parse the apiVersion %s Error: %q", groupVersionStr, err)
		return err
	}
	if groupVersion.Group != types.SchemeGroupVersion.Group {
		err := fmt.Errorf("the file at path %s doesn't have the correct group. Expected group %s Actual group %s", path, types.SchemeGroupVersion.Group, groupVersion.Group)
		logrus.Debug(err)
		return err
	}
	if groupVersion.Version != types.SchemeGroupVersion.Version {
		logrus.Warnf("The file at path %s was generated using a different version. File version is %s and move2kube version is %s", path, groupVersion.Version, types.SchemeGroupVersion.Version)
	}
	actualKindI, ok := yamlMap["kind"]
	if !ok {
		err := fmt.Errorf("the file at path %s does not have a kind specified", path)
		logrus.Debug(err)
		return err
	}
	actualKind, ok := actualKindI.(string)
	if !ok {
		err := fmt.Errorf("the kind is not a string in the yaml file at path %s", path)
		logrus.Debug(err)
		return err
	}
	if kind != "" && actualKind != kind {
		err := fmt.Errorf("the file at path %s does not have the expected kind. Expected: %s Actual: %s", path, kind, actualKind)
		logrus.Debug(err)
		return err
	}
	jsonBytes, err := json.Marshal(yamlMap)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(jsonBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		logrus.Debugf("Error occurred while unmarshalling yaml file at path %s Error: %q", path, err)
		return err
	}
	return nil
}

// WriteJSON writes an json to disk
func WriteJSON(outputPath string, data interface{}) error {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	if err := encoder.Encode(data); err != nil {
		logrus.Error("Error while Encoding object")
		return err
	}
	err := os.WriteFile(outputPath, b.Bytes(), DefaultFilePermission)
	if err != nil {
		logrus.Errorf("Error writing json to file: %s", err)
		return err
	}
	return nil
}

// ReadJSON reads an json into an object
func ReadJSON(file string, data interface{}) error {
	jsonFile, err := os.ReadFile(file)
	if err != nil {
		logrus.Debugf("Error in reading json file %s: %s.", file, err)
		return err
	}
	jsonFile = bytes.TrimPrefix(jsonFile, []byte("\xef\xbb\xbf"))
	err = json.Unmarshal(jsonFile, &data)
	if err != nil {
		logrus.Debugf("Error in unmarshalling json file %s: %s.", file, err)
		return err
	}
	return nil
}

// ReadXML reads an json into an object
func ReadXML(file string, data interface{}) error {
	xmlFile, err := os.ReadFile(file)
	if err != nil {
		logrus.Debugf("Error in reading xml file %s: %s.", file, err)
		return err
	}
	err = xml.Unmarshal(xmlFile, &data)
	if err != nil {
		logrus.Debugf("Error in unmarshalling xml file %s: %s.", file, err)
		return err
	}
	return nil
}

// NormalizeForFilename normalizes a string to only filename valid characters
func NormalizeForFilename(name string) string {
	processedString := MakeFileNameCompliant(name)
	//TODO: Make it more robust by taking some characters from start and also from end
	const maxPrefixLength = 15
	if len(processedString) > maxPrefixLength {
		processedString = processedString[0:maxPrefixLength]
	}
	crc64Table := crc64.MakeTable(0xC96C5795D7870F42)
	crc64Int := crc64.Checksum([]byte(name), crc64Table)
	return processedString + "-" + strconv.FormatUint(crc64Int, 16)
}

// NormalizeForServiceName converts the string to be compatible for service name
func NormalizeForServiceName(svcName string) string {
	newName := disallowedDNSCharactersRegex.ReplaceAllLiteralString(strings.ToLower(svcName), "-")
	if newName != svcName {
		logrus.Infof("Changing service name to %s from %s", svcName, newName)
	}
	return newName
}

// IsStringPresent checks if a value is present in a slice
func IsStringPresent(list []string, value string) bool {
	for _, val := range list {
		if strings.EqualFold(val, value) {
			return true
		}
	}
	return false
}

// IsIntPresent checks if a value is present in a slice
func IsIntPresent(list []int, value int) bool {
	for _, val := range list {
		if val == value {
			return true
		}
	}
	return false
}

// MergeStringSlices merges two string slices
func MergeStringSlices(slice1 []string, slice2 ...string) []string {
	for _, item := range slice2 {
		if !IsStringPresent(slice1, item) {
			slice1 = append(slice1, item)
		}
	}
	return slice1
}

// MergeIntSlices merges two int slices
func MergeIntSlices(slice1 []int, slice2 []int) []int {
	for _, item := range slice2 {
		if !IsIntPresent(slice1, item) {
			slice1 = append(slice1, item)
		}
	}
	return slice1
}

// MergeInt32Slices merges two int slices
func MergeInt32Slices(slice1 []int32, slice2 []int32) []int32 {
	for _, item := range slice2 {
		if !IsInt32Present(slice1, item) {
			slice1 = append(slice1, item)
		}
	}
	return slice1
}

// IsInt32Present checks if a value is present in a slice
func IsInt32Present(list []int32, value int32) bool {
	for _, val := range list {
		if val == value {
			return true
		}
	}
	return false
}

// GetStringFromTemplate returns string for a template
func GetStringFromTemplate(tpl string, config interface{}) (string, error) {
	var tplbuffer bytes.Buffer
	var packageTemplate = template.Must(template.New("").Parse(tpl))
	err := packageTemplate.Execute(&tplbuffer, config)
	if err != nil {
		logrus.Warnf("Unable to transform template %q to string using the data %v", tpl, config)
		return "", err
	}
	return tplbuffer.String(), nil
}

// GetClosestMatchingString returns the closest matching string for a given search string
func GetClosestMatchingString(options []string, searchstring string) string {
	// tokenize all strings
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	searchstring = reg.ReplaceAllLiteralString(searchstring, "")
	searchstring = strings.ToLower(searchstring)

	leastDistance := math.MaxInt32
	matchString := ""

	// Simply find the option with least distance
	for _, option := range options {
		// do tokensize the search space string too
		tokenizedOption := reg.ReplaceAllLiteralString(option, "")
		tokenizedOption = strings.ToLower(tokenizedOption)

		currDistance := smetrics.WagnerFischer(tokenizedOption, searchstring, 1, 1, 2)

		if currDistance < leastDistance {
			matchString = option
			leastDistance = currDistance
		}
	}

	return matchString
}

// MergeStringMaps merges two string maps
func MergeStringMaps(map1 map[string]string, map2 map[string]string) map[string]string {
	if map1 == nil {
		return map2
	}
	if map2 == nil {
		return map1
	}
	for k, v := range map2 {
		map1[k] = v
	}
	return map1
}

// MergeStringSliceMaps merges two string slice maps
func MergeStringSliceMaps(map1 map[string][]string, map2 map[string][]string) map[string][]string {
	if map1 == nil {
		return map2
	}
	if map2 == nil {
		return map1
	}
	for k, v := range map2 {
		map1[k] = MergeStringSlices(map1[k], v...)
	}
	return map1
}

// MakeFileNameCompliant returns a DNS-1123 standard string
// Motivated by https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
// Also see page 1 "ASSUMPTIONS" heading of https://tools.ietf.org/html/rfc952
// Also see page 13 of https://tools.ietf.org/html/rfc1123#page-13
func MakeFileNameCompliant(name string) string {
	if len(name) == 0 {
		logrus.Error("The input name is empty.")
		return ""
	}
	baseName := filepath.Base(name)
	invalidChars := regexp.MustCompile("[^a-zA-Z0-9-.]+")
	processedName := invalidChars.ReplaceAllLiteralString(baseName, "-")
	if len(processedName) > 63 {
		logrus.Debugf("Warning: The processed name %q is longer than 63 characters long.", processedName)
	}
	first := processedName[0]
	last := processedName[len(processedName)-1]
	if first == '-' || first == '.' || last == '-' || last == '.' {
		logrus.Debugf("Warning: The first and/or last characters of the name %q are not alphanumeric.", processedName)
	}
	return processedName
}

// GetSHA256Hash returns the SHA256 hash of the string.
// The hash is 256 bits/32 bytes and encoded as a 64 char hexadecimal string.
func GetSHA256Hash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

// MakeStringDNSNameCompliant makes the string into a valid DNS name.
func MakeStringDNSNameCompliant(s string) string {
	name := strings.ToLower(s)
	name = regexp.MustCompile(`[^a-z0-9-.]`).ReplaceAllLiteralString(name, "-")
	start, end := name[0], name[len(name)-1]
	if start == '-' || start == '.' || end == '-' || end == '.' {
		logrus.Debugf("The first and/or last characters of the string %q are not alphanumeric.", s)
	}
	return name
}

// MakeStringContainerImageNameCompliant makes the string into a valid image name.
func MakeStringContainerImageNameCompliant(s string) string {
	name := strings.ToLower(s)
	name = regexp.MustCompile(`[^a-z0-9-.:]`).ReplaceAllLiteralString(name, "-")
	start, end := name[0], name[len(name)-1]
	if start == '-' || start == '.' || end == '-' || end == '.' {
		logrus.Debugf("The first and/or last characters of the string %q are not alphanumeric.", s)
	}
	return name
}

// MakeStringDNSSubdomainNameCompliant makes the string a valid DNS subdomain name.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-subdomain-names
// 1. contain no more than 253 characters
// 2. contain only lowercase alphanumeric characters, '-' or '.'
// 3. start with an alphanumeric character
// 4. end with an alphanumeric character
func MakeStringDNSSubdomainNameCompliant(s string) string {
	name := s
	if len(name) > 253 {
		hash := GetSHA256Hash(name)
		name = name[:253-65] // leave room for the hash (64 chars) plus hyphen (1 char).
		name = name + "-" + hash
	}
	return MakeStringDNSNameCompliant(name)
}

// MakeStringDNSLabelNameCompliant makes the string a valid DNS label name.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names
// 1. contain at most 63 characters
// 2. contain only lowercase alphanumeric characters or '-'
// 3. start with an alphanumeric character
// 4. end with an alphanumeric character
func MakeStringDNSLabelNameCompliant(s string) string {
	name := s
	if len(name) > 63 {
		hash := GetSHA256Hash(name)
		hash = hash[:32]
		name = name[:63-33] // leave room for the hash (32 chars) plus hyphen (1 char).
		name = name + "-" + hash
	}
	return MakeStringDNSNameCompliant(name)
}

// MakeStringEnvNameCompliant makes the string into a valid Environment variable name.
func MakeStringEnvNameCompliant(s string) string {
	name := strings.ToUpper(s)
	name = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllLiteralString(name, "_")
	if regexp.MustCompile(`^[0-9]`).Match([]byte(name)) {
		logrus.Debugf("The first characters of the string %q must not be a digit.", s)
	}
	return name
}

// MakeStringPathSegmentNameCompliant makes the string a valid path segment name.
// See https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#path-segment-names
// The name cannot be "." or ".." and the name should not contain "/" or "%".
// See https://tools.ietf.org/html/rfc3986#section-3.3
// segment       = *pchar
// pchar         = unreserved / pct-encoded / sub-delims / ":" / "@"
// unreserved    = ALPHA / DIGIT / "-" / "." / "_" / "~"
// pct-encoded   = "%" HEXDIG HEXDIG
// sub-delims    = "!" / "$" / "&" / "'" / "(" / ")" / "*" / "+" / "," / ";" / "="
// 2.3.  Unreserved Characters
//    Characters that are allowed in a URI but do not have a reserved
//    purpose are called unreserved.  These include uppercase and lowercase
//    letters, decimal digits, hyphen, period, underscore, and tilde.
//       unreserved  = ALPHA / DIGIT / "-" / "." / "_" / "~"
// 1.3.  Syntax Notation
//    This specification uses the Augmented Backus-Naur Form (ABNF)
//    notation of [RFC2234], including the following core ABNF syntax rules
//    defined by that specification: ALPHA (letters), CR (carriage return),
//    DIGIT (decimal digits), DQUOTE (double quote), HEXDIG (hexadecimal
//    digits), LF (line feed), and SP (space).  The complete URI syntax is
//    collected in Appendix A.
// func MakeStringPathSegmentNameCompliant(s string) string {
// 	return s
// }

// CleanAndFindCommonDirectory finds the common ancestor directory among a list of absolute paths.
// Cleans the paths you give it before finding the directory.
// Also see FindCommonDirectory
func CleanAndFindCommonDirectory(paths []string) string {
	cleanedpaths := make([]string, len(paths))
	for i, path := range paths {
		cleanedpaths[i] = filepath.Clean(path)
	}
	return FindCommonDirectory(cleanedpaths)
}

// FindCommonDirectory finds the common ancestor directory among a list of cleaned absolute paths.
// Will not clean the paths you give it before trying to find the directory.
// Also see CleanAndFindCommonDirectory
func FindCommonDirectory(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	slash := string(filepath.Separator)
	commonDir := paths[0]
	for commonDir != slash {
		found := true
		for _, path := range paths {
			if !strings.HasPrefix(path+slash, commonDir+slash) {
				found = false
				break
			}
		}
		if found {
			break
		}
		commonDir = filepath.Dir(commonDir)
	}
	return commonDir
}

// CreateAssetsData creates an assets directory and dumps the assets data into it
func CreateAssetsData(assetsFS embed.FS, permissions map[string]int) (assetsPath string, tempPath string, err error) {
	// Return the absolute version of existing asset paths.
	tempPath, err = filepath.Abs(TempPath)
	if err != nil {
		logrus.Errorf("Unable to make the temporary directory path %q absolute. Error: %q", tempPath, err)
		return "", "", err
	}
	assetsPath, err = filepath.Abs(AssetsPath)
	if err != nil {
		logrus.Errorf("Unable to make the assets path %q absolute. Error: %q", assetsPath, err)
		return "", "", err
	}

	// Try to create a new temporary directory for the assets.
	if newTempPath, err := os.MkdirTemp("", types.AppName+"*"); err != nil {
		logrus.Errorf("Unable to create temp dir. Defaulting to local path.")
	} else {
		tempPath = newTempPath
		assetsPath = filepath.Join(newTempPath, AssetsDir)
	}

	// Either way create the subdirectory and untar the assets into it.
	if err := os.MkdirAll(assetsPath, DefaultDirectoryPermission); err != nil {
		logrus.Errorf("Unable to create the assets directory at path %q Error: %q", assetsPath, err)
		return "", "", err
	}
	if err := CopyEmbedFSToDir(assetsFS, ".", assetsPath, permissions); err != nil {
		logrus.Errorf("Unable to untar the assets into the assets directory at path %q Error: %q", assetsPath, err)
		return "", "", err
	}
	return assetsPath, tempPath, nil
}

// CopyEmbedFSToDir converts a string into a directory
func CopyEmbedFSToDir(embedFS embed.FS, source, dest string, permissions map[string]int) (err error) {
	f, err := embedFS.Open(GetUnixPath(source))
	if err != nil {
		logrus.Errorf("Error while reading embedded file : %s", err)
		return err
	}
	finfo, err := f.Stat()
	if err != nil {
		logrus.Errorf("Error while reading stat of embedded file : %s", err)
		return err
	}
	if finfo != nil && !finfo.Mode().IsDir() {
		permission, ok := permissions[GetUnixPath(source)]
		if !ok {
			logrus.Errorf("Permission missing for file %s. Do `make generate` to update permissions file.", dest)
		}
		df, err := os.OpenFile(dest, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(permission))
		if err != nil {
			logrus.Errorf("Error while opening temp dest assets file : %s", err)
			return err
		}
		defer df.Close()
		size, err := io.Copy(df, f)
		if err != nil {
			logrus.Errorf("Error while copying embedded file : %s", err)
			return err
		}
		if size != finfo.Size() {
			return fmt.Errorf("size mismatch: Wrote %d, Expected %d", size, finfo.Size())
		}
		return nil
	}
	if err := os.MkdirAll(dest, DefaultDirectoryPermission); err != nil {
		return err
	}
	dirEntries, err := embedFS.ReadDir(GetUnixPath(source))
	if err != nil {
		logrus.Errorf("Error while trying to read directory : %s", err)
		return err
	}
	for _, de := range dirEntries {
		CopyEmbedFSToDir(embedFS, filepath.Join(source, de.Name()), filepath.Join(dest, removeDollarPrefixFromHiddenDir(de.Name())), permissions)
	}
	return nil
}

// GetUnixPath return Unix Path for any path
func GetUnixPath(path string) string {
	return strings.ReplaceAll(path, `\`, `/`)
}

// GetWindowsPath return Windows Path for any path
func GetWindowsPath(path string) string {
	return strings.ReplaceAll(path, `/`, `\`)
}

func removeDollarPrefixFromHiddenDir(name string) string {
	if strings.HasPrefix(name, "$.") || strings.HasPrefix(name, "$_") {
		name = name[1:]
	}
	return name
}

// CopyFile copies a file from src to dst.
// The dst file will be truncated if it exists.
// Returns an error if it failed to copy all the bytes.
func CopyFile(dst, src string) error {
	srcfile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open the source file at path %q Error: %q", src, err)
	}
	defer srcfile.Close()

	srcfileinfo, err := srcfile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get size of the source file at path %q Error: %q", src, err)
	}
	srcfilesize := srcfileinfo.Size()

	dstfile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultFilePermission)
	if err != nil {
		return fmt.Errorf("failed to create the destination file at path %q Error: %q", dst, err)
	}
	defer dstfile.Close()

	written, err := io.Copy(dstfile, srcfile)
	if written != srcfilesize {
		return fmt.Errorf("failed to copy all the bytes from source %q to destination %q. %d out of %d bytes written. Error: %v", src, dst, written, srcfilesize, err)
	}
	if err != nil {
		return fmt.Errorf("failed to copy from source %q to destination %q. Error: %q", src, dst, err)
	}

	return dstfile.Close()
}

// UniqueStrings returns a new slice with only the unique strings from the input slice.
func UniqueStrings(xs []string) []string {
	exists := map[string]int{}
	nextIdx := 0
	for _, x := range xs {
		if _, ok := exists[x]; ok {
			continue
		}
		exists[x] = nextIdx
		nextIdx++
	}
	unique := make([]string, len(exists))
	for x, idx := range exists {
		unique[idx] = x
	}
	return unique
}

// SplitYAML splits a file into multiple YAML documents.
func SplitYAML(rawYAML []byte) ([][]byte, error) {
	dec := yaml.NewDecoder(bytes.NewReader(rawYAML))
	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}

// ReverseInPlace reverses a slice of strings in place.
func ReverseInPlace(xs []string) {
	for i := 0; i < len(xs)/2; i++ {
		j := len(xs) - i - 1
		xs[i], xs[j] = xs[j], xs[i]
	}
}

// IsParent can be used to check if a path is one of the parent directories of another path.
// Also returns true if the paths are the same.
func IsParent(child, parent string) bool {
	var err error
	child, err = filepath.Abs(child)
	if err != nil {
		logrus.Fatalf("Failed to make the path %s absolute. Error: %s", child, err)
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		logrus.Fatalf("Failed to make the path %s absolute. Error: %s", parent, err)
	}
	if parent == "/" {
		return true
	}
	childParts := strings.Split(child, string(os.PathSeparator))
	parentParts := strings.Split(parent, string(os.PathSeparator))
	if len(parentParts) > len(childParts) {
		return false
	}
	for i, parentPart := range parentParts {
		if childParts[i] != parentPart {
			return false
		}
	}
	return true
}

// GetRandomString generates a random string
func GetRandomString() string {
	return fmt.Sprintf("%d", rand.Intn(10000000))
}

// SplitOnDotExpectInsideQuotes splits a string on dot.
// Stuff inside double or single quotes will not be split.
func SplitOnDotExpectInsideQuotes(s string) []string {
	return regexp.MustCompile(`[^."']+|"[^"]*"|'[^']*'`).FindAllString(s, -1)
}

// StripQuotes strips a single layer of double or single quotes from the left and right ends
// Example: "github.com" -> github.com
// Example: 'github.com' -> github.com
// Example: "'github.com'" -> 'github.com'
func StripQuotes(s string) string {
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return strings.TrimSuffix(strings.TrimPrefix(s, `"`), `"`)
	}
	if strings.HasPrefix(s, `'`) && strings.HasSuffix(s, `'`) {
		return strings.TrimSuffix(strings.TrimPrefix(s, `'`), `'`)
	}
	return s
}

// GetRuntimeObjectMetadata returns the metadata field from a k8s object.
func GetRuntimeObjectMetadata(obj runtime.Object) metav1.ObjectMeta {
	k8sObjValue := reflect.ValueOf(obj).Elem()
	return k8sObjValue.FieldByName("ObjectMeta").Interface().(metav1.ObjectMeta)
}

// IsSameRuntimeObject returns true if the 2 k8s resources are same.
// 2 resources are the same if they have the same group, version, kind, namespace and name.
// Also prints an error if the 2 objects have the same kind, namespace and name but different group versions.
func IsSameRuntimeObject(obj1, obj2 runtime.Object) bool {
	meta1 := GetRuntimeObjectMetadata(obj1)
	meta2 := GetRuntimeObjectMetadata(obj2)
	if meta1.GetName() != meta2.GetName() || meta1.GetNamespace() != meta2.GetNamespace() {
		return false
	}
	gvk1 := obj1.GetObjectKind().GroupVersionKind()
	gvk2 := obj2.GetObjectKind().GroupVersionKind()
	if gvk1 != gvk2 {
		if gvk1.Kind == gvk2.Kind {
			logrus.Errorf("The 2 objects have the same kind, namespace and name but different group versions. Object1:\n%+v\nObject2:\n%+v", obj1, obj2)
		}
		return false
	}
	return true
}

// MarshalObjToYaml marshals an object to yaml
func MarshalObjToYaml(obj runtime.Object) ([]byte, error) {
	objJSONBytes, err := json.Marshal(obj)
	if err != nil {
		logrus.Errorf("Error while marshalling object %+v to json. Error: %q", obj, err)
		return nil, err
	}
	var jsonObj interface{}
	if err := yaml.Unmarshal(objJSONBytes, &jsonObj); err != nil {
		logrus.Errorf("Unable to unmarshal the json as yaml:\n%s\nError: %q", objJSONBytes, err)
		return nil, err
	}
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(jsonObj); err != nil {
		logrus.Errorf("Error while encoding the json object:\n%s\nError: %q", jsonObj, err)
		return nil, err
	}
	return b.Bytes(), nil
}

// ConvertInterfaceToSliceOfStrings converts an interface{} to a []string type.
// It can handle []interface{} as long as all the values are strings.
func ConvertInterfaceToSliceOfStrings(xI interface{}) ([]string, error) {
	if x, ok := xI.([]string); ok {
		return x, nil
	}
	vIs, ok := xI.([]interface{})
	if !ok {
		return nil, fmt.Errorf("failed to convert to []string. Actual value is %+v of type %T", xI, xI)
	}
	vs := []string{}
	for _, vI := range vIs {
		v, ok := vI.(string)
		if !ok {
			return vs, fmt.Errorf("some of the values are not strings. Actual value is %+v of type %T", xI, xI)
		}
		vs = append(vs, v)
	}
	return vs, nil
}

// GatherGitInfo tries to find the git repo for the path if one exists.
func GatherGitInfo(path string) (repoName, repoDir, repoHostName, repoURL, repoBranch string, err error) {
	if finfo, err := os.Stat(path); err != nil {
		logrus.Errorf("Failed to stat the path %q Error %q", path, err)
		return "", "", "", "", "", err
	} else if !finfo.IsDir() {
		pathDir := filepath.Dir(path)
		logrus.Debugf("The path %q is not a directory. Using %q instead.", path, pathDir)
		path = pathDir
	}
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		logrus.Debugf("Unable to open the path %q as a git repo. Error: %q", path, err)
		return "", "", "", "", "", err
	}
	remotes, err := repo.Remotes()
	if err != nil || len(remotes) == 0 {
		logrus.Debugf("No remotes found at path %q Error: %q", path, err)
	}
	var preferredRemote *git.Remote
	if preferredRemote = getGitRemoteByName(remotes, "upstream"); preferredRemote == nil {
		if preferredRemote = getGitRemoteByName(remotes, "origin"); preferredRemote == nil {
			preferredRemote = remotes[0]
		}
	}
	if workTree, err := repo.Worktree(); err == nil {
		repoDir = workTree.Filesystem.Root()
	} else {
		logrus.Debugf("Unable to get the repo directory. Error: %q", err)
	}
	if ref, err := repo.Head(); err == nil {
		repoBranch = filepath.Base(string(ref.Name()))
	} else {
		logrus.Debugf("Unable to get the current branch. Error: %q", err)
	}
	if len(preferredRemote.Config().URLs) == 0 {
		err = fmt.Errorf("unable to get origins")
		logrus.Debugf("%s", err)
	}
	u := preferredRemote.Config().URLs[0]
	if strings.HasPrefix(u, "git") {
		parts := strings.Split(u, ":")
		if len(parts) == 2 {
			u = parts[1]
		}
	}
	giturl, err := url.Parse(u)
	if err != nil {
		logrus.Debugf("Unable to get origin remote host : %s", err)
	}
	repoName = filepath.Base(giturl.Path)
	repoName = strings.TrimSuffix(repoName, filepath.Ext(repoName))
	err = nil
	return
}

func getGitRemoteByName(remotes []*git.Remote, remoteName string) *git.Remote {
	for _, r := range remotes {
		if r.Config().Name == remoteName {
			return r
		}
	}
	return nil
}

// GetObjFromInterface loads from map[string]interface{} to struct
func GetObjFromInterface(obj interface{}, loadinto interface{}) error {
	decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   &loadinto,
		TagName:  "yaml",
	})
	if err := decoder.Decode(obj); err != nil {
		logrus.Errorf("Unable to load obj %+v into %T : %s", obj, loadinto, err)
		return err
	}
	logrus.Debugf("Object Loaded is %+v", loadinto)
	return nil
}

// GetMapInterfaceFromObj converts a struct to map[string]interface{} using yaml marshaller
func GetMapInterfaceFromObj(obj interface{}) (mapobj interface{}, err error) {
	objbytes, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}
	mapobj = map[string]interface{}{}
	err = yaml.Unmarshal(objbytes, &mapobj)
	if err != nil {
		return nil, err
	}
	return mapobj, nil
}

// Interrupt creates SIGINT signal
func Interrupt() error {
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		logrus.Fatal(err)
		return err
	}
	if err := p.Signal(os.Interrupt); err != nil {
		logrus.Fatal(err)
		return err
	}
	return nil
}

// GetTypesMap returns a type registry for the types in the array
func GetTypesMap(typeInstances interface{}) (typesMap map[string]reflect.Type) {
	typesMap = map[string]reflect.Type{}
	types := reflect.ValueOf(typeInstances)
	for i := 0; i < types.Len(); i++ {
		t := reflect.TypeOf(types.Index(i).Interface()).Elem()
		tn := t.Name()
		if ot, ok := typesMap[tn]; ok {
			logrus.Errorf("Two transformer classes have the same name %s : %T, %T; Ignoring %T", tn, ot, t, t)
			continue
		}
		typesMap[tn] = t
	}
	return typesMap
}
