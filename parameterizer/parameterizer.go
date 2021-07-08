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
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/common/deepcopy"
	"github.com/konveyor/move2kube/internal/k8sschema"
	"github.com/konveyor/move2kube/qaengine"
	parameterizertypes "github.com/konveyor/move2kube/types/parameterizer"
	qatypes "github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	stringInterpRegex            = regexp.MustCompile(`\${([^}]+)}`)
	invalidOCTemplateChars       = regexp.MustCompile(`[^a-zA-Z0-9_]+`)
	specialJSONPathChars         = regexp.MustCompile(`(~|\/)`)
	templateInnerParametersRegex = regexp.MustCompile(`\$\([^)]+\)`)
)

// Parameterize does the parameterization based on a spec
func Parameterize(srcDir, outDir string, packSpecPath parameterizertypes.PackagingSpecPathT, ps []parameterizertypes.ParameterizerT) ([]string, error) {
	filesWritten := []string{}
	cleanSrcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return nil, err
	}
	cleanOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return nil, err
	}
	if packSpecPath.Helm == "" {
		packSpecPath.Helm = filepath.Join(packSpecPath.Out, "helm-chart")
	}
	if packSpecPath.Kustomize == "" {
		packSpecPath.Kustomize = filepath.Join(packSpecPath.Out, "kustomize")
	}
	if packSpecPath.OCTemplates == "" {
		packSpecPath.OCTemplates = filepath.Join(packSpecPath.Out, "openshift-template")
	}
	if len(packSpecPath.Envs) == 0 {
		packSpecPath.Envs = []string{"dev", "staging", "prod"}
	}
	pathedKs, err := k8sschema.GetK8sResourcesWithPaths(filepath.Join(cleanSrcDir, packSpecPath.Src))
	if err != nil {
		return filesWritten, err
	}
	if packSpecPath.Helm != "" {
		// helm chart with multiple values.yaml
		helmChartName := packSpecPath.HelmChartName
		if helmChartName == "" {
			helmChartName = common.DefaultProjectName
		}
		namedValues := map[string]parameterizertypes.HelmValuesT{}
		helmChartDir := filepath.Join(cleanOutDir, packSpecPath.Helm, helmChartName)
		helmTemplatesDir := filepath.Join(helmChartDir, "templates")
		if err := os.MkdirAll(helmTemplatesDir, common.DefaultDirectoryPermission); err != nil {
			return filesWritten, err
		}
		for kPath, ks := range pathedKs {
			for _, k := range ks {
				k = deepcopy.DeepCopy(k).(parameterizertypes.K8sResourceT)
				if err := parameterize(parameterizertypes.TargetHelm, packSpecPath.Envs, k, ps, namedValues, nil, nil); err != nil {
					return filesWritten, err
				}
				finalKPath := filepath.Join(helmTemplatesDir, kPath)
				if err := writeResourceStripQuotesAndAppendToFile(k, finalKPath); err != nil {
					return filesWritten, err
				}
				filesWritten = append(filesWritten, finalKPath)
			}
		}
		for env, values := range namedValues {
			finalKPath := filepath.Join(helmChartDir, "values-"+env+".yaml")
			if err := common.WriteYaml(finalKPath, values); err != nil {
				return filesWritten, err
			}
			filesWritten = append(filesWritten, finalKPath)
		}
		helmChartYaml := map[string]interface{}{
			"apiVersion":  "v2",
			"name":        helmChartName,
			"version":     "0.1.0",
			"description": "A Helm Chart generated by Move2Kube for " + helmChartName,
			"keywords":    []string{helmChartName},
		}
		finalKPath := filepath.Join(helmChartDir, "Chart.yaml")
		if err := common.WriteYaml(finalKPath, helmChartYaml); err != nil {
			return filesWritten, err
		}
		filesWritten = append(filesWritten, finalKPath)
	}
	if packSpecPath.Kustomize != "" {
		// kustomize json patches with multiple overlays
		kustDir := filepath.Join(cleanOutDir, packSpecPath.Kustomize)
		baseDir := filepath.Join(kustDir, "base")
		if err := os.MkdirAll(baseDir, common.DefaultDirectoryPermission); err != nil {
			return filesWritten, err
		}
		kustPatches := map[string]map[parameterizertypes.PatchMetadataT][]parameterizertypes.PatchT{}
		kPaths := []string{}
		for kPath, ks := range pathedKs {
			for _, k := range ks {
				// base
				finalKPath := filepath.Join(baseDir, kPath)
				if err := writeResourceAppendToFile(k, finalKPath); err != nil {
					return filesWritten, err
				}
				filesWritten = append(filesWritten, finalKPath)
				// compute the json patch
				currKustPatches := map[string]map[string]parameterizertypes.PatchT{} // keyed by env and json pointer/path
				if err := parameterize(parameterizertypes.TargetKustomize, packSpecPath.Envs, k, ps, nil, currKustPatches, nil); err != nil {
					return filesWritten, err
				}
				// patch metadata to put in kustomization.yaml
				group, version, kind, metadataName, err := getGVKNFromK(k)
				if err != nil {
					return filesWritten, err
				}
				patchFilename := fmt.Sprintf("%s-%s-%s-%s.yaml", group, version, kind, metadataName)
				if group == "" {
					patchFilename = fmt.Sprintf("%s-%s-%s.yaml", version, kind, metadataName)
				}
				patchFilename = strings.ToLower(common.MakeFileNameCompliant(patchFilename))
				patchMetadata := parameterizertypes.PatchMetadataT{
					Path:   patchFilename,
					Target: parameterizertypes.PatchMetadataTargetT{Kind: kind, Group: group, Version: version, Name: metadataName},
				}
				for env, patches := range currKustPatches {
					if _, ok := kustPatches[env]; !ok {
						kustPatches[env] = map[parameterizertypes.PatchMetadataT][]parameterizertypes.PatchT{}
					}
					for _, v := range patches {
						kustPatches[env][patchMetadata] = append(kustPatches[env][patchMetadata], v)
					}
				}
				kPaths = append(kPaths, kPath)
			}
			kustomization := map[string]interface{}{"resources": kPaths}
			finalKPath := filepath.Join(baseDir, "kustomization.yaml")
			if err := common.WriteYaml(finalKPath, kustomization); err != nil {
				return filesWritten, err
			}
			filesWritten = append(filesWritten, finalKPath)
		}
		// create a overlay for each env
		for env, kMetaPatches := range kustPatches {
			envDir := filepath.Join(kustDir, "overlays", env)
			if err := os.MkdirAll(envDir, common.DefaultDirectoryPermission); err != nil {
				return filesWritten, err
			}
			metas := []parameterizertypes.PatchMetadataT{}
			for kMeta, patches := range kMetaPatches {
				finalKPath := filepath.Join(envDir, kMeta.Path)
				if err := common.WriteYaml(finalKPath, patches); err != nil {
					return filesWritten, err
				}
				metas = append(metas, kMeta)
				filesWritten = append(filesWritten, finalKPath)
			}
			kustomization := map[string]interface{}{"resources": []string{"../../base"}, "patches": metas}
			finalKPath := filepath.Join(envDir, "kustomization.yaml")
			if err := common.WriteYaml(finalKPath, kustomization); err != nil {
				return filesWritten, err
			}
			filesWritten = append(filesWritten, finalKPath)
		}
	}
	if packSpecPath.OCTemplates != "" {
		// openshift templates for each env
		newKs := []parameterizertypes.K8sResourceT{}
		ocParams := map[string]map[string]string{}
		for _, ks := range pathedKs {
			for _, k := range ks {
				k = deepcopy.DeepCopy(k).(parameterizertypes.K8sResourceT)
				if err := parameterize(parameterizertypes.TargetOCTemplates, packSpecPath.Envs, k, ps, nil, nil, ocParams); err != nil {
					return filesWritten, err
				}
				newKs = append(newKs, k)
			}
		}
		singleSet := []parameterizertypes.OCParamT{}
		if len(ocParams) > 0 {
			for _, kvs := range ocParams {
				for k, v := range kvs {
					singleSet = append(singleSet, parameterizertypes.OCParamT{Name: k, Value: v})
				}
				break
			}
		}
		templ := map[string]interface{}{
			"apiVersion": "template.openshift.io/v1",
			"kind":       "Template",
			"metadata":   metav1.ObjectMeta{Name: common.MakeStringDNSNameCompliant(common.DefaultProjectName + "-template")},
			"objects":    newKs,
			"parameters": singleSet,
		}
		ocDir := filepath.Join(cleanOutDir, packSpecPath.OCTemplates)
		if err := os.MkdirAll(ocDir, common.DefaultDirectoryPermission); err != nil {
			return filesWritten, err
		}
		finalKPath := filepath.Join(ocDir, "template.yaml")
		if err := common.WriteYaml(finalKPath, templ); err != nil {
			return filesWritten, err
		}
		filesWritten = append(filesWritten, finalKPath)
		for env, params := range ocParams {
			finalKPath := filepath.Join(ocDir, "parameters-"+env+".yaml")
			finalParams := []string{}
			for k, v := range params {
				finalParams = append(finalParams, fmt.Sprintf("%s=%s", k, v))
			}
			if err := ioutil.WriteFile(finalKPath, []byte(strings.Join(finalParams, "\n")), common.DefaultFilePermission); err != nil {
				return filesWritten, err
			}
			filesWritten = append(filesWritten, finalKPath)
		}
	}
	return filesWritten, nil
}

// ------------------------------
// Utilities

func getGVKNFromK(k parameterizertypes.K8sResourceT) (group string, version string, kind string, metadataName string, err error) {
	var apiVersion string
	kind, apiVersion, metadataName, err = k8sschema.GetInfoFromK8sResource(k)
	if err != nil {
		return kind, "", "", metadataName, err
	}
	t1s := strings.Split(apiVersion, "/")
	if len(t1s) == 0 || len(t1s) > 2 {
		err = fmt.Errorf("failed to get group and version from %s", apiVersion)
		return kind, apiVersion, apiVersion, metadataName, err
	}
	if len(t1s) == 1 {
		version = t1s[0]
	} else if len(t1s) == 2 {
		group, version = t1s[0], t1s[1]
	}
	return group, version, kind, metadataName, nil
}

func getParameters(templ string) ([]string, error) {
	matches := stringInterpRegex.FindAllStringSubmatch(templ, -1)
	if len(matches) == 0 {
		return nil, fmt.Errorf("no parameters found in the template: %s", templ)
	}
	parameters := []string{}
	for _, match := range matches {
		parameters = append(parameters, match[1])
	}
	return parameters, nil
}

func doesMatchEnv(p parameterizertypes.ParameterValueT, env, kind, apiVersion, metadataName string, matches map[string]string) bool {
	if p.Envs != nil && !common.IsStringPresent(p.Envs, string(env)) {
		return false
	}
	if p.Kind != "" && p.Kind != kind {
		return false
	}
	if p.APIVersion != "" && p.APIVersion != apiVersion {
		return false
	}
	if p.MetadataName != "" && p.MetadataName != metadataName {
		return false
	}
	for k, v := range p.Custom {
		if matches[k] != v {
			return false
		}
	}
	return true
}

func parseTemplate(templ, orig, regex string) ([]string, []parameterizertypes.ParamOrStringT, error) {
	parameters := stringInterpRegex.FindAllStringSubmatchIndex(templ, -1)
	if len(parameters) == 0 {
		return nil, nil, fmt.Errorf("no parameters found in the template: %s", templ)
	}
	idxs := []int{}
	for _, match := range parameters {
		// assume len(match) == 4 (2 idxs for the original string, 2 for the capturing group). Not sure when this can be false.
		idxs = append(idxs, match[:2]...)
	}
	paramsAndStrings := splitOnIdxs(templ, idxs)
	var newre *regexp.Regexp
	if regex == "" {
		var err error
		newre, err = regexp.Compile("^" + stringInterpRegex.ReplaceAllLiteralString(templ, `(.+)`) + "$")
		if err != nil {
			return nil, paramsAndStrings, fmt.Errorf("failed to make a new regex using the template: %s\nError: %q", templ, err)
		}
	} else {
		var err error
		newre, err = regexp.Compile("^" + regex + "$")
		if err != nil {
			return nil, paramsAndStrings, fmt.Errorf("the given regex is invalid: %s\nError: %q", regex, err)
		}
	}
	orignalValues := newre.FindAllStringSubmatch(orig, -1)
	if len(orignalValues) == 0 {
		return nil, paramsAndStrings, fmt.Errorf("failed to extract the orignal values from the string %s for the template %s using the regex %+v", orig, templ, newre)
	}
	if len(orignalValues[0]) != len(parameters)+1 {
		return nil, paramsAndStrings, fmt.Errorf("expected to find %d matches (one for each parameter). Actual matches are %+v", len(parameters), orignalValues)
	}
	// skip the first element which is the full matched string.
	return orignalValues[0][1:], paramsAndStrings, nil
}

func splitOnIdxs(s string, idxs []int) []parameterizertypes.ParamOrStringT {
	ss := []parameterizertypes.ParamOrStringT{}
	prevIdx := 0
	isParam := false
	for _, idx := range idxs {
		currs := parameterizertypes.ParamOrStringT{IsParam: isParam, Data: s[prevIdx:idx]}
		prevIdx = idx
		isParam = !isParam
		ss = append(ss, currs)
	}
	return ss
}

func subKeysToJSONPointer6901(subKeys []string) string {
	for i, subKey := range subKeys {
		if strings.HasPrefix(subKey, "[") && strings.HasSuffix(subKey, "]") {
			subKeys[i] = strings.TrimSuffix(strings.TrimPrefix(subKey, "["), "]")
			continue
		}
		subKeys[i] = specialJSONPathChars.ReplaceAllStringFunc(subKey, escapeForJSONPath)
	}
	return "/" + strings.Join(subKeys, "/")
}

func escapeForJSONPath(x string) string {
	if x == "~" {
		return "~0"
	}
	if x == "/" {
		return "~1"
	}
	return x
}

// fillCustomTemplate is used to fill in templates
func fillCustomTemplate(templ, kind, apiVersion, metadataName string, matches map[string]string) (string, error) {
	var errs []string
	result := templateInnerParametersRegex.ReplaceAllStringFunc(templ, func(s string) string {
		s = strings.TrimSuffix(strings.TrimPrefix(s, "$("), ")")
		sV, ok := matches[s]
		if ok {
			return sV
		}
		switch s {
		case "$(kind)":
			return "Deployment"
		case "$(apiVersion)":
			return "move2kube.konveyor.io/v1alpha1"
		case "$(metadataName)":
			return ""
		default:
			errs = append(errs, fmt.Sprintf("failed to find the key %s in the matches %+v", s, matches))
			return ""
		}
	})
	if len(errs) > 0 {
		return result, fmt.Errorf("%s", strings.Join(errs, "\n"))
	}
	return result, nil
}

// ------------------------------
// Parameterization

func parameterize(target parameterizertypes.ParamTargetT, envs []string, k parameterizertypes.K8sResourceT, ps []parameterizertypes.ParameterizerT, namedValues map[string]parameterizertypes.HelmValuesT, namedKustPatches map[string]map[string]parameterizertypes.PatchT, namedOCParams map[string]map[string]string) error {
	for _, p := range ps {
		ok, err := parameterizeFilter(envs, k, p)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		switch target {
		case parameterizertypes.TargetHelm:
			if err := parameterizeHelperHelm(envs, k, p, namedValues, namedKustPatches, namedOCParams); err != nil {
				return err
			}
		case parameterizertypes.TargetKustomize:
			if err := parameterizeHelperKustomize(envs, k, p, namedValues, namedKustPatches, namedOCParams); err != nil {
				return err
			}
		case parameterizertypes.TargetOCTemplates:
			if err := parameterizeHelperOCTemplates(envs, k, p, namedValues, namedKustPatches, namedOCParams); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported parameterization type: %+v", target)
		}
	}
	return nil
}

// parameterizeFilter returns true if this parameterizer can be applied to the given k8s resource
func parameterizeFilter(envs []string, k parameterizertypes.K8sResourceT, p parameterizertypes.ParameterizerT) (bool, error) {
	log.Trace("start parameterizeFilter")
	defer log.Trace("end parameterizeFilter")
	kind, apiVersion, metadataName, err := k8sschema.GetInfoFromK8sResource(k)
	if err != nil {
		return false, err
	}
	if len(p.Filters) == 0 {
		// empty map matches all kinds, apiVersions and names
		return true, nil
	}
	for _, filter := range p.Filters {
		// empty kind matches all kinds
		if filter.Kind != "" {
			re, err := regexp.Compile("^" + filter.Kind + "$")
			if err != nil {
				return false, err
			}
			if !re.MatchString(kind) {
				continue
			}
		}
		// empty apiVersion matches all apiVersions
		if filter.APIVersion != "" {
			re, err := regexp.Compile("^" + filter.APIVersion + "$")
			if err != nil {
				return false, err
			}
			if !re.MatchString(apiVersion) {
				continue
			}
		}
		// empty name matches all names
		if filter.Name != "" {
			re, err := regexp.Compile("^" + filter.Name + "$")
			if err != nil {
				return false, err
			}
			if !re.MatchString(metadataName) {
				continue
			}
		}
		if filter.Envs != nil {
			found := false
			for _, env := range envs {
				if common.IsStringPresent(filter.Envs, env) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		return true, nil
	}
	return false, nil
}

func parameterizeHelperHelm(envs []string, k parameterizertypes.K8sResourceT, p parameterizertypes.ParameterizerT, namedValues map[string]parameterizertypes.HelmValuesT, namedKustPatches map[string]map[string]parameterizertypes.PatchT, namedOCParams map[string]map[string]string) error {
	log.Trace("start parameterizeHelperHelm")
	defer log.Trace("end parameterizeHelperHelm")

	if len(p.Target) == 0 {
		return fmt.Errorf("the target is empty")
	}
	kind, apiVersion, metadataName, err := k8sschema.GetInfoFromK8sResource(k)
	if err != nil {
		return fmt.Errorf("failed to get the kind, apiVersion, and name from the k8s resource: %+v\nError: %q", k, err)
	}
	resultKVs, err := GetAll(p.Target, k)
	if err != nil {
		return fmt.Errorf("the key %s does not exist on the k8s resource: %+v Error: %q", p.Target, k, err)
	}
	for _, resultKV := range resultKVs {
		t1 := []string{}
		for _, k := range resultKV.Key {
			t1 = append(t1, `"`+k+`"`)
		}
		key := strings.Join(t1, ".")
		templ := p.Template
		if templ == "" {
			templ = fmt.Sprintf(`${"%s"."%s"."%s".%s}`, kind, apiVersion, metadataName, key)
		}
		parameters, err := getParameters(templ)
		if err != nil {
			return fmt.Errorf("failed to get the parameters from the template: %s\nError: %q", templ, err)
		}
		paramValue := p.Default
		if paramValue == nil {
			paramValue = resultKV.Value
		}
		if p.Question != nil {
			if p.Question.Type == "" {
				p.Question.Type = qatypes.InputSolutionFormType
			}
			flagNoID := p.Question.ID == ""
			if flagNoID {
				p.Question.ID = parameterizertypes.ParamQuesIDPrefix + common.Delim + apiVersion + common.Delim + kind + common.Delim + key
			}
			filledDesc, err := fillCustomTemplate(p.Question.Desc, kind, apiVersion, metadataName, resultKV.Matches)
			if err != nil {
				return err
			}
			origQuesDesc := p.Question.Desc
			p.Question.Desc = filledDesc
			ques, err := qaengine.FetchAnswer(*p.Question)
			if err != nil {
				return fmt.Errorf("failed to ask a question to the user in order to parameterize a k8s resource: %+v\nError: %q", p.Question, err)
			}
			var ok bool
			paramValue, ok = ques.Answer.(string)
			if !ok {
				return fmt.Errorf("failed to ask a question to the user in order to parameterize a k8s resource: %+v\nError: the answer was not a string", p.Question)
			}
			if flagNoID {
				p.Question.ID = ""
			}
			p.Question.Desc = origQuesDesc
		}
		if len(parameters) == 1 {
			parameter := parameters[0]
			subKeys := GetSubKeys(parameter)
			for i, subKey := range subKeys {
				if !strings.HasPrefix(subKey, "$(") || !strings.HasSuffix(subKey, ")") {
					subKeys[i] = `"` + subKey + `"`
					continue
				}
				subKey = strings.TrimSuffix(strings.TrimPrefix(subKey, "$("), ")")
				matchedSubKey, ok := resultKV.Matches[subKey]
				if ok {
					subKeys[i] = `"` + matchedSubKey + `"`
					continue
				}
				switch subKey {
				case "kind":
					subKeys[i] = `"` + kind + `"`
				case "apiVersion":
					subKeys[i] = `"` + apiVersion + `"`
				case "metadataName":
					subKeys[i] = `"` + metadataName + `"`
				default:
					return fmt.Errorf("failed to find the sub key $(%s) in the any of the keys that matched: %+v", subKey, resultKV)
				}
			}
			paramKey := strings.Join(subKeys, ".")
			helmTemplate := fmt.Sprintf(`{{ index .Values %s }}`, strings.Join(subKeys, " "))
			if len(p.Parameters) > 0 {
				if len(p.Parameters) != 1 {
					return fmt.Errorf("the template only has a single parameter. Expected a single paramter definition. Actual length: %d Parameters: %+v", len(p.Parameters), p.Parameters)
				}
				param := p.Parameters[0]
				if param.Name != parameter {
					return fmt.Errorf("the name in the paramter definition doesn't match the name in the template. Parameters: %+v", param)
				}
				if param.HelmTemplate != "" {
					helmTemplate = param.HelmTemplate
				}
				if param.Default != "" {
					paramValue = param.Default
				}
			}
			if err := set(key, helmTemplate, k); err != nil {
				return fmt.Errorf("failed to set the key %s to the value %s in the k8s resource: %+v\nError: %q", key, helmTemplate, k, err)
			}
			for _, env := range envs {
				origParamValue := paramValue
				if len(p.Parameters) > 0 {
					param := p.Parameters[0]
					for _, pV := range param.Values {
						if doesMatchEnv(pV, env, kind, apiVersion, metadataName, resultKV.Matches) {
							paramValue = pV.Value
							break
						}
					}
				}
				// set the key in the values.yaml
				if _, ok := namedValues[env]; !ok {
					namedValues[env] = parameterizertypes.HelmValuesT{}
				}
				if err := setCreatingNew(paramKey, paramValue, namedValues[env]); err != nil {
					return fmt.Errorf("failed to set the key %s to the value %+v in the values.yaml %+v for the env %s . Error: %q", paramKey, paramValue, namedValues[env], env, err)
				}
				paramValue = origParamValue
			}
			return nil
		}
		// multiple parameters only make sense when the original value is a string
		originalValueStr, ok := resultKV.Value.(string)
		if !ok {
			return fmt.Errorf(
				`the template %s contains multiple parameters.
This only makes sense when the original value is a string.
Actual value is %+v of type %T`,
				templ, resultKV.Value, resultKV.Value,
			)
		}
		defaultStr := originalValueStr
		if paramValue != nil {
			defaultStr, ok = paramValue.(string)
			if !ok {
				return fmt.Errorf("the default parameter value is not a string. Actual value %+v is of type %T", paramValue, paramValue)
			}
		}
		originalValues, paramsAndStrings, err := parseTemplate(templ, defaultStr, p.Regex)
		if err != nil {
			return fmt.Errorf("failed to parse the multi parameter template: %s\nError: %q", templ, err)
		}
		helmTemplates := []string{}
		paramKeys := []string{}
		for _, parameter := range parameters {
			subKeys := GetSubKeys(parameter)
			for i, subKey := range subKeys {
				if !strings.HasPrefix(subKey, "$(") || !strings.HasSuffix(subKey, ")") {
					subKeys[i] = `"` + subKey + `"`
					continue
				}
				subKey = strings.TrimSuffix(strings.TrimPrefix(subKey, "$("), ")")
				matchedSubKey, ok := resultKV.Matches[subKey]
				if ok {
					subKeys[i] = `"` + matchedSubKey + `"`
					continue
				}
				switch subKey {
				case "kind":
					subKeys[i] = `"` + kind + `"`
				case "apiVersion":
					subKeys[i] = `"` + apiVersion + `"`
				case "metadataName":
					subKeys[i] = `"` + metadataName + `"`
				default:
					return fmt.Errorf("failed to find the sub key $(%s) in the any of the keys that matched: %+v", subKey, resultKV)
				}
			}
			paramKeys = append(paramKeys, strings.Join(subKeys, "."))
			helmTemplate := fmt.Sprintf(`{{ index .Values %s }}`, strings.Join(subKeys, " "))
			for _, stParameter := range p.Parameters {
				if stParameter.Name != parameter {
					continue
				}
				if stParameter.HelmTemplate != "" {
					helmTemplate = stParameter.HelmTemplate
				}
				break
			}
			helmTemplates = append(helmTemplates, helmTemplate)
		}
		fullHelmTemplate := ""
		helmTemplateIdx := 0
		for _, pOrS := range paramsAndStrings {
			if !pOrS.IsParam {
				fullHelmTemplate += pOrS.Data
				continue
			}
			currHelmTemplate := helmTemplates[helmTemplateIdx]
			for _, param := range p.Parameters {
				if "${"+param.Name+"}" != pOrS.Data {
					continue
				}
				if param.HelmTemplate != "" {
					currHelmTemplate = param.HelmTemplate
				}
				break
			}
			fullHelmTemplate += currHelmTemplate
			helmTemplateIdx++
		}
		if err := set(key, fullHelmTemplate, k); err != nil {
			return fmt.Errorf("failed to set the key %s to the value %s in the k8s resource: %+v\nError: %q", key, fullHelmTemplate, k, err)
		}
		// set all the keys in the values.yaml
		for i, parameter := range parameters {
			paramKey := paramKeys[i]
			paramValue := originalValues[i]
			for _, env := range envs {
				origParamValue := paramValue
				for _, param := range p.Parameters {
					if param.Name != parameter {
						continue
					}
					if param.Default != "" {
						paramValue = param.Default
					}
					for _, pV := range param.Values {
						if doesMatchEnv(pV, env, kind, apiVersion, metadataName, resultKV.Matches) {
							paramValue = pV.Value
							break
						}
					}
					break
				}
				// set the key in the values.yaml
				if _, ok := namedValues[env]; !ok {
					namedValues[env] = parameterizertypes.HelmValuesT{}
				}
				if err := setCreatingNew(paramKey, paramValue, namedValues[env]); err != nil {
					return fmt.Errorf("failed to set the key %s to the value %+v in the values.yaml %+v for the env %s . Error: %q", paramKey, paramValue, namedValues[env], env, err)
				}
				paramValue = origParamValue
			}
		}
	}
	return nil
}

func parameterizeHelperKustomize(envs []string, k parameterizertypes.K8sResourceT, p parameterizertypes.ParameterizerT, namedValues map[string]parameterizertypes.HelmValuesT, namedKustPatches map[string]map[string]parameterizertypes.PatchT, namedOCParams map[string]map[string]string) error {
	log.Trace("start parameterizeHelperKustomize")
	defer log.Trace("end parameterizeHelperKustomize")

	if len(p.Target) == 0 {
		return fmt.Errorf("the target is empty")
	}
	kind, apiVersion, metadataName, err := k8sschema.GetInfoFromK8sResource(k)
	if err != nil {
		return fmt.Errorf("failed to get the kind, apiVersion, and name from the k8s resource: %+v\nError: %q", k, err)
	}
	resultKVs, err := GetAll(p.Target, k)
	if err != nil {
		return fmt.Errorf("the key %s does not exist on the k8s resource: %+v Error: %q", p.Target, k, err)
	}
	for _, resultKV := range resultKVs {
		t1 := []string{}
		for _, k := range resultKV.Key {
			t1 = append(t1, `"`+k+`"`)
		}
		key := strings.Join(t1, ".")
		JSONPointer := subKeysToJSONPointer6901(resultKV.Key)
		paramValue := p.Default
		if paramValue == nil {
			paramValue = resultKV.Value
		}
		if p.Question != nil {
			if p.Question.Type == "" {
				p.Question.Type = qatypes.InputSolutionFormType
			}
			flagNoID := p.Question.ID == ""
			if flagNoID {
				p.Question.ID = parameterizertypes.ParamQuesIDPrefix + common.Delim + apiVersion + common.Delim + kind + common.Delim + key
			}
			filledDesc, err := fillCustomTemplate(p.Question.Desc, kind, apiVersion, metadataName, resultKV.Matches)
			if err != nil {
				return err
			}
			origQuesDesc := p.Question.Desc
			p.Question.Desc = filledDesc
			ques, err := qaengine.FetchAnswer(*p.Question)
			if err != nil {
				return fmt.Errorf("failed to ask a question to the user in order to parameterize a k8s resource: %+v\nError: %q", p.Question, err)
			}
			var ok bool
			paramValue, ok = ques.Answer.(string)
			if !ok {
				return fmt.Errorf("failed to ask a question to the user in order to parameterize a k8s resource: %+v\nError: the answer was not a string", p.Question)
			}
			if flagNoID {
				p.Question.ID = ""
			}
			p.Question.Desc = origQuesDesc
		}
		for _, env := range envs {
			origParamValue := paramValue
			if len(p.Parameters) > 0 {
				if len(p.Parameters) > 1 {
					log.Debugf("more than one parameter specified for kustomize parameterization, ignoring all of them. Actual length: %d Parameters: %+v", len(p.Parameters), p.Parameters)
				} else {
					param := p.Parameters[0]
					// no need to check the parameter name since for kustomize there should be at most one parameter
					for _, pV := range param.Values {
						if doesMatchEnv(pV, env, kind, apiVersion, metadataName, resultKV.Matches) {
							paramValue = pV.Value
							break
						}
					}
				}
			}
			if _, ok := namedKustPatches[env]; !ok {
				namedKustPatches[env] = map[string]parameterizertypes.PatchT{}
			}
			// set the key in the parameters.yaml
			namedKustPatches[env][JSONPointer] = parameterizertypes.PatchT{Op: parameterizertypes.ReplaceOp, Path: JSONPointer, Value: paramValue}
			paramValue = origParamValue
		}
	}
	return nil
}

func parameterizeHelperOCTemplates(envs []string, k parameterizertypes.K8sResourceT, p parameterizertypes.ParameterizerT, namedValues map[string]parameterizertypes.HelmValuesT, namedKustPatches map[string]map[string]parameterizertypes.PatchT, namedOCParams map[string]map[string]string) error {
	log.Trace("start parameterizeHelperOCTemplates")
	defer log.Trace("end parameterizeHelperOCTemplates")

	if len(p.Target) == 0 {
		return fmt.Errorf("the target is empty")
	}
	kind, apiVersion, metadataName, err := k8sschema.GetInfoFromK8sResource(k)
	if err != nil {
		return fmt.Errorf("failed to get the kind, apiVersion, and name from the k8s resource: %+v\nError: %q", k, err)
	}
	resultKVs, err := GetAll(p.Target, k)
	if err != nil {
		return fmt.Errorf("the key %s does not exist on the k8s resource: %+v Error: %q", p.Target, k, err)
	}
	for _, resultKV := range resultKVs {
		t1 := []string{}
		for _, k := range resultKV.Key {
			t1 = append(t1, `"`+k+`"`)
		}
		key := strings.Join(t1, ".")
		templ := p.Template
		if templ == "" {
			templ = fmt.Sprintf(`${"%s"."%s"."%s".%s}`, kind, apiVersion, metadataName, key)
		}
		parameters, err := getParameters(templ)
		if err != nil {
			return fmt.Errorf("failed to get the parameters from the template: %s\nError: %q", templ, err)
		}
		paramValue := p.Default
		if paramValue == nil {
			paramValue = resultKV.Value
		}
		if p.Question != nil {
			if p.Question.Type == "" {
				p.Question.Type = qatypes.InputSolutionFormType
			}
			flagNoID := p.Question.ID == ""
			if flagNoID {
				p.Question.ID = parameterizertypes.ParamQuesIDPrefix + common.Delim + apiVersion + common.Delim + kind + common.Delim + key
			}
			filledDesc, err := fillCustomTemplate(p.Question.Desc, kind, apiVersion, metadataName, resultKV.Matches)
			if err != nil {
				return err
			}
			origQuesDesc := p.Question.Desc
			p.Question.Desc = filledDesc
			ques, err := qaengine.FetchAnswer(*p.Question)
			if err != nil {
				return fmt.Errorf("failed to ask a question to the user in order to parameterize a k8s resource: %+v\nError: %q", p.Question, err)
			}
			var ok bool
			paramValue, ok = ques.Answer.(string)
			if !ok {
				return fmt.Errorf("failed to ask a question to the user in order to parameterize a k8s resource: %+v\nError: the answer was not a string", p.Question)
			}
			if flagNoID {
				p.Question.ID = ""
			}
			p.Question.Desc = origQuesDesc
		}
		if len(parameters) == 1 {
			parameter := parameters[0]       // services.$(containerName).image
			subKeys := GetSubKeys(parameter) // [services, $(containerName), image]
			for i, subKey := range subKeys {
				if !strings.HasPrefix(subKey, "$(") || !strings.HasSuffix(subKey, ")") { // subkeys like services or image
					continue
				}
				// $(containerName)
				subKey = strings.TrimSuffix(strings.TrimPrefix(subKey, "$("), ")") // containerName
				matchedSubKey, ok := resultKV.Matches[subKey]                      // match against the parameters specified in the target: spec.containers.[containerName:name=nginx].image
				if ok {
					subKeys[i] = matchedSubKey
					continue
				}
				// match against the builtin parameters
				switch subKey {
				case "kind":
					subKeys[i] = kind
				case "apiVersion":
					subKeys[i] = apiVersion
				case "metadataName":
					subKeys[i] = metadataName
				default:
					return fmt.Errorf("failed to find the sub key $(%s) in the any of the keys that matched: %+v", subKey, resultKV)
				}
			}
			// [services, nginx, image]
			ocParamKey := strings.Join(subKeys, "_")                                     // services_nginx_image
			ocParamKey = invalidOCTemplateChars.ReplaceAllLiteralString(ocParamKey, "_") // openshift templates require parameters that match [a-zA-Z0-9_]*
			ocParamKey = strings.ToUpper(ocParamKey)                                     // SERVICES_NGINX_IMAGE
			ocTemplate := `${` + ocParamKey + `}`                                        // ${SERVICES_NGINX_IMAGE}
			if len(p.Parameters) > 0 {
				if len(p.Parameters) != 1 {
					return fmt.Errorf("the template only has a single parameter. Expected a single paramter definition. Actual length: %d Parameters: %+v", len(p.Parameters), p.Parameters)
				}
				param := p.Parameters[0]
				if param.Name != parameter {
					return fmt.Errorf("the name in the paramter definition doesn't match the name in the template. Parameters: %+v", param)
				}
				if param.OpenshiftTemplate != "" {
					ocTemplate = param.OpenshiftTemplate
				}
				if param.Default != "" {
					paramValue = param.Default
				}
			}
			flagNonString := false
			for _, env := range envs {
				origParamValue := paramValue
				if len(p.Parameters) > 0 {
					param := p.Parameters[0]
					for _, pV := range param.Values {
						if doesMatchEnv(pV, env, kind, apiVersion, metadataName, resultKV.Matches) {
							paramValue = pV.Value
							break
						}
					}
				}
				if _, ok := namedOCParams[env]; !ok {
					namedOCParams[env] = map[string]string{}
				}
				// set the key in the parameters.yaml
				if paramValueStr, ok := paramValue.(string); ok {
					namedOCParams[env][ocParamKey] = paramValueStr
				} else {
					flagNonString = true
					paramValueStr, err := cast.ToStringE(paramValue)
					if err != nil {
						return fmt.Errorf("openshift templates require parameter value to be string (or convertible to string)")
					}
					namedOCParams[env][ocParamKey] = paramValueStr
				}
				paramValue = origParamValue
			}
			if flagNonString {
				ocTemplate = `${{` + ocParamKey + `}}`
			}
			if err := set(key, ocTemplate, k); err != nil {
				return fmt.Errorf("failed to set the key %s to the value %s in the k8s resource: %+v\nError: %q", key, ocTemplate, k, err)
			}
			return nil
		}
		// multiple parameters only make sense when the original value is a string
		originalValueStr, ok := resultKV.Value.(string)
		if !ok {
			return fmt.Errorf(
				`the template %s contains multiple parameters.
This only makes sense when the original value is a string.
Actual value is %+v of type %T`,
				templ, resultKV.Value, resultKV.Value,
			)
		}
		defaultStr := originalValueStr
		if paramValue != nil {
			defaultStr, ok = paramValue.(string)
			if !ok {
				return fmt.Errorf("the default parameter value is not a string. Actual value %+v is of type %T", paramValue, paramValue)
			}
		}
		originalValues, paramsAndStrings, err := parseTemplate(templ, defaultStr, p.Regex)
		if err != nil {
			return fmt.Errorf("failed to parse the multi parameter template: %s\nError: %q", templ, err)
		}
		ocTemplates := []string{}
		paramKeys := []string{}
		for _, parameter := range parameters {
			subKeys := GetSubKeys(parameter)
			for i, subKey := range subKeys {
				if !strings.HasPrefix(subKey, "$(") || !strings.HasSuffix(subKey, ")") {
					continue
				}
				subKey = strings.TrimSuffix(strings.TrimPrefix(subKey, "$("), ")")
				matchedSubKey, ok := resultKV.Matches[subKey]
				if ok {
					subKeys[i] = matchedSubKey
					continue
				}
				switch subKey {
				case "kind":
					subKeys[i] = kind
				case "apiVersion":
					subKeys[i] = apiVersion
				case "metadataName":
					subKeys[i] = metadataName
				default:
					return fmt.Errorf("failed to find the sub key $(%s) in the any of the keys that matched: %+v", subKey, resultKV)
				}
			}
			ocParamKey := strings.Join(subKeys, "_")                                     // services_nginx_image
			ocParamKey = invalidOCTemplateChars.ReplaceAllLiteralString(ocParamKey, "_") // openshift templates require parameters that match [a-zA-Z0-9_]*
			ocParamKey = strings.ToUpper(ocParamKey)                                     // SERVICES_NGINX_IMAGE
			ocTemplate := `${` + ocParamKey + `}`                                        // ${SERVICES_NGINX_IMAGE}
			paramKeys = append(paramKeys, ocParamKey)
			for _, param := range p.Parameters {
				if param.Name != parameter {
					continue
				}
				if param.OpenshiftTemplate != "" {
					ocTemplate = param.OpenshiftTemplate
				}
				break
			}
			ocTemplates = append(ocTemplates, ocTemplate)
		}
		fullOCTemplate := ""
		ocTemplateIdx := 0
		for _, pOrS := range paramsAndStrings {
			if !pOrS.IsParam {
				fullOCTemplate += pOrS.Data
				continue
			}
			currOCTemplate := ocTemplates[ocTemplateIdx]
			for _, param := range p.Parameters {
				if "${"+param.Name+"}" != pOrS.Data {
					continue
				}
				if param.HelmTemplate != "" {
					currOCTemplate = param.HelmTemplate
				}
				break
			}
			fullOCTemplate += currOCTemplate
			ocTemplateIdx++
		}
		if err := set(key, fullOCTemplate, k); err != nil {
			return fmt.Errorf("failed to set the key %s to the value %s in the k8s resource: %+v\nError: %q", key, fullOCTemplate, k, err)
		}
		// set all the keys in the values.yaml
		for i, parameter := range parameters {
			ocParamKey := paramKeys[i]
			paramValue := originalValues[i]
			for _, env := range envs {
				origParamValue := paramValue
				for _, param := range p.Parameters {
					if param.Name != parameter {
						continue
					}
					if param.Default != "" {
						paramValue = param.Default
					}
					for _, pV := range param.Values {
						if doesMatchEnv(pV, env, kind, apiVersion, metadataName, resultKV.Matches) {
							paramValue = pV.Value
							break
						}
					}
					break
				}
				// set the key in the values.yaml
				if _, ok := namedOCParams[env]; !ok {
					namedOCParams[env] = map[string]string{}
				}
				namedOCParams[env][ocParamKey] = paramValue
				paramValue = origParamValue
			}
		}
	}
	return nil
}
