/*
Copyright IBM Corporation 2021

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

package translator

import (
	"github.com/konveyor/move2kube/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/*
1. If handler, use
*/

/*
apiVersion: move2kube.konveyor.io/v1alpha1
kind: Generator
metadata:
  name: m2k-java-containerizer
spec:
  class: Executable
  mode: Container // Container, CustomResource, Service, Custom
  detect: "detect.sh" // Analyse is called if detect is not present
  analyse: "analyse.sh" // Should return a json
  localEnv: "M2KCONTAINERIZER_ENV_AVAILABLE"
  container:
    image: "m2k-java-containerizer:latest"
    build:
      context: "."
      dockerfile: Dockerfile
*/

const TranslatorKind = "Translator"

type Mode string

const (
	ModeContainer Mode = "Container"
	ModeCR        Mode = "CustomResource"
	ModeService   Mode = "Service" // Possibly Terraform
	ModeCustom    Mode = "Custom"
)

// Translator defines definition of cf runtime instance apps file
type Translator struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty"`
	Spec              TranslatorSpec `yaml:"spec,omitempty"`
}

// TranslatorSpec stores the data
type TranslatorSpec struct {
	FilePath string `yaml:"-"`
	Mode     Mode   `yaml:"mode"`
	Class    string `yaml:"class"`
	Config   interface{}
}

// NewDockerfileContainerizer creates a new instance of DockerfileContainerizer
func NewTranslator() Translator {
	return Translator{
		TypeMeta: metav1.TypeMeta{
			Kind:       TranslatorKind,
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}
