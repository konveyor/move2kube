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

package generator

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
  mode: Container // Container, CustomResource, Service, Custom
  type: External // External, Internal
  detect: "detect.sh" // Analyse is called if detect is not present
  analyse: "analyse.sh" // Should return a json
  localEnv: "M2KCONTAINERIZER_ENV_AVAILABLE"
  container:
    image: "m2k-java-containerizer:latest"
    build:
      context: "."
      dockerfile: Dockerfile
*/

const GeneratorKind = "Generator"

type Mode string

const (
	ModeContainer Mode = "Container"
	ModeCR        Mode = "CustomResource"
	ModeService   Mode = "Service" // Possibly Terraform
	ModeCustom    Mode = "Custom"
)

// Generator defines definition of cf runtime instance apps file
type Generator struct {
	metav1.TypeMeta   `yaml:",inline"`
	metav1.ObjectMeta `yaml:"metadata,omitempty"`
	Spec              GeneratorSpec `yaml:"spec,omitempty"`
}

// GeneratorSpec stores the data
type GeneratorSpec struct {
	FilePath string `yaml:"-"`
	Mode     Mode   `yaml:"mode"`
	Class    string `yaml:"class"`
	Config   interface{}
}

// NewDockerfileContainerizer creates a new instance of DockerfileContainerizer
func NewDockerfileContainerizer() Generator {
	return Generator{
		TypeMeta: metav1.TypeMeta{
			Kind:       GeneratorKind,
			APIVersion: types.SchemeGroupVersion.String(),
		},
	}
}
