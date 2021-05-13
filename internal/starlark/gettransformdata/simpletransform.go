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

package gettransformdata

import (
	"fmt"
	"regexp"

	"github.com/konveyor/move2kube/internal/common"
	starcommon "github.com/konveyor/move2kube/internal/starlark/common"
	"github.com/konveyor/move2kube/internal/starlark/types"
	starjson "github.com/qri-io/starlib/encoding/json"
	"github.com/qri-io/starlib/util"
	log "github.com/sirupsen/logrus"
	"go.starlark.net/starlark"
)

// -----------
// File Format
// -----------
/*
"""some transforms for migrating myapp"""

def select_gpu_nodes(x):
    x["metadata"]["annotations"]["openshift.io/node-selector"] = "type=gpu-node,region=west"
    return x

def lower_number_of_replicas(x):
    x["spec"]["replicas"] = 2
    return x

def change_the_ports(x):
    x["spec"]["template"]["spec"]["containers"][0]["ports"] = query({"id" : "services.svc1.ports"})
    return x

outputs = {
    "transforms": [
        {"transform": "select_gpu_nodes", "filter": {"Namespace": ["v1"]}},
        {"transform": "lower_number_of_replicas", "filter": {"Deployment": ["v1", "v1beta1"]}},
        {"transform": "change_the_ports", "filter": {"Deployment": ["v1", "v1beta1"]}},
    ],
}
*/

// SimpleTransformT implements the TransformT interface
type SimpleTransformT struct {
	kindsAPIVersions  types.KindsAPIVersionsT
	transformFn       *starlark.Function
	dynamicQuestionFn types.DynamicQuestionFnT
}

// Various keys used in the file format.
const (
	SimpleTransformTOutputs    = "outputs"
	SimpleTransformTTransforms = "transforms"
	SimpleTransformTTransform  = "transform"
	// SimpleTransformTFilters is the key used to specify filters for each transformation.
	// It is a json object. Key and values are regex patterns. Example:
	//   "filter": {
	//     "Pod": ["v1"],
	//     "Deployment": ["v1", "v1beta1"],
	//     "Ingress": ["v1.*"],
	//   }
	// Any empty key matches all kinds. Exmaple: "": ["v1", "v1beta1"]
	// Any empty array matches all apiVersions. Exmaple: "Deployment": []
	SimpleTransformTFilters    = "filter"
	SimpleTransformTQuestionFn = "query"
)

// Transform transforms the k8s resource
func (st *SimpleTransformT) Transform(k8sResource types.K8sResourceT) (types.K8sResourceT, error) {
	log.Trace("start SimpleTransformT.Transform")
	defer log.Trace("end SimpleTransformT.Transform")
	thread := &starlark.Thread{Name: "my thread"}
	k8sResourceValue, err := util.Marshal(k8sResource)
	if err != nil {
		return k8sResource, err
	}
	transformedK8sResourceValue, err := starlark.Call(thread, st.transformFn, starlark.Tuple{k8sResourceValue}, nil)
	if err != nil {
		return k8sResource, err
	}
	transformedK8sResourceI, err := util.Unmarshal(transformedK8sResourceValue)
	if err != nil {
		return k8sResource, err
	}
	transformedK8sResource, ok := transformedK8sResourceI.(types.K8sResourceT)
	if !ok {
		return transformedK8sResource, fmt.Errorf("expected the transformed value to be a map type. Actual value %+v is of type %T", transformedK8sResourceI, transformedK8sResourceI)
	}
	return transformedK8sResource, nil
}

// Filter returns true if this transformation can be applied to the given k8s resource
func (st *SimpleTransformT) Filter(k8sResource types.K8sResourceT) (bool, error) {
	log.Trace("start SimpleTransformT.Filter")
	defer log.Trace("end SimpleTransformT.Filter")
	k8sResourceKind, k8sResourceAPIVersion, _, err := starcommon.GetInfoFromK8sResource(k8sResource)
	if err != nil {
		return false, err
	}
	if len(st.kindsAPIVersions) == 0 {
		// empty map matches all kinds and apiVersions
		return true, nil
	}
	for kind, apiVersions := range st.kindsAPIVersions {
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

// NewSimpleTransform returns a new instance of SimpleTransformT
func NewSimpleTransform(transformFn *starlark.Function, kindsAPIVersions types.KindsAPIVersionsT) *SimpleTransformT {
	log.Trace("start NewSimpleTransform")
	defer log.Trace("end NewSimpleTransform")
	return &SimpleTransformT{
		transformFn:      transformFn,
		kindsAPIVersions: kindsAPIVersions,
	}
}

// GetTransformsFromSource returns a list of transforms given the transformation script
func (st *SimpleTransformT) GetTransformsFromSource(transformStr string, dynQuesFn types.DynamicQuestionFnT) ([]types.TransformT, error) {
	log.Trace("start SimpleTransformT.GetTransformsFromSource")
	defer log.Trace("end SimpleTransformT.GetTransformsFromSource")
	st.dynamicQuestionFn = dynQuesFn
	globalsAfter, err := st.getTransformGlobals(transformStr)
	if err != nil {
		return nil, err
	}
	if err := st.validate(globalsAfter); err != nil {
		return nil, fmt.Errorf("validation failed for script. Error: %q", err)
	}
	return st.getTransformsFromGlobals(globalsAfter)
}

func (st *SimpleTransformT) getTransformGlobals(transformStr string) (starlark.StringDict, error) {
	log.Trace("start SimpleTransformT.getTransformGlobals")
	defer log.Trace("end SimpleTransformT.getTransformGlobals")
	globalsBefore, err := st.getPredeclaredVariables()
	if err != nil {
		return nil, err
	}
	thread := &starlark.Thread{Name: "m2k-starlark-transformations-thread"}
	return starlark.ExecFile(thread, "", transformStr, globalsBefore)
}

// getPredeclaredVariables gives all of the variables required to run the script and do the transformation.
func (st *SimpleTransformT) getPredeclaredVariables() (starlark.StringDict, error) {
	log.Trace("start SimpleTransformT.getPredeclaredVariables")
	defer log.Trace("end SimpleTransformT.getPredeclaredVariables")
	// TODO: fill all necessary globals required to run the script and do the transformation.
	globalsBefore, err := starjson.LoadModule()
	if err != nil {
		return globalsBefore, err
	}
	globalsBefore[SimpleTransformTQuestionFn] = starlark.NewBuiltin(SimpleTransformTQuestionFn, st.dynamicAskQuestion)
	return globalsBefore, nil
}

func (st *SimpleTransformT) dynamicAskQuestion(_ *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	log.Trace("start SimpleTransformT.dynamicAskQuestion")
	defer log.Trace("end SimpleTransformT.dynamicAskQuestion")
	log.Debugf("%s called from starlark", SimpleTransformTQuestionFn)
	log.Debugf("args: %+v", args)
	log.Debugf("kwargs: %+v", kwargs)
	argDictValue := &starlark.Dict{}
	if err := starlark.UnpackPositionalArgs(SimpleTransformTQuestionFn, args, kwargs, 1, &argDictValue); err != nil {
		return starlark.None, fmt.Errorf("invalid args provided to '%s'. Expected a single dict argument. Error: %q", SimpleTransformTQuestionFn, err)
	}
	argI, err := util.Unmarshal(argDictValue)
	if err != nil {
		return starlark.None, fmt.Errorf("failed to unmarshal the argument provided to '%s'. Expected a single dict argument. Error: %q", SimpleTransformTQuestionFn, err)
	}
	answerI, err := st.dynamicQuestionFn(argI)
	if err != nil {
		return starlark.None, err
	}
	answerValue, err := util.Marshal(answerI)
	if err != nil {
		return starlark.None, fmt.Errorf("failed to marshal the answer %+v of type %T into a starlark value. Error: %q", answerI, answerI, err)
	}
	return answerValue, err
}

// getTransformsFromGlobals is responsible for extracting transformations from the transformation script.
// This makes it very specific to the format of the script file.
func (*SimpleTransformT) getTransformsFromGlobals(transformGlobals starlark.StringDict) ([]types.TransformT, error) {
	log.Trace("start SimpleTransformT.getTransformsFromGlobals")
	defer log.Trace("end SimpleTransformT.getTransformsFromGlobals")
	ouputsValue, ok := transformGlobals[SimpleTransformTOutputs]
	if !ok {
		return nil, fmt.Errorf("the script did not set the 'outputs' global variable")
	}
	ouputsI, err := util.Unmarshal(ouputsValue)
	if err != nil {
		return nil, err
	}
	outputs, ok := ouputsI.(types.MapT)
	if !ok {
		return nil, fmt.Errorf("expected %s to be of type %T . Actual value %+v is of type %T", SimpleTransformTOutputs, types.MapT{}, ouputsI, ouputsI)
	}
	transformObjsI, ok := outputs[SimpleTransformTTransforms]
	if !ok {
		return nil, nil
	}
	transformObjs, ok := transformObjsI.([]interface{})
	if !ok {
		return nil, fmt.Errorf("expected transforms to be an array. Actual value %+v is of type %T", transformObjsI, transformObjsI)
	}
	transforms := []types.TransformT{}
	for _, transformObjI := range transformObjs {
		transformObj, ok := transformObjI.(types.MapT)
		if !ok {
			return transforms, fmt.Errorf("expected transform to be an object. Actual value %+v is of type %T", transformObjI, transformObjI)
		}

		// the transformation function
		transformFnNameI, ok := transformObj[SimpleTransformTTransform]
		if !ok {
			return transforms, fmt.Errorf("expected to find key 'transform' with the function to do the transformation. Actual map is:\n%+v", transformObj)
		}
		transformFnName, ok := transformFnNameI.(string)
		if !ok {
			return transforms, fmt.Errorf("expected key 'transform' to be a string. Actual value %+v is of type %T", transformFnNameI, transformFnNameI)
		}
		transformFnValue, ok := transformGlobals[transformFnName]
		if !ok {
			return transforms, fmt.Errorf("there is no function called %s in the transformation script. Please check the 'transform' function names", transformFnName)
		}
		var transformFnI interface{} = transformFnValue
		transformFn, ok := transformFnI.(*starlark.Function)
		if !ok {
			return transforms, fmt.Errorf("expected %s to be a function. Actual value %+v is of type %T", transformFnName, transformFnI, transformFnI)
		}

		// the filters
		kindsAPIVersionsI, ok := transformObj[SimpleTransformTFilters]
		if !ok {
			// no filter
			transforms = append(transforms, NewSimpleTransform(transformFn, nil))
			continue
		}
		kindsAPIVersionsMap, ok := kindsAPIVersionsI.(types.MapT)
		if !ok {
			return transforms, fmt.Errorf("expected filters to be of type %T . Actual value %+v is of type %T", types.MapT{}, kindsAPIVersionsI, kindsAPIVersionsI)
		}
		kindsAPIVersions := types.KindsAPIVersionsT{}
		for k, v := range kindsAPIVersionsMap {
			xs, err := common.ConvertInterfaceToSliceOfStrings(v)
			if err != nil {
				return transforms, fmt.Errorf("expected value for key %s in filters map to be an array of strings. Error: %q", k, err)
			}
			kindsAPIVersions[k] = xs
		}

		transforms = append(transforms, NewSimpleTransform(transformFn, kindsAPIVersions))
	}
	return transforms, nil
}

func (st *SimpleTransformT) validate(transformGlobals starlark.StringDict) error {
	log.Trace("start SimpleTransformT.validate")
	defer log.Trace("end SimpleTransformT.validate")
	ouputsValue, ok := transformGlobals[SimpleTransformTOutputs]
	if !ok {
		return fmt.Errorf("the script did not set the 'outputs' global variable")
	}
	ouputsI, err := util.Unmarshal(ouputsValue)
	if err != nil {
		return err
	}
	outputs, ok := ouputsI.(types.MapT)
	if !ok {
		return fmt.Errorf("expected %s to be of type %T . Actual value %+v is of type %T", SimpleTransformTOutputs, types.MapT{}, ouputsI, ouputsI)
	}
	transformsI, ok := outputs[SimpleTransformTTransforms]
	if !ok {
		return fmt.Errorf("the outputs object is missing the key '%s'", SimpleTransformTTransforms)
	}
	if _, ok := transformsI.([]interface{}); !ok {
		return fmt.Errorf("expected the key '%s' in the outputs object to contain an array. Actual value %+v is of type %T", SimpleTransformTTransforms, transformsI, transformsI)
	}
	return nil
}
