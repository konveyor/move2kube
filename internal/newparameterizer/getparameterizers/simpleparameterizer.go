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
	"html/template"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/internal/common"
	newparamcommon "github.com/konveyor/move2kube/internal/newparameterizer/common"
	"github.com/konveyor/move2kube/internal/newparameterizer/types"
	starcommon "github.com/konveyor/move2kube/internal/starlark/common"
	startypes "github.com/konveyor/move2kube/internal/starlark/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
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
	// fill the template with a single key from the values.yaml,
	// the key is {{ index .Values "Deployment" "apps/v1" "nginx" "spec" "replicas" }}
	// the default value is the original value for replicas specified in the Deployment yaml
	- target: 'spec.replicas'
      filter:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

	// fill the template with a single key from the values.yaml
	// specify the key to use for the values.yaml
	// the key is {{ index .Values "common" "replicas" }}
	// the default value is the original value for replicas specified in the Deployment yaml
    - target: 'spec.replicas'
      template: '${common.replicas}'
      filter:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

	// fill the template with a single key from the values.yaml
	// specify the default value to put in the values.yaml
	// the key is {{ index .Values "Deployment" "apps/v1" "nginx" "spec" "replicas" }}
	// the default value is 2
	- target: 'spec.replicas'
	  default: 2
      filter:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

	// fill the template with a single key from the values.yaml
	// specify the key to use for the values.yaml
	// specify the default value to put in the values.yaml
	// the key is {{ index .Values "common" "replicas" }}
	// the default value is 2
    - target: 'spec.replicas'
      template: '${common.replicas}'
	  default: 2
      filter:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'

	// fill the template with multiple keys from the values.yaml
	// the keys are:
	// ${imageregistry.url}       becomes {{ index .Values "imageregistry" "url" }}
	// ${imageregistry.namespace} becomes {{ index .Values "imageregistry" "namespace" }}
	// ${image.name}              becomes {{ index .Values "image" "name" }}
	// ${image.tag}               becomes {{ index .Values "image" "tag" }}
	// the default values are taken from the original value specified in the Deployment yaml
	- target: 'spec.template.spec.containers.[0].image'
	  template: '${imageregistry.url}/${imageregistry.namespace}/${image.name}:${image.tag}'
	  filter:
		- kind: 'Deployment'
		  apiVersion: '.*v1.*'
		  name: 'd1.*'

	// fill the template with multiple keys from the values.yaml
	// specify the default values to put in the values.yaml
	// the keys are:
	// ${imageregistry.url}       becomes {{ index .Values "imageregistry" "url" }}
	// ${imageregistry.namespace} becomes {{ index .Values "imageregistry" "namespace" }}
	// ${image.name}              becomes {{ index .Values "image" "name" }}
	// ${image.tag}               becomes {{ index .Values "image" "tag" }}
	// the default values are:
	// ${imageregistry.url}       becomes us.icr.io
	// ${imageregistry.namespace} becomes move2kube
	// ${image.name}              becomes myimage
	// ${image.tag}               becomes myimagetag
	- target: 'spec.template.spec.containers.[0].image'
	  template: '${imageregistry.url}/${imageregistry.namespace}/${image.name}:${image.tag}'
	  default: 'us.icr.io/move2kube/myimage:myimagetag'
	  filter:
		- kind: 'Deployment'
		  apiVersion: '.*v1.*'
		  name: 'd1.*'

	// fill the template with multiple keys from the values.yaml
	// specify the default values to put in the values.yaml
	// specify environment specific default values to put in the values.yaml
	// the keys are:
	// ${imageregistry.url}       becomes {{ index .Values "imageregistry" "url" }}
	// ${imageregistry.namespace} becomes {{ index .Values "imageregistry" "namespace" }}
	// ${image.name}              becomes {{ index .Values "services" "nginx" "imagename" }}
	// ${image.tag}               becomes {{ index .Values "image" "tag" }}
	// the default values are:
	// ${imageregistry.url}       becomes us.icr.io
	// ${imageregistry.namespace} becomes move2kube
	// ${image.name}              becomes different values depending on the environment
	// ${image.tag}               becomes myimagetag
	- target: 'spec.template.spec.containers.[0].image'
	  template: '${imageregistry.url}/${imageregistry.namespace}/${image.name}:${image.tag}'
	  default: 'us.icr.io/move2kube/myimage:myimagetag'
	  filter:
		- kind: 'Deployment'
		  apiVersion: '.*v1.*'
		  name: 'd1.*'
	  envs: [dev, staging, prod]
	  parameters:
		- name: image.name
		  default: default_image
		  openshiftTemplateParameter: 'services_${metadataName}_imagename'
		  helmTemplateParameter: 'services.${metadataName}.imagename'
		  values:
			- env: [dev, staging, prod]
	 		  apiVersion: apps/v1
			  metadataName: nginx
			  value: nginx_image
			- env: [prod]
			  apiVersion: apps/v1
			  metadataName: javaspringapp
			  value: openjdk8
			- env: [dev]
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
	Parameterizers []SimpleParameterizerT `yaml:"parameterizers,omitempty" json:"parameterizers,omitempty"`
}

// SimpleParameterizer implements the ParameterizerT interface
type SimpleParameterizerT struct {
	Target     string       `yaml:"target" json:"target"`
	Template   string       `yaml:"template,omitempty" json:"template,omitempty"`
	Default    interface{}  `yaml:"default,omitempty" json:"default,omitempty"`
	Parameters []ParameterT `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Filters    []FilterT    `yaml:"filters,omitempty" json:"filters,omitempty"`
}

// ParameterT is used to specify the defaults for the parameters in the template
type ParameterT struct {
	Name              string            `yaml:"name" json:"name"`
	Default           interface{}       `yaml:"default,omitempty" json:"default,omitempty"`
	HelmTemplate      string            `yaml:"helmTemplate,omitempty" json:"helmTemplate,omitempty"`
	OpenshiftTemplate string            `yaml:"openshiftTemplate,omitempty" json:"openshiftTemplate,omitempty"`
	Values            []ParameterValueT `yaml:"values,omitempty" json:"values,omitempty"`
}

type ParameterValueT struct {
	Env          []string `yaml:"env,omitempty" json:"env,omitempty"`
	Kind         string   `yaml:"kind,omitempty" json:"kind,omitempty"`
	ApiVersion   string   `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	MetadataName string   `yaml:"metadataName,omitempty" json:"metadataName,omitempty"`
	Value        string   `yaml:"value" json:"value"`
}

type FilterT struct {
	Kind       string `yaml:"kind,omitempty" json:"kind,omitempty"`
	ApiVersion string `yaml:"apiVersion,omitempty" json:"apiVersion,omitempty"`
	Name       string `yaml:"name,omitempty" json:"name,omitempty"`
}

var (
	stringInterpRegex = regexp.MustCompile(`\${([^}]+)}`)
)

func splitOnIdxs(s string, idxs []int) []string {
	prevIdx := 0
	ss := []string{}
	for _, idx := range idxs {
		currs := s[prevIdx:idx]
		prevIdx = idx
		if len(currs) == 0 {
			continue
		}
		ss = append(ss, currs)
	}
	return ss
}

func parseTemplate(s string, orig interface{}) ([]string, []string, map[string]string, error) {
	re := regexp.MustCompile(`\${([^}]+)}`)
	matches := re.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return nil, nil, nil, fmt.Errorf("no interpolation variables (example: ${foo}) found in the template %s", s)
	}
	idxs := []int{}
	vars := []string{}
	for _, match := range matches {
		// assume len(match) == 4 (2 idxs for the original string, 2 for the capturing group). Not sure when this can be false.
		idxs = append(idxs, match[:2]...)
		vars = append(vars, s[match[2]:match[3]])
	}
	ss := splitOnIdxs(s, idxs)
	newre, err := regexp.Compile("^" + re.ReplaceAllString(s, `(.*)`) + "$")
	if err != nil {
		return ss, vars, nil, fmt.Errorf("failed to make a new regex using the template: %s\nError: %q", s, err)
	}
	newmatches := newre.FindAllStringSubmatch(orig, -1)
	if len(newmatches) == 0 {
		return ss, vars, nil, fmt.Errorf("no matches found with the new regex %+v", newre)
	}
	if len(newmatches[0]) != len(vars)+1 {
		return ss, vars, nil, fmt.Errorf("expected to find a single match per interpolation variable. Actual %+v", newmatches)
	}
	matchedvars := map[string]string{}
	for i, newmatch := range newmatches[0][1:] {
		matchedvars[vars[i]] = newmatch
	}
	return ss, vars, matchedvars, nil
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

	templ := st.Template
	if st.Template == "" {
		kind, apiVersion, name, err := starcommon.GetInfoFromK8sResource(k8sResource)
		if err != nil {
			return k8sResource, fmt.Errorf("failed to get the kind, apiVersion, and name from the k8s resource: %+v\nError: %q", k8sResource, err)
		}
		subKeys := newparamcommon.GetSubKeys(st.Target)
		for i, subKey := range subKeys {
			subKeys[i] = `"` + subKey + `"`
		}
		templ = `${` + newKey + `}`
	}

	_, _, matchedVars, err := parseTemplate(st.Template, originalValue)
	if err != nil {
		return k8sResource, fmt.Errorf("failed to get the interpolation variables from the template: %s\nError: %q", st.Template, err)
	}


		// templ := fmt.Sprintf(`{{ index .Values "%s" "%s" "%s" %s }}`, kind, apiVersion, name, strings.Join(subKeys, " "))
		// newKey := fmt.Sprintf(`"%s"."%s"."%s".%s`, kind, apiVersion, name, strings.Join(subKeys, "."))

		// if err := newparamcommon.Set(st.Target, templ, k8sResource); err != nil {
		// 	return k8sResource, fmt.Errorf("failed to parameterize the k8s resource: %+v\nusing the parameterizer %+v\nError: %q", k8sResource, st, err)
		// }
		// var finalValue interface{} = originalValue
		// }
		// if err := newparamcommon.SetCreatingNew(newKey, finalValue, values); err != nil {
		// 	return k8sResource, fmt.Errorf("failed to set the key %s to the value %+v in the values.yaml. Error: %q", st.Target, finalValue, err)
		// }
		// return k8sResource, nil
	} else {
		originalValueStr := ""
		switch t1 := originalValue.(type) {
		case string, int, float32, float64:
			var err error
			originalValueStr, err = cast.ToStringE(t1)
			if err != nil {
				return k8sResource, fmt.Errorf("failed to convert the orignal value to string: %+v\nError: %q", t1, err)
			}
		default:
			return k8sResource, fmt.Errorf("expected the key %s to be a string or int or float. Actual value is %+v of type %T", t1, t1)
		}
		subKeys := []string{}
		for matchedVar := range matchedVars {
			subKeys = append(subKeys, matchedVar)
		}
		newKey = strings.Join(subKeys, ".")
	}
	if err := newparamcommon.Set(st.Target, templ, k8sResource); err != nil {
		return k8sResource, fmt.Errorf("failed to parameterize the k8s resource: %+v\nusing the parameterizer %+v\nError: %q", k8sResource, st, err)
	}
	var finalValue interface{} = originalValue
	if err := newparamcommon.SetCreatingNew(newKey, finalValue, values); err != nil {
		return k8sResource, fmt.Errorf("failed to set the key %s to the value %+v in the values.yaml. Error: %q", st.Target, finalValue, err)
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
	if err := common.ReadMove2KubeYaml(parameterizerPath, &param); err != nil {
		log.Errorf("failed to read the paarameterizer from the file at path %s . Error: %q", parameterizerPath, err)
		return nil, err
	}
	ps := []types.ParameterizerT{}
	for i := range param.Spec.Parameterizers {
		ps = append(ps, &param.Spec.Parameterizers[i])
	}
	return ps, nil
}
