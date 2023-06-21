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

package environment

// Container stores container based execution information
type Container struct {
	// ID is the id of the running container.
	ID string `yaml:"-"`
	// Image is the image name and tag to use when starting the container.
	Image string `yaml:"image"`
	// KeepAliveCommand is the command to run when starting the container.
	// It should keep the container alive so that the detect and transform commands can be executed later.
	// By default we will use the entrypoint of the container.
	KeepAliveCommand []string `yaml:"keepAliveCommand,omitempty"`
	// WorkingDir is the directory where the command will be run.
	WorkingDir string `yaml:"workingDir,omitempty"`
	// ImageBuild contains the instructions to build the image used by this container.
	ImageBuild ImageBuild `yaml:"build"`
}

// ImageBuild stores container build information
type ImageBuild struct {
	ForceRebuild bool   `yaml:"forceRebuild"` // Force rebuild the image even if it exists
	Dockerfile   string `yaml:"dockerfile"`   // Default : Look for Dockerfile in the same folder
	Context      string `yaml:"context"`      // Default : Same folder as the yaml
}
