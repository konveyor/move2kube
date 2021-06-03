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

package getparameterizers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	newparamcommon "github.com/konveyor/move2kube/internal/newparameterizer/common"
	"github.com/konveyor/move2kube/internal/newparameterizer/types"
	starcommon "github.com/konveyor/move2kube/internal/starlark/common"
	startypes "github.com/konveyor/move2kube/internal/starlark/types"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// -------------------------
// File Format for Packaging
// -------------------------
/*
apiVersion: move2kube.konveyor.io/v1alpha1
kind: Packaging
metadata:
  name: t1
spec:
  paths:
	- src: custom/yamls
	  out: custom/
	  helm: my/dest/helm
	  kustomize: another/kustomize
	- src: yamls
	  helm: my/dest/helm
	  kustomize: another/kustomize
  parameterizers:
    names:
	  - t1
	objects:
		- target: 'metadata.annotations."openshift.io/node-selector"'
		  template: '${metadata.annotations.nodeselector}'
		  filter:
			- kind: 'Deployment'
			  apiVersion: '.*v1.*'
			  name: 'd1.*'
			- kind: 'Service'
			  apiVersion: 'v1'
*/

// ------------------------------
// File Format for Parameterizers
// ------------------------------
/*
apiVersion: move2kube.konveyor.io/v1alpha1
kind: Parameterizers
spec:
  parameterizers:
  	# Case 1:
	# fill the template with a single key from the values.yaml,
	# the key is {{ index .Values "Deployment" "apps/v1" "nginx" "spec" "replicas" }}
	# the default value is the original value for replicas specified in the Deployment yaml
	- target: 'spec.replicas'
      filters:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

  	# Case 2:
	# fill the template with a single key from the values.yaml
	# specify the key to use for the values.yaml
	# the key is {{ index .Values "common" "replicas" }}
	# the default value is the original value for replicas specified in the Deployment yaml
    - target: 'spec.replicas'
      template: '${common.replicas}'
      filters:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

  	# Case 3:
	# fill the template with a single key from the values.yaml
	# specify the default value to put in the values.yaml
	# the key is {{ index .Values "Deployment" "apps/v1" "nginx" "spec" "replicas" }}
	# the default value is 2
	- target: 'spec.replicas'
	  default: 2
      filters:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

  	# Case 4:
	# fill the template with a single key from the values.yaml
	# specify the key to use for the values.yaml
	# specify the default value to put in the values.yaml
	# the key is {{ index .Values "common" "replicas" }}
	# the default value is 2
    - target: 'spec.replicas'
      template: '${common.replicas}'
	  default: 2
      filters:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

	# IMPORTANT !!!!!!
	# The following only makes sense when the field we are parameterizing is a string
	# IMPORTANT !!!!!!

  	# Case 5:
	# fill the template with multiple keys from the values.yaml
	# the keys are:
	# ${imageregistry.url}       becomes {{ index .Values "imageregistry" "url" }}
	# ${imageregistry.namespace} becomes {{ index .Values "imageregistry" "namespace" }}
	# ${image.name}              becomes {{ index .Values "image" "name" }}
	# ${image.tag}               becomes {{ index .Values "image" "tag" }}
	# the default values are taken from the original value specified in the Deployment yaml
	- target: 'spec.template.spec.containers.[0].image'
	  template: '${imageregistry.url}/${imageregistry.namespace}/${image.name}:${image.tag}'
	  filters:
		- kind: 'Deployment'
		  apiVersion: '.*v1.*'
		  name: 'd1.*'

  	# Case 6:
	# fill the template with multiple keys from the values.yaml
	# specify the default values to put in the values.yaml
	# the keys are:
	# ${imageregistry.url}       becomes {{ index .Values "imageregistry" "url" }}
	# ${imageregistry.namespace} becomes {{ index .Values "imageregistry" "namespace" }}
	# ${image.name}              becomes {{ index .Values "image" "name" }}
	# ${image.tag}               becomes {{ index .Values "image" "tag" }}
	# the default values are:
	# ${imageregistry.url}       becomes us.icr.io
	# ${imageregistry.namespace} becomes move2kube
	# ${image.name}              becomes myimage
	# ${image.tag}               becomes myimagetag
	- target: 'spec.template.spec.containers.[0].image'
	  template: '${imageregistry.url}/${imageregistry.namespace}/${image.name}:${image.tag}'
	  default: 'us.icr.io/move2kube/myimage:myimagetag'
	  filters:
		- kind: 'Deployment'
		  apiVersion: '.*v1.*'
		  name: 'd1.*'

  	# Case 7:
	# fill the template with multiple keys from the values.yaml
	# specify the default values to put in the values.yaml
	# specify environment specific default values to put in the values.yaml
	# the keys are:
	# ${imageregistry.url}       becomes {{ index .Values "imageregistry" "url" }}
	# ${imageregistry.namespace} becomes {{ index .Values "imageregistry" "namespace" }}
	# ${image.name}              becomes {{ index .Values "services" "nginx" "imagename" }}
	# ${image.tag}               becomes {{ index .Values "image" "tag" }}
	# the default values are:
	# ${imageregistry.url}       becomes us.icr.io
	# ${imageregistry.namespace} becomes move2kube
	# ${image.name}              becomes different values depending on the environment
	# ${image.tag}               becomes myimagetag
	- target: 'spec.template.spec.containers.[0].image'
	  template: '${imageregistry.url}/${imageregistry.namespace}/${image.name}:${image.tag}'
	  default: 'us.icr.io/move2kube/myimage:myimagetag'
	  filters:
		- kind: 'Deployment'
		  apiVersion: '.*v1.*'
		  name: 'd1.*'
	  envs: [dev, staging, prod]
	  parameters:
		- name: image.name
		  default: default_image
		  openshiftTemplateParameter: 'SERVICES_${metadataName}_IMAGENAME'
		  helmTemplateParameter: 'services.${metadataName}.imagename'
		  values:
			- envs: [dev, staging, prod]
	 		  apiVersion: apps/v1
			  metadataName: nginx
			  value: nginx_image
			- envs: [prod]
			  apiVersion: apps/v1
			  metadataName: javaspringapp
			  value: openjdk8
			- envs: [dev]
			  apiVersion: extensions/v1beta1
			  metadataName: javaspringapp
			  value: openjdk-dev8
*/

// SimpleParameterizerFile is the file format for the parameterizers
type SimpleParameterizerFile struct {
	metav1.TypeMeta   `yaml:",inline" json:",inline"`
	metav1.ObjectMeta `yaml:"metadata" json:"metadata"`
	Spec              SimpleParameterizerFileSpec `yaml:"spec" json:"spec"`
}

// SimpleParameterizerFileSpec is the spec inside the file format for the parameterizers
type SimpleParameterizerFileSpec struct {
	Parameterizers []SimpleParameterizerT `yaml:"parameterizers" json:"parameterizers"`
}

// SimpleParameterizerT implements the ParameterizerT interface
type SimpleParameterizerT struct {
	Target     string       `yaml:"target" json:"target"`
	Template   string       `yaml:"template,omitempty" json:"template,omitempty"`
	Default    interface{}  `yaml:"default,omitempty" json:"default,omitempty"`
	Filters    []FilterT    `yaml:"filters,omitempty" json:"filters,omitempty"`
	Envs       []string     `yaml:"envs,omitempty" json:"envs,omitempty"`
	Parameters []ParameterT `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

type FilterT struct {
	Kind       string `yaml:"kind,omitempty" json:"kind,omitempty"`
	ApiVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
}

// ParameterT is used to specify the environment specific defaults for the keys in the template
type ParameterT struct {
	Name              string            `yaml:"name" json:"name"`
	Default           string            `yaml:"default,omitempty" json:"default,omitempty"`
	HelmTemplate      string            `yaml:"helmTemplate,omitempty" json:"helmTemplate,omitempty"`
	OpenshiftTemplate string            `yaml:"openshiftTemplate,omitempty" json:"openshiftTemplate,omitempty"`
	Values            []ParameterValueT `yaml:"values,omitempty" json:"values,omitempty"`
}

type ParameterValueT struct {
	Envs         []string `yaml:"envs,omitempty" json:"envs,omitempty"`
	Kind         string   `yaml:"kind,omitempty" json:"kind,omitempty"`
	ApiVersion   string   `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	MetadataName string   `yaml:"metadataName,omitempty" json:"metadataName,omitempty"`
	Value        string   `yaml:"value" json:"value"`
}

type paramOrStringT struct {
	isParam bool
	data    string
}

var (
	stringInterpRegex = regexp.MustCompile(`\${([^}]+)}`)
)

func splitOnIdxs(s string, idxs []int) []paramOrStringT {
	ss := []paramOrStringT{}
	prevIdx := 0
	isParam := false
	for _, idx := range idxs {
		currs := paramOrStringT{isParam: isParam, data: s[prevIdx:idx]}
		prevIdx = idx
		isParam = !isParam
		ss = append(ss, currs)
	}
	return ss
}

func parseTemplate(templ, orig string) ([]string, []paramOrStringT, error) {
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
	newre, err := regexp.Compile("^" + stringInterpRegex.ReplaceAllString(templ, `(.+)`) + "$")
	if err != nil {
		return nil, paramsAndStrings, fmt.Errorf("failed to make a new regex using the template: %s\nError: %q", templ, err)
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

// Parameterize parameterizes the k8s resource
func (st *SimpleParameterizerT) Parameterize(k8sResource startypes.K8sResourceT, values map[string]interface{}) (startypes.K8sResourceT, error) {
	log.Trace("start SimpleParameterizerT.Parameterize")
	defer log.Trace("end SimpleParameterizerT.Parameterize")
	if len(st.Target) == 0 {
		return k8sResource, fmt.Errorf("the target is empty")
	}
	originalValue, ok := newparamcommon.Get(st.Target, k8sResource)
	if !ok {
		return k8sResource, fmt.Errorf("the key %s does not exist on the k8s resource: %+v", st.Target, k8sResource)
	}
	kind, apiVersion, metadataName, err := starcommon.GetInfoFromK8sResource(k8sResource)
	if err != nil {
		return k8sResource, fmt.Errorf("failed to get the kind, apiVersion, and name from the k8s resource: %+v\nError: %q", k8sResource, err)
	}

	templ := st.Template
	if templ == "" {
		templ = fmt.Sprintf(`${"%s"."%s"."%s".%s}`, kind, apiVersion, metadataName, st.Target)
	}
	parameters, err := getParameters(templ)
	if err != nil {
		return k8sResource, fmt.Errorf("failed to get the parameters from the template: %s\nError: %q", templ, err)
	}
	if len(parameters) == 1 {
		parameter := parameters[0]
		subkeys := newparamcommon.GetSubKeys(parameter)
		for i, subkey := range subkeys {
			switch subkey {
			case "$kind":
				subkeys[i] = `"` + kind + `"`
			case "$apiVersion":
				subkeys[i] = `"` + apiVersion + `"`
			case "$metadataName":
				subkeys[i] = `"` + metadataName + `"`
			default:
				subkeys[i] = `"` + subkey + `"`
			}
		}
		helmTemplate := fmt.Sprintf(`{{ index .Values %s }}`, strings.Join(subkeys, " "))
		for _, stParameter := range st.Parameters {
			if stParameter.Name != parameter {
				continue
			}
			if stParameter.HelmTemplate != "" {
				helmTemplate = stParameter.HelmTemplate
			}
			break
		}
		if err := newparamcommon.Set(st.Target, helmTemplate, k8sResource); err != nil {
			return k8sResource, fmt.Errorf("failed to set the key %s to the value %s in the k8s resource: %+v\nError: %q", st.Target, helmTemplate, k8sResource, err)
		}
		// set the key in the values.yaml
		paramKey := strings.Join(subkeys, ".")
		paramValue := st.Default
		if st.Default == nil {
			paramValue = originalValue
		}
		if err := newparamcommon.SetCreatingNew(paramKey, paramValue, values); err != nil {
			return k8sResource, fmt.Errorf("failed to set the key %s to the value %+v in the values.yaml. Error: %q", parameter, paramValue, err)
		}
		return k8sResource, nil
	}

	// multiple parameters only make sense when the original value is a string
	originalValueStr, ok := originalValue.(string)
	if !ok {
		return k8sResource, fmt.Errorf(
			`the template %s contains multiple parameters.
This only makes sense when the original value is a string.
Actual value is %+v of type %T`,
			templ, originalValue, originalValue,
		)
	}
	defaultStr := originalValueStr
	if st.Default != nil {
		defaultStr, ok = st.Default.(string)
		if !ok {
			return k8sResource, fmt.Errorf(
				`the template %s contains multiple parameters.
Expected the default value to be a string.
Actual value is %+v of type %T`,
				templ, st.Default, st.Default,
			)
		}
	}
	originalValues, paramsAndStrings, err := parseTemplate(templ, defaultStr)
	if err != nil {
		return k8sResource, fmt.Errorf("failed to parse the multi parameter template: %s\nError: %q", templ, err)
	}
	helmTemplates := []string{}
	for _, parameter := range parameters {
		subkeys := newparamcommon.GetSubKeys(parameter)
		for i, subkey := range subkeys {
			switch subkey {
			case "$kind":
				subkeys[i] = `"` + kind + `"`
			case "$apiVersion":
				subkeys[i] = `"` + apiVersion + `"`
			case "$metadataName":
				subkeys[i] = `"` + metadataName + `"`
			default:
				subkeys[i] = `"` + subkey + `"`
			}
		}
		helmTemplate := fmt.Sprintf(`{{ index .Values %s }}`, strings.Join(subkeys, " "))
		for _, stParameter := range st.Parameters {
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
		if pOrS.isParam {
			fullHelmTemplate += helmTemplates[helmTemplateIdx]
			helmTemplateIdx++
			continue
		}
		fullHelmTemplate += pOrS.data
	}
	if err := newparamcommon.Set(st.Target, fullHelmTemplate, k8sResource); err != nil {
		return k8sResource, fmt.Errorf("failed to set the key %s to the value %s in the k8s resource: %+v\nError: %q", st.Target, fullHelmTemplate, k8sResource, err)
	}
	// set all the keys in the values.yaml
	for i, parameter := range parameters {
		subkeys := newparamcommon.GetSubKeys(parameter)
		for i, subkey := range subkeys {
			switch subkey {
			case "$kind":
				subkeys[i] = `"` + kind + `"`
			case "$apiVersion":
				subkeys[i] = `"` + apiVersion + `"`
			case "$metadataName":
				subkeys[i] = `"` + metadataName + `"`
			default:
				subkeys[i] = `"` + subkey + `"`
			}
		}
		paramKey := strings.Join(subkeys, ".")
		paramValue := originalValues[i]
		for _, stParameter := range st.Parameters {
			if stParameter.Name != parameter {
				continue
			}
			if stParameter.Default != "" {
				paramValue = stParameter.Default
			}
			break
		}
		if err := newparamcommon.SetCreatingNew(paramKey, paramValue, values); err != nil {
			return k8sResource, fmt.Errorf("failed to set the key %s to the value %+v in the values.yaml. Error: %q", parameter, paramValue, err)
		}
	}
	return k8sResource, nil
}

// Filter returns true if this parameterizer can be applied to the given k8s resource
func (st *SimpleParameterizerT) Filter(k8sResource startypes.K8sResourceT) (bool, error) {
	log.Trace("start SimpleParameterizerT.Filter")
	defer log.Trace("end SimpleParameterizerT.Filter")
	k8sResourceKind, k8sResourceAPIVersion, k8sResourceName, err := starcommon.GetInfoFromK8sResource(k8sResource)
	if err != nil {
		return false, err
	}
	if len(st.Filters) == 0 {
		// empty map matches all kinds, apiVersions and names
		return true, nil
	}
	for _, filter := range st.Filters {
		// empty kind matches all kinds
		if filter.Kind != "" {
			re, err := regexp.Compile("^" + filter.Kind + "$")
			if err != nil {
				return false, err
			}
			if !re.MatchString(k8sResourceKind) {
				continue
			}
		}
		// empty apiVersion matches all apiVersions
		if filter.ApiVersion != "" {
			re, err := regexp.Compile("^" + filter.ApiVersion + "$")
			if err != nil {
				return false, err
			}
			if !re.MatchString(k8sResourceAPIVersion) {
				continue
			}
		}
		// empty name matches all names
		if filter.Name != "" {
			re, err := regexp.Compile("^" + filter.Name + "$")
			if err != nil {
				return false, err
			}
			if !re.MatchString(k8sResourceName) {
				continue
			}
		}
		return true, nil
	}
	return false, nil
}

// GetParameterizersFromPath returns a list of parameterizers given a file path
func (*SimpleParameterizerT) GetParameterizersFromPath(parameterizerPath string) ([]types.ParameterizerT, error) {
	log.Trace("start SimpleParameterizerT.GetParameterizersFromPath")
	defer log.Trace("end SimpleParameterizerT.GetParameterizersFromPath")
	param := SimpleParameterizerFile{}
	if err := common.ReadMove2KubeYamlStrict(parameterizerPath, &param); err != nil {
		log.Errorf("failed to read the paarameterizer from the file at path %s . Error: %q", parameterizerPath, err)
		return nil, err
	}
	ps := []types.ParameterizerT{}
	for i := range param.Spec.Parameterizers {
		ps = append(ps, &param.Spec.Parameterizers[i])
	}
	return ps, nil
}
