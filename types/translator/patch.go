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
	irtypes "github.com/konveyor/move2kube/types/ir"
)

/*
Features to support:
1. Allow for template composing
2. Allow for multiple templates from files
3. Allow for new files to be copied - if it was executed from within container, it will be copied from inside the container
4. Allow for existing files to be edited - like pom.xml
5. Allow for existing files to be edited - compute the patch automatically
*/

/*
{
  paths: [
	  {
      srcfilepath: srcfilepath,
      destfilepath : destfilepath,
    },
  ],
  templatepaths : [
    // If the same dest is present for more than one file, they get appended
    // All templates should be in either /m2k/templates directory in container or in templates relative directory to the containerizer yaml
    {
      filename : filename,
      destfilepath : destfilepath,
    },
  ],
  templates : [
	  {
      destinationFilePath: destfilepath,
      template : template,
    },
  ],
  computepatches: [
    {
      filepathInSource: filepathInSource,
      destfilePath: destfilepath, // Optional automatically computed
    }
  ],
  patches: [
	  {
      filepathInSource: filepathInSource,
      destfilePath: destfilepath, // Optional automatically computed
      patchstring : patchstring,
      type : jsonpatch // should support xmlpatch, jsonpatch, yamlpatch, linuxpatch depending on file extension - Optinal automatically computed
    }
  ],
  config : {},
  ir : irtypes.IR,
}
*/

type Patch struct {
	Paths           []Path           `json:"paths"`
	TemplatePaths   []TemplatePath   `json:"templatePaths"`
	Templates       []Template       `json:"templates"`
	DeltasToCompute []DeltaToCompute `json:"patchesToCompute"`
	Deltas          []Delta          `json:"daltas"`
	Config          interface{}      `json:"config"`
	IR              irtypes.IR       `json:"ir"`
}

type Path struct {
	SrcFilePath  string `json:"sourceFilePath"`
	DestFilePath string `json:"destinationFilePath"`
}

type TemplatePath struct {
	Filename     string `json:"fileName"`
	DestFilePath string `json:"destinationFilePath"`
}

type Template struct {
	DestFilePath string `json:"destinationFilePath"`
	Template     string `json:"template"`
}

type DeltaToCompute struct {
	FilesInSource       string `json:"filesInSource"`
	DestinationFilePath string `json:"destinationFilePath"`
}

type Delta struct {
	FilePathInSource    string `json:"filePathInSource"`
	DestinationFilePath string `json:"destinationFilePath"`
	PatchString         string `json:"patchString"`
	Type                string `json:"type"` // should support xmlpatch, jsonpatch, yamlpatch, linuxpatch depending on file extension - Optinal automatically computed
}
