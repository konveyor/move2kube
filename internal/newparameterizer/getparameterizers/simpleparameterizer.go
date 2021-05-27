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

// -----------
// File Format
// -----------
/*
apiVersion: move2kube.konveyor.io/v1alpha1
kind: Parameterizer
metadata:
  name: t1
spec:
  parameterizers:
  	- target: 'metadata.annotations."openshift.io/node-selector"'
      template: '${metadata.annotations.nodeselector}'
	  filter:
		- kind: 'Deployment'
	  	  apiVersion: '.*v1.*'
		  name: 'd1.*'
		- kind: 'Service'
		  apiVersion: 'v1'
    - target: 'spec.replicas'
      template: '${common.replicas}'
      parameters:
	  	- name: 'common.replicas'
		  value: 10
      filter:
        - kind: 'Deployment'
		  apiVersion: '.*v1.*'
	- target: 'spec.template.spec.containers.[0].image'
	  template: '${imageregistryurl}/{imageregistrynamespace}/${imagename}:${imagetag}'
	  parameters:
	  	- name: imageregistryurl
		  default: quay.io
		- name: imageregistrynamespace
 		  default: konveyor
		- name: imagename
		  default: default_image
		  openshiftTemplateParameter: 'foobar_${resourceName}'
		  helmTemplateParameter: 'services.${kind}.${apiVersion}.${resourceName}.imagename'
		  values:
			- env: [dev, staging, prod]
	 		  apiVersion: apps/v1
			  resourceName: nginx
			  value: nginx_image
			- env: [prod]
			  apiVersion: apps/v1
			  resourceName: javaspringapp
			  value: openjdk8
			- env: [dev]
			  apiVersion: extensions/v1beta1
			  resourceName: javaspringapp
			  value: openjdk-dev8
		- name: imagetag
		  default: latest
	  envs: [dev, staging, prod]
	  filter:
		- kind: 'Deployment'
		  apiVersion: '.*v1.*'
		  name: 'd1.*'
*/

// SimpleParameterizerFile is the file format for the parameterizers
type SimpleParameterizerFile struct {
	metav1.TypeMeta   `yaml:",inline" json:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	Spec              SimpleParameterizerFileSpec `yaml:"spec,omitempty" json:"spec,omitempty"`
}

// SimpleParameterizerFileSpec is the spec inside the file format for the parameterizers
type SimpleParameterizerFileSpec struct {
	Parameterizers []SimpleParameterizerT `yaml:"parameterizers,omitempty" json:"parameterizers,omitempty"`
}

// SimpleParameterizer implements the ParameterizerT interface
type SimpleParameterizerT struct {
	Target           string                      `yaml:"target" json:"target"`
	HelmTemplate     string                      `yaml:"template,omitempty" json:"template,omitempty"`
	Default          interface{}                 `yaml:"default,omitempty" json:"default,omitempty"`
	KindsAPIVersions startypes.KindsAPIVersionsT `yaml:"filter,omitempty" json:"filter,omitempty"`
}

// Parameterize parameterizes the k8s resource
func (st *SimpleParameterizerT) Parameterize(k8sResource startypes.K8sResourceT, values map[string]interface{}) (startypes.K8sResourceT, error) {
	log.Trace("start SimpleParameterizerT.Parameterize")
	defer log.Trace("end SimpleParameterizerT.Parameterize")
	originalValue, ok := newparamcommon.Get(st.Target, k8sResource)
	if !ok {
		return nil, fmt.Errorf("the key %s does not exist on the k8s resource: %+v", st.Target, k8sResource)
	}
	templ := st.HelmTemplate
	newKey := st.Target // TODO: somehow get it from st.HelmTemplate
	if templ == "" {
		kind, apiVersion, name, err := starcommon.GetInfoFromK8sResource(k8sResource)
		if err != nil {
			return k8sResource, fmt.Errorf("failed to get the kind, apiVersion, and name from the k8s resource: %+v\nError: %q", k8sResource, err)
		}
		subKeys := newparamcommon.GetSubKeys(st.Target)
		for i, subKey := range subKeys {
			subKeys[i] = `"` + subKey + `"`
		}
		templ = fmt.Sprintf(`{{ index .Values "%s" "%s" "%s" %s }}`, kind, apiVersion, name, strings.Join(subKeys, " "))
		newKey = fmt.Sprintf(`"%s"."%s"."%s".%s`, kind, apiVersion, name, strings.Join(subKeys, "."))
	}
	if err := newparamcommon.Set(st.Target, templ, k8sResource); err != nil {
		return k8sResource, fmt.Errorf("failed to parameterize the k8s resource: %+v\nusing the parameterizer %+v\nError: %q", k8sResource, st, err)
	}
	var finalValue interface{} = st.Default
	if finalValue == nil {
		finalValue = originalValue
	}
	if err := newparamcommon.SetCreatingNew(newKey, finalValue, values); err != nil {
		return k8sResource, fmt.Errorf("failed to set the key %s to the value %+v in the values.yaml. Error: %q", st.Target, finalValue, err)
	}
	return k8sResource, nil
}

// Filter returns true if this parameterizer can be applied to the given k8s resource
func (st *SimpleParameterizerT) Filter(k8sResource startypes.K8sResourceT) (bool, error) {
	log.Trace("start SimpleParameterizerT.Filter")
	defer log.Trace("end SimpleParameterizerT.Filter")
	k8sResourceKind, k8sResourceAPIVersion, _, err := starcommon.GetInfoFromK8sResource(k8sResource)
	if err != nil {
		return false, err
	}
	if len(st.KindsAPIVersions) == 0 {
		// empty map matches all kinds and apiVersions
		return true, nil
	}
	for kind, apiVersions := range st.KindsAPIVersions {
		// empty kind matches all kinds
		if kind != "" {
			re, err := regexp.Compile("^" + kind + "$")
			if err != nil {
				return false, err
			}
			if !re.MatchString(k8sResourceKind) {
				continue
			}
		}
		if len(apiVersions) == 0 {
			// empty array matches all apiVersions
			return true, nil
		}
		for _, apiVersion := range apiVersions {
			re, err := regexp.Compile("^" + apiVersion + "$")
			if err != nil {
				return false, err
			}
			if re.MatchString(k8sResourceAPIVersion) {
				return true, nil
			}
		}
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
