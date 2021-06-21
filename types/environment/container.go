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

package environment

// Container stores container based execution information
type Container struct {
	Image          string         `yaml:"image"`
	WorkingDir     string         `yaml:"workingDir,omitempty"`
	ContainerBuild ContainerBuild `yaml:"build"`
}

type ContainerBuild struct {
	Dockerfile string `yaml:"dockerfile"` // Default : Look for Dockerfile in the same folder
	Context    string `yaml:"context"`    // Default : Same folder as the yaml
}
