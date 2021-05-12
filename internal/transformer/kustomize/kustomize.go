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

package kustomize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/konveyor/move2kube/internal/common"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
)

/*
import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/gonvenience/ytbx"
	"github.com/homeport/dyff/pkg/dyff"
	"github.com/konveyor/move2kube/internal/common"
	collecttypes "github.com/konveyor/move2kube/types/collection"
	"github.com/konveyor/move2kube/types/qaengine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/runtime"
)
*/

// PatchMetadataT is contains the target k8s resources and the patch filename
type PatchMetadataT struct {
	Path   string `yaml:"path"`
	Target struct {
		Group     string `yaml:"group"`
		Version   string `yaml:"version"`
		Kind      string `yaml:"kind"`
		Namespace string `yaml:"namespace,omitempty"`
		Name      string `yaml:"name"`
	}
}

// PatchT represents a single json patch https://tools.ietf.org/html/rfc6902
type PatchT struct {
	Op    string      `yaml:"op"`
	Path  string      `yaml:"path"`
	Value interface{} `yaml:"value"`
}

var humanReadableNodeKind = map[yaml.Kind]string{
	yaml.AliasNode:    "AliasNode",
	yaml.DocumentNode: "DocumentNode",
	yaml.MappingNode:  "MappingNode",
	yaml.ScalarNode:   "ScalarNode",
	yaml.SequenceNode: "SequenceNode",
}

// GenerateKustomize generates all the kustomize artifacts given both the original and parameterized objects.
func GenerateKustomize(kustomizePath string, filenames []string, objs, paramObjs []runtime.Object) error {
	// deploy/kustomize/base/
	kustomizeBaseDir := filepath.Join(kustomizePath, "base")
	if err := os.MkdirAll(kustomizeBaseDir, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to make the kustomize base directory at path %s . Error: %q", kustomizeBaseDir, err)
		return err
	}
	// deploy/kustomize/overlay/dev/
	kustomizeOverlayDevDir := filepath.Join(kustomizePath, "overlay", "dev")
	if err := os.MkdirAll(kustomizeOverlayDevDir, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to make the kustomize overlay dev directory at path %s . Error: %q", kustomizeOverlayDevDir, err)
		return err
	}
	// deploy/kustomize/overlay/staging/
	kustomizeOverlayStagingDir := filepath.Join(kustomizePath, "overlay", "staging")
	if err := os.MkdirAll(kustomizeOverlayStagingDir, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to make the kustomize overlay staging directory at path %s . Error: %q", kustomizeOverlayStagingDir, err)
		return err
	}
	// deploy/kustomize/overlay/prod/
	kustomizeOverlayProdDir := filepath.Join(kustomizePath, "overlay", "prod")
	if err := os.MkdirAll(kustomizeOverlayProdDir, common.DefaultDirectoryPermission); err != nil {
		log.Errorf("Failed to make the kustomize overlay prod directory at path %s . Error: %q", kustomizeOverlayProdDir, err)
		return err
	}

	patchMetadatas := []PatchMetadataT{}
	for i, obj := range objs {
		paramObj := paramObjs[i]
		filename := filenames[i]
		patchMetadata, patches, err := computeKustomizePatches(filename, obj, paramObj)
		if err != nil {
			log.Errorf("Failed to get the diff between the object:\n%+v\nand the parameterized version:\n%+v\nError: %q", obj, paramObj, err)
			continue
		}
		if len(patches) == 0 {
			continue
		}
		patchMetadatas = append(patchMetadatas, patchMetadata)
		patchesYamlBytes, err := common.ObjectToYamlBytes(patches)
		if err != nil {
			log.Errorf("Error while encoding the object to yaml. Error: %q", err)
			return err
		}
		filePath := filepath.Join(kustomizeOverlayDevDir, filename)
		if err := ioutil.WriteFile(filePath, patchesYamlBytes, common.DefaultFilePermission); err != nil {
			log.Errorf("Failed to write the patches:\n%s\nto file at path %s . Error: %q", string(patchesYamlBytes), filePath, err)
			continue
		}
		filePath = filepath.Join(kustomizeOverlayStagingDir, filename)
		if err := ioutil.WriteFile(filePath, patchesYamlBytes, common.DefaultFilePermission); err != nil {
			log.Errorf("Failed to write the patches:\n%s\nto file at path %s . Error: %q", string(patchesYamlBytes), filePath, err)
			continue
		}
		filePath = filepath.Join(kustomizeOverlayProdDir, filename)
		if err := ioutil.WriteFile(filePath, patchesYamlBytes, common.DefaultFilePermission); err != nil {
			log.Errorf("Failed to write the patches:\n%s\nto file at path %s . Error: %q", string(patchesYamlBytes), filePath, err)
			continue
		}
	}

	// Base
	// deploy/kustomize/base/kustomization.yaml
	kustFilePath := filepath.Join(kustomizeBaseDir, "kustomization.yaml")
	kustBase := map[string][]string{"resources": filenames}
	if err := common.WriteYaml(kustFilePath, kustBase); err != nil {
		log.Errorf("Failed to write the base kustomization.yaml to file at path %s:\n%+v\nError: %q", kustBase, kustFilePath, err)
	}

	// Overlays
	kustOverlay := map[string]interface{}{
		"resources": []string{"../../base"},
		"patches":   patchMetadatas,
	}
	kustOverlayYamlBytes, err := common.ObjectToYamlBytes(kustOverlay)
	if err != nil {
		log.Errorf("Error while encoding the object to yaml. Error: %q", err)
		return err
	}
	// deploy/kustomize/overlay/dev/kustomization.yaml
	kustOverlayDevFilePath := filepath.Join(kustomizeOverlayDevDir, "kustomization.yaml")
	if err := ioutil.WriteFile(kustOverlayDevFilePath, kustOverlayYamlBytes, common.DefaultFilePermission); err != nil {
		log.Errorf("Failed to write the overlay kustomization.yaml to file at path %s:\n%+v\nError: %q", string(kustOverlayYamlBytes), kustOverlayDevFilePath, err)
	}
	// deploy/kustomize/overlay/staging/kustomization.yaml
	kustOverlayStagingFilePath := filepath.Join(kustomizeOverlayStagingDir, "kustomization.yaml")
	if err := ioutil.WriteFile(kustOverlayStagingFilePath, kustOverlayYamlBytes, common.DefaultFilePermission); err != nil {
		log.Errorf("Failed to write the overlay kustomization.yaml to file at path %s:\n%+v\nError: %q", string(kustOverlayYamlBytes), kustOverlayStagingFilePath, err)
	}
	// deploy/kustomize/overlay/prod/kustomization.yaml
	kustOverlayProdFilePath := filepath.Join(kustomizeOverlayProdDir, "kustomization.yaml")
	if err := ioutil.WriteFile(kustOverlayProdFilePath, kustOverlayYamlBytes, common.DefaultFilePermission); err != nil {
		log.Errorf("Failed to write the overlay kustomization.yaml to file at path %s:\n%+v\nError: %q", string(kustOverlayYamlBytes), kustOverlayProdFilePath, err)
	}

	return nil
}

// computeKustomizePatches returns the json patches https://kubectl.docs.kubernetes.io/references/kustomize/glossary/#patchjson6902
func computeKustomizePatches(filename string, obj, paramObj runtime.Object) (PatchMetadataT, []PatchT, error) {
	metadata := getMetadata(filename, obj)
	tempDirForParam, err := ioutil.TempDir("", "parameterize-*")
	if err != nil {
		log.Errorf("Failed to create a temporary directory for computing diffs between objects. Error: %q", err)
		return metadata, nil, err
	}
	patches, err := getKustomizePatches(obj, paramObj, tempDirForParam)
	if err != nil {
		log.Errorf("Failed to get the patches. Error: %q", err)
		return metadata, nil, err
	}
	return metadata, patches, nil
}

func getMetadata(filename string, obj runtime.Object) PatchMetadataT {
	metadata := PatchMetadataT{}
	metadata.Path = filename
	metadata.Target.Group = obj.GetObjectKind().GroupVersionKind().Group
	metadata.Target.Version = obj.GetObjectKind().GroupVersionKind().Version
	metadata.Target.Kind = obj.GetObjectKind().GroupVersionKind().Kind
	objMetadata := common.GetRuntimeObjectMetadata(obj)
	metadata.Target.Namespace = objMetadata.Namespace
	metadata.Target.Name = objMetadata.Name
	return metadata
}

func getKustomizePatches(obj1, obj2 runtime.Object, tempDirForParam string) ([]PatchT, error) {
	log.Tracef("getPatches start")
	defer log.Tracef("getPatches end")

	x, err := common.MarshalObjToYaml(obj1)
	if err != nil {
		log.Errorf("Failed to marshal the 1st runtime object:\n%+v\nto yaml. Error: %q", obj1, err)
		return nil, err
	}
	y, err := common.MarshalObjToYaml(obj2)
	if err != nil {
		log.Errorf("Failed to marshal the 2nd runtime object:\n%+v\nto yaml. Error: %q", obj2, err)
		return nil, err
	}
	xFilePath := filepath.Join(tempDirForParam, "x.yaml")
	if err := ioutil.WriteFile(xFilePath, []byte(x), common.DefaultFilePermission); err != nil {
		log.Errorf("Failed to write k8s yaml to file at path %s . Error: %q", xFilePath, err)
		return nil, err
	}
	yFilePath := filepath.Join(tempDirForParam, "y.yaml")
	if err := ioutil.WriteFile(yFilePath, []byte(y), common.DefaultFilePermission); err != nil {
		log.Errorf("Failed to write k8s yaml to file at path %s . Error: %q", yFilePath, err)
		return nil, err
	}
	xFile, yFile, err := ytbx.LoadFiles(xFilePath, yFilePath)
	if err != nil {
		log.Errorf("Failed to load k8s yaml files from paths %s and %s . Error: %q", xFilePath, yFilePath, err)
		return nil, err
	}
	report, err := dyff.CompareInputFiles(xFile, yFile, dyff.IgnoreOrderChanges(true), dyff.KubernetesEntityDetection(true))
	if err != nil {
		log.Errorf("Failed to compare k8s yaml files:\n%+v\nand\n%+v\nError: %q", xFile, yFile, err)
		return nil, err
	}

	return getPatchContents(report, xFile.Documents[0])

}

func getPatchContents(report dyff.Report, xFile *yaml.Node) ([]PatchT, error) {
	log.Trace("getPatchContents start")
	log.Trace("getPatchContents end")

	patchContents := []PatchT{}
	for i, diff := range report.Diffs {
		log.Debugf("diff[%d]: %+v", i, diff)
		newPatches, err := getPatches(report, xFile, diff)
		if err != nil {
			log.Debug(err)
			continue
		}
		patchContents = append(patchContents, newPatches...)
	}
	return patchContents, nil
}

func getPatches(report dyff.Report, xFile *yaml.Node, diff dyff.Diff) ([]PatchT, error) {
	log.Trace("getPatch start")
	log.Trace("getPatch end")

	if len(diff.Details) == 0 {
		return nil, fmt.Errorf("no details found for the diff: %+v", diff)
	}
	if len(diff.Details) > 1 {
		// TODO: might have to deal with case where there are 2 details (-) and (+)
		return nil, fmt.Errorf("more than 1 detail found for the diff: %+v", diff)
	}
	detail := diff.Details[0]
	newPaths, newNodes, err := detailToPatches(diff.Path, xFile, detail)
	if err != nil {
		return nil, fmt.Errorf("failed to get the patches for the diff with path %+v and detail %+v . Error: %q", diff.Path, detail, err)
	}
	if len(newPaths) == 0 {
		return nil, fmt.Errorf("got 0 paths for the json patch operation. Diff: %+v Detail: %+v", diff, detail)
	}

	log.Debugf("new paths: %+v", newPaths)

	if detail.Kind == dyff.MODIFICATION {
		if detail.From == nil {
			return nil, fmt.Errorf("the FROM value is empty in the json patch replace operation")
		}
		var v interface{}
		if err := detail.From.Decode(&v); err != nil {
			log.Debug(err)
			return nil, err
		}
		return []PatchT{{
			Op:    "replace",
			Path:  newPaths[0],
			Value: v,
		}}, nil
	}

	if detail.Kind == dyff.REMOVAL {
		if len(newPaths) > 1 {
			return nil, fmt.Errorf("json patch REMOVAL with removing multiple nodes from list is not supported")
		}
		return []PatchT{{
			Op:   "remove",
			Path: newPaths[0],
		}}, nil
	}

	if detail.Kind != dyff.ADDITION {
		return nil, fmt.Errorf("unknown json patch operation. Detail: %+v", detail)
	}

	patches := []PatchT{}
	for i, newPath := range newPaths {
		newNode := detail.To
		if len(newNodes) > 0 {
			newNode = newNodes[i]
		}
		var v interface{}
		if newNode == nil {
			log.Debug("The TO value is empty in the json patch addition operation")
			continue
		}
		if err := newNode.Decode(&v); err != nil {
			log.Debug(err)
			continue
		}
		patches = append(patches, PatchT{
			Op:    "add",
			Path:  newPath,
			Value: v,
		})
	}
	return patches, nil
}

func detailToPatches(path ytbx.Path, value *yaml.Node, detail dyff.Detail) ([]string, []*yaml.Node, error) {
	jsonPointer, err := pathToJSONPointer6901(path, value)
	if err != nil {
		return nil, nil, err
	}

	if detail.Kind == dyff.MODIFICATION {
		return []string{jsonPointer}, nil, nil
	}

	if detail.Kind == dyff.ADDITION {
		return detailToPatchesAddition(jsonPointer, path, value, detail)
	}

	if detail.Kind == dyff.REMOVAL {
		return detailToPatchesRemoval(jsonPointer, path, value, detail)
	}

	return nil, nil, fmt.Errorf("unknown diff operation. Detail: %+v", detail)
}

// detailToPatchesAddition is the special case to handle addition of a node to a list
func detailToPatchesAddition(jsonPointer string, path ytbx.Path, value *yaml.Node, detail dyff.Detail) ([]string, []*yaml.Node, error) {
	log.Trace("detailToPatchesAddition start")
	log.Trace("detailToPatchesAddition end")

	log.Debugf("jsonPointer: %s", jsonPointer)
	if detail.To == nil {
		return nil, nil, fmt.Errorf("the json patch ADDITION operation has an empty TO field: %+v", detail)
	}
	log.Debugf("To: %+v", detail.To)
	someNode := detail.To
	kindName := humanReadableNodeKind[someNode.Kind]
	log.Debugf("The yaml node kind is %s", kindName)
	if someNode.Kind != yaml.SequenceNode {
		// normal addition
		return []string{jsonPointer}, nil, nil
	}
	// adding one or more nodes to a list
	if len(someNode.Content) == 0 {
		return nil, nil, fmt.Errorf("the list of nodes we are trying to add is empty")
	}
	jsonPointer += "/-" // append the node(s) to the end of the list
	if len(someNode.Content) > 1 {
		log.Debugf("Adding multiple nodes to a list.")
	}
	paths := []string{}
	for k := 0; k < len(someNode.Content); k++ {
		paths = append(paths, jsonPointer)
	}
	return paths, someNode.Content, nil
}

// detailToPatchesRemoval is the special case to handle removal of a node from a list
func detailToPatchesRemoval(jsonPointer string, path ytbx.Path, value *yaml.Node, detail dyff.Detail) ([]string, []*yaml.Node, error) {
	log.Trace("detailToPatchesRemoval start")
	log.Trace("detailToPatchesRemoval end")

	log.Debugf("jsonPointer: %s", jsonPointer)
	if detail.From == nil {
		return nil, nil, fmt.Errorf("the json patch REMOVAL operation has an empty FROM field: %+v", detail)
	}
	listWeAreRemovingFrom, err := ytbx.Grab(value, jsonPointer)
	if err != nil {
		return nil, nil, err
	}
	log.Debugf("listWeAreRemovingFrom: %+v", listWeAreRemovingFrom)

	if listWeAreRemovingFrom.Kind != yaml.SequenceNode {
		log.Debugf("Not removing from a list.")
		return []string{jsonPointer}, nil, nil
	}

	someNode := detail.From
	kindName := humanReadableNodeKind[someNode.Kind]
	log.Debugf("From: %+v and kind is %s", detail.From, kindName)

	if someNode.Kind != yaml.SequenceNode {
		return nil, nil, fmt.Errorf("expected to find sequence node. Actual: %+v", someNode)
	}

	log.Debugf("sequence: %+v", someNode)
	if len(someNode.Content) == 0 {
		return nil, nil, fmt.Errorf("the list of nodes we are trying to remove is empty")
	}
	if len(someNode.Content) > 1 {
		return nil, nil, fmt.Errorf("removing multiple nodes from the list is not supported")
	}
	removedNode := someNode.Content[0]
	// find the index of the node we are remving from the list
	idx := -1
	for k, cont := range listWeAreRemovingFrom.Content {
		if cont == removedNode {
			idx = k
			break
		}
	}
	if idx == -1 {
		return nil, nil, fmt.Errorf("failed to find the node in the list we are removing from. List: %+v Node: %+v", listWeAreRemovingFrom, removedNode)
	}
	jsonPointer += "/" + cast.ToString(idx) // index in the list that should be removed
	return []string{jsonPointer}, nil, nil
}

func pathToJSONPointer6901(path ytbx.Path, value *yaml.Node) (string, error) {
	jsonPointer := ""
	for _, pathElement := range path.PathElements {
		if pathElement.Key == "" {
			if pathElement.Idx == -1 {
				jsonPointer += "/" + pathElement.Name
				continue
			}
			jsonPointer += "/" + cast.ToString(pathElement.Idx)
			continue
		}
		someNode, err := ytbx.Grab(value, jsonPointer)
		if err != nil {
			log.Debugf("Failed to grab using path %s . Error: %q", jsonPointer, err)
			return jsonPointer, err
		}
		if someNode.Kind != yaml.SequenceNode {
			return jsonPointer, fmt.Errorf("expected a sequence node. Actual: %+v", value)
		}
		idx := -1
		for i, x := range someNode.Content {
			if x.Kind != yaml.MappingNode {
				return jsonPointer, fmt.Errorf("expected a mapping node. Actual: %+v", x)
			}
			var xMap map[string]interface{}
			if err := x.Decode(&xMap); err != nil {
				return jsonPointer, fmt.Errorf("failed to decode the node into a map. Node: %+v Error: %q", x, err)
			}
			if xMap[pathElement.Key] == pathElement.Name {
				idx = i
				break
			}
		}
		if idx == -1 {
			return jsonPointer, fmt.Errorf("failed to find the object with the key value pair [%s]: [%s] in the list %+v", pathElement.Key, pathElement.Name, someNode.Content)
		}
		jsonPointer += "/" + cast.ToString(idx)
	}
	return jsonPointer, nil
}
