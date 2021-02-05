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

package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash/crc64"
	"io"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-git/go-git/v5"
	"github.com/konveyor/move2kube/internal/assets"
	"github.com/konveyor/move2kube/types"
	log "github.com/sirupsen/logrus"
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
		log.Warnf("Error in walking through files due to : %q", err)
		return nil, err
	} else if !info.IsDir() {
		log.Warnf("The path %q is not a directory.", inputPath)
	}
	err := filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil && path == inputPath { // if walk for root search path return gets error
			// then stop walking and return this error
			return err
		}
		if err != nil {
			log.Warnf("Skipping path %q due to error: %q", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
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
		log.Warnf("Error in walking through files due to : %q", err)
		return files, err
	}
	log.Debugf("No of files with %s ext identified : %d", exts, len(files))
	return files, nil
}

//GetFilesByName returns files by name
func GetFilesByName(inputPath string, names []string) ([]string, error) {
	var files []string
	if info, err := os.Stat(inputPath); os.IsNotExist(err) {
		log.Warnf("Error in walking through files due to : %q", err)
		return files, err
	} else if !info.IsDir() {
		log.Warnf("The path %q is not a directory.", inputPath)
	}
	err := filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
		if err != nil && path == inputPath { // if walk for root search path return gets error
			// then stop walking and return this error
			return err
		}
		if err != nil {
			log.Warnf("Skipping path %q due to error: %q", path, err)
			return nil
		}
		// Skip directories
		if info.IsDir() {
			return nil
		}
		fname := filepath.Base(path)
		for _, name := range names {
			if fname == name {
				files = append(files, path)
			}
		}
		return nil
	})
	if err != nil {
		log.Warnf("Error in walking through files due to : %s", err)
		return files, err
	}
	log.Debugf("No of files with %s names identified : %d", names, len(files))
	return files, nil
}

//YamlAttrPresent returns YAML attributes
func YamlAttrPresent(path string, attr string) (bool, interface{}) {
	yamlFile, err := ioutil.ReadFile(path)
	if err != nil {
		log.Warnf("Error in reading yaml file %s: %s. Skipping", path, err)
		return false, nil
	}
	var fileContents map[string]interface{}
	err = yaml.Unmarshal(yamlFile, &fileContents)
	if err != nil {
		log.Warnf("Error in unmarshalling yaml file %s: %s. Skipping", path, err)
		return false, nil
	}
	if value, ok := fileContents[attr]; ok {
		log.Debugf("%s file has %s attribute", path, attr)
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

// WriteYaml writes an yaml to disk
func WriteYaml(outputPath string, data interface{}) error {
	var b bytes.Buffer
	encoder := yaml.NewEncoder(&b)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		log.Error("Error while Encoding object")
		return err
	}
	if err := encoder.Close(); err != nil {
		log.Error("Error while closing the encoder. Data not written to file", outputPath, "Error:", err)
		return err
	}
	err := ioutil.WriteFile(outputPath, b.Bytes(), DefaultFilePermission)
	if err != nil {
		log.Errorf("Error writing yaml to file. error: %s,  outputPath %s", err, outputPath)
		return err
	}
	return nil
}

// ReadYaml reads an yaml into an object
func ReadYaml(file string, data interface{}) error {
	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		log.Debugf("Error in reading yaml file %s: %s.", file, err)
		return err
	}
	err = yaml.Unmarshal(yamlFile, data)
	if err != nil {
		log.Debugf("Error in unmarshalling yaml file %s: %s.", file, err)
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
					log.Warnf("The application file (%s) was generated using a different version than (%s)", val, types.SchemeGroupVersion.String())
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
	yamlData, err := ioutil.ReadFile(path)
	if err != nil {
		log.Debugf("Failed to read the yaml file at path %s Error: %q", path, err)
		return err
	}
	yamlMap := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(yamlData), yamlMap); err != nil {
		log.Debugf("Error occurred while unmarshalling yaml file at path %s Error: %q", path, err)
		return err
	}
	groupVersionI, ok := yamlMap["apiVersion"]
	if !ok {
		err := fmt.Errorf("Did not find apiVersion in the yaml file at path %s", path)
		log.Debug(err)
		return err
	}
	groupVersionStr, ok := groupVersionI.(string)
	if !ok {
		err := fmt.Errorf("The apiVersion is not a string in the yaml file at path %s", path)
		log.Debug(err)
		return err
	}
	groupVersion, err := schema.ParseGroupVersion(groupVersionStr)
	if err != nil {
		log.Debugf("Failed to parse the apiVersion %s Error: %q", groupVersionStr, err)
		return err
	}
	if groupVersion.Group != types.SchemeGroupVersion.Group {
		err := fmt.Errorf("The file at path %s doesn't have the correct group. Expected group %s Actual group %s", path, types.SchemeGroupVersion.Group, groupVersion.Group)
		log.Debug(err)
		return err
	}
	if groupVersion.Version != types.SchemeGroupVersion.Version {
		log.Warnf("The file at path %s was generated using a different version. File version is %s and move2kube version is %s", path, groupVersion.Version, types.SchemeGroupVersion.Version)
	}
	if err := yaml.Unmarshal(yamlData, out); err != nil {
		log.Debugf("Error occurred while unmarshalling yaml file at path %s Error: %q", path, err)
		return err
	}
	return nil
}

// WriteJSON writes an json to disk
func WriteJSON(outputPath string, data interface{}) error {
	var b bytes.Buffer
	encoder := json.NewEncoder(&b)
	if err := encoder.Encode(data); err != nil {
		log.Error("Error while Encoding object")
		return err
	}
	err := ioutil.WriteFile(outputPath, b.Bytes(), DefaultFilePermission)
	if err != nil {
		log.Errorf("Error writing json to file: %s", err)
		return err
	}
	return nil
}

// ReadJSON reads an json into an object
func ReadJSON(file string, data interface{}) error {
	jsonFile, err := ioutil.ReadFile(file)
	if err != nil {
		log.Debugf("Error in reading json file %s: %s.", file, err)
		return err
	}
	err = json.Unmarshal(jsonFile, &data)
	if err != nil {
		log.Debugf("Error in unmarshalling json file %s: %s.", file, err)
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
	re := regexp.MustCompile("[._]")
	newName := strings.ToLower(re.ReplaceAllString(svcName, "-"))
	if newName != svcName {
		log.Infof("Changing service name to %s from %s", svcName, newName)
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
func MergeStringSlices(slice1 []string, slice2 []string) []string {
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

// GetStringFromTemplate returns string for a template
func GetStringFromTemplate(tpl string, config interface{}) (string, error) {
	var tplbuffer bytes.Buffer
	var packageTemplate = template.Must(template.New("").Parse(tpl))
	err := packageTemplate.Execute(&tplbuffer, config)
	if err != nil {
		log.Warnf("Unable to translate template %q to string using the data %v", tpl, config)
		return "", err
	}
	return tplbuffer.String(), nil
}

// WriteTemplateToFile writes a templated string to a file
func WriteTemplateToFile(tpl string, config interface{}, writepath string, filemode os.FileMode) error {
	var tplbuffer bytes.Buffer
	var packageTemplate = template.Must(template.New("").Parse(tpl))
	err := packageTemplate.Execute(&tplbuffer, config)
	if err != nil {
		log.Warnf("Unable to translate template %q to string using the data %v", tpl, config)
		return err
	}
	err = ioutil.WriteFile(writepath, tplbuffer.Bytes(), filemode)
	if err != nil {
		log.Warnf("Error writing file at %s : %s", writepath, err)
		return err
	}
	return nil
}

// GetClosestMatchingString returns the closest matching string for a given search string
func GetClosestMatchingString(options []string, searchstring string) string {
	// tokenize all strings
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	searchstring = reg.ReplaceAllString(searchstring, "")
	searchstring = strings.ToLower(searchstring)

	leastDistance := math.MaxInt32
	matchString := ""

	// Simply find the option with least distance
	for _, option := range options {
		// do tokensize the search space string too
		tokenizedOption := reg.ReplaceAllString(option, "")
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
	mergedmap := map[string]string{}
	for k, v := range map1 {
		mergedmap[k] = v
	}
	for k, v := range map2 {
		mergedmap[k] = v
	}
	return mergedmap
}

// MakeFileNameCompliant returns a DNS-1123 standard string
// Motivated by https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set
// Also see page 1 "ASSUMPTIONS" heading of https://tools.ietf.org/html/rfc952
// Also see page 13 of https://tools.ietf.org/html/rfc1123#page-13
func MakeFileNameCompliant(name string) string {
	if len(name) == 0 {
		log.Error("The input name is empty.")
		return ""
	}
	baseName := filepath.Base(name)
	invalidChars := regexp.MustCompile("[^a-zA-Z0-9-.]+")
	processedName := invalidChars.ReplaceAllString(baseName, "-")
	if len(processedName) > 63 {
		log.Debugf("Warning: The processed name %q is longer than 63 characters long.", processedName)
	}
	first := processedName[0]
	last := processedName[len(processedName)-1]
	if first == '-' || first == '.' || last == '-' || last == '.' {
		log.Debugf("Warning: The first and/or last characters of the name %q are not alphanumeric.", processedName)
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
	name = regexp.MustCompile(`[^a-z0-9-.]`).ReplaceAllString(name, "-")
	start, end := name[0], name[len(name)-1]
	if start == '-' || start == '.' || end == '-' || end == '.' {
		log.Warnf("The first and/or last characters of the string %q are not alphanumeric.", s)
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
func CreateAssetsData() (assetsPath string, tempPath string, err error) {
	// Return the absolute version of existing asset paths.
	tempPath, err = filepath.Abs(TempPath)
	if err != nil {
		log.Errorf("Unable to make the temporary directory path %q absolute. Error: %q", tempPath, err)
		return "", "", err
	}
	assetsPath, err = filepath.Abs(AssetsPath)
	if err != nil {
		log.Errorf("Unable to make the assets path %q absolute. Error: %q", assetsPath, err)
		return "", "", err
	}

	// Try to create a new temporary directory for the assets.
	if newTempPath, err := ioutil.TempDir("", TempDirPrefix); err != nil {
		log.Errorf("Unable to create temp dir. Defaulting to local path.")
	} else {
		tempPath = newTempPath
		assetsPath = filepath.Join(newTempPath, AssetsDir)
	}

	// Either way create the subdirectory and untar the assets into it.
	if err := os.MkdirAll(assetsPath, DefaultDirectoryPermission); err != nil {
		log.Errorf("Unable to create the assets directory at path %q Error: %q", assetsPath, err)
		return "", "", err
	}
	if err := UnTarString(assets.Tar, assetsPath); err != nil {
		log.Errorf("Unable to untar the assets into the assets directory at path %q Error: %q", assetsPath, err)
		return "", "", err
	}

	return assetsPath, tempPath, nil
}

// CopyFile copies a file from src to dst.
// The dst file will be truncated if it exists.
// Returns an error if it failed to copy all the bytes.
func CopyFile(dst, src string) error {
	srcfile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("Failed to open the source file at path %q Error: %q", src, err)
	}
	defer srcfile.Close()

	srcfileinfo, err := srcfile.Stat()
	if err != nil {
		return fmt.Errorf("Failed to get size of the source file at path %q Error: %q", src, err)
	}
	srcfilesize := srcfileinfo.Size()

	dstfile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, DefaultFilePermission)
	if err != nil {
		return fmt.Errorf("Failed to create the destination file at path %q Error: %q", dst, err)
	}
	defer dstfile.Close()

	written, err := io.Copy(dstfile, srcfile)
	if written != srcfilesize {
		return fmt.Errorf("Failed to copy all the bytes from source %q to destination %q. %d out of %d bytes written. Error: %v", src, dst, written, srcfilesize, err)
	}
	if err != nil {
		return fmt.Errorf("Failed to copy from source %q to destination %q. Error: %q", src, dst, err)
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

// GetGitRemoteNames returns a list of remotes if there is a repo and remotes exists.
func GetGitRemoteNames(path string) ([]string, error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}
	remotes, err := repo.Remotes()
	if err != nil {
		return nil, err
	}
	remoteNames := []string{}
	for _, remote := range remotes {
		remoteNames = append(remoteNames, remote.Config().Name)
	}
	return remoteNames, nil
}

// GetGitRepoDetails returns the remote urls for a git repo at path.
func GetGitRepoDetails(path, remoteName string) (remoteURLs []string, branch string, repoDir string, finalerr error) {
	repo, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		log.Debugf("Unable to open the path %q as a git repo. Error: %q", path, err)
		return nil, "", "", err
	}

	if workTree, err := repo.Worktree(); err == nil {
		repoDir = workTree.Filesystem.Root()
	} else {
		log.Debugf("Unable to get the repo directory. Error: %q", err)
	}

	if ref, err := repo.Head(); err == nil {
		branch = filepath.Base(string(ref.Name()))
	} else {
		log.Debugf("Unable to get the current branch. Error: %q", err)
	}

	if remote, err := repo.Remote(remoteName); err == nil {
		remoteURLs = remote.Config().URLs
	} else {
		log.Debugf("Unable to get remote named %s Error: %q", remoteName, err)
	}

	return remoteURLs, branch, repoDir, nil
}

// GetGitRepoName returns the remote repo's name and context.
func GetGitRepoName(path string) (repo string, root string) {
	r, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		log.Debugf("Unable to open %s as a git repo : %s", path, err)
		return "", ""
	}
	remote, err := r.Remote("origin")
	if err != nil {
		log.Debugf("Unable to get origin remote : %s", err)
		return "", ""
	}
	if len(remote.Config().URLs) == 0 {
		log.Debugf("Unable to get origins")
		return "", ""
	}
	u := remote.Config().URLs[0]
	if strings.HasPrefix(u, "git") {
		parts := strings.Split(u, ":")
		if len(parts) != 2 {
			// Unable to find git repo name
			return "", ""
		}
		u = parts[1]
	}
	giturl, err := url.Parse(u)
	if err != nil {
		log.Debugf("Unable to get origin remote host : %s", err)
		return "", ""
	}
	name := filepath.Base(giturl.Path)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	w, err := r.Worktree()
	if err != nil {
		log.Warnf("Unable to get root of repo : %s", err)
	}
	return name, w.Filesystem.Root()
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

// GetGVK returns the group version kind given a k8s resource object.
func GetGVK(obj runtime.Object) schema.GroupVersionKind {
	k8sObjValue := reflect.ValueOf(obj).Elem()
	typeMeta, ok := k8sObjValue.FieldByName("TypeMeta").Interface().(metav1.TypeMeta)
	if !ok {
		log.Fatal("Failed to retrieve object type metadata")
	}
	return typeMeta.GroupVersionKind()
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
		log.Fatalf("Failed to make the path %s absolute. Error: %s", child, err)
	}
	parent, err = filepath.Abs(parent)
	if err != nil {
		log.Fatalf("Failed to make the path %s absolute. Error: %s", parent, err)
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

// SplitOnDotExpectInsideQuotes splits a string on dot.
// Stuff inside double or single quotes will not be split.
func SplitOnDotExpectInsideQuotes(s string) []string {
	return regexp.MustCompile(`[^."']+|"[^"]*"|'[^']*'`).FindAllString(s, -1)
}
