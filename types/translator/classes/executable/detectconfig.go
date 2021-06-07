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

package executable

type Config struct {
	DetectCMD  string    `yaml:"detectCMD"`
	AnalyseCMD string    `yaml:"analyseCMD"`
	LocalEnv   string    `yaml:"localEnv"` //When this environment variable is set, the environment is setup to run the program locally
	Container  Container `yaml:"container,omitempty"`
}

// Container stores container based execution information
type Container struct {
	Image          string         `yaml:"image"`
	ContainerBuild ContainerBuild `yaml:"build"`
}

// ContainerBuild stores container build information
type ContainerBuild struct {
	Context    string `yaml:"context"`    // Default : Same folder as the yaml
	Dockerfile string `yaml:"dockerfile"` // Default : Look for Dockerfile in the same folder
}
