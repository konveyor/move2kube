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

package sourcetypes

// CfInstanceApps for reading cf running instance metadata
type CfInstanceApps struct {
	CfResources []CfResource `json:"resources"`
}

// CfResource reads entity
type CfResource struct {
	CfAppEntity CfSourceApplication `json:"entity"`
}

// CfSourceApplication reads source application
type CfSourceApplication struct {
	Name              string            `json:"name"`
	Buildpack         string            `json:"buildpack"`
	DetectedBuildpack string            `json:"detected_buildpack"`
	Memory            int64             `json:"memory"`
	Instances         int               `json:"instances"`
	DockerImage       string            `json:"dockerimage"`
	Ports             []int32           `json:"ports"`
	Env               map[string]string `json:"environment_json,omitempty"`
}
