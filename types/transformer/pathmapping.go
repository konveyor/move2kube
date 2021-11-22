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

package transformer

// PathMappingType refers to the Path Mapping type
type PathMappingType = string

const (
	// DefaultPathMappingType allows normal copy with overwrite
	DefaultPathMappingType PathMappingType = "Default" // Normal Copy with overwrite
	// TemplatePathMappingType allows copy of source to destination and applying of template
	TemplatePathMappingType PathMappingType = "Template" // Source path when relative, is relative to yaml file location
	// SourcePathMappingType allows for copying of source directory to another directory
	SourcePathMappingType PathMappingType = "Source" // Source path becomes relative to source directory
	// ModifiedSourcePathMappingType allows for copying of deltas wrt source
	ModifiedSourcePathMappingType PathMappingType = "SourceDiff" // Source path becomes relative to source directory
	// PathTemplatePathMappingType allows for path template registration
	PathTemplatePathMappingType PathMappingType = "PathTemplate" // Path Template type
	// CustomTemplatePathMappingType allows copy of source to destination and applying of template with custom delimiter
	CustomTemplatePathMappingType PathMappingType = "CustomTemplate" // Source path when relative, is relative to yaml file location
)

// PathMapping is the mapping between source and intermediate files and output files
type PathMapping struct {
	Type           PathMappingType `yaml:"type,omitempty" json:"type,omitempty"` // Default - Normal copy
	SrcPath        string          `yaml:"sourcePath" json:"sourcePath" m2kpath:"normal"`
	DestPath       string          `yaml:"destinationPath" json:"destinationPath" m2kpath:"normal"` // Relative to output directory
	TemplateConfig interface{}     `yaml:"templateConfig" json:"templateConfig"`
}
