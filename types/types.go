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

package types

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// AppName represents the full app name
	AppName string = "move2kube"
	// AppNameShort represents the short app name
	AppNameShort string = "m2k"
	// GroupName is the group name use in this package
	GroupName = AppName + ".konveyor.io"
)

// TypeMeta stores apiversion and kind for resources
type TypeMeta struct {
	// APIVersion defines the versioned schema of this representation of an object.
	APIVersion string `yaml:"apiVersion,omitempty"`
	// Kind is a string value representing the resource this object represents.
	Kind string `yaml:"kind,omitempty"`
}

// ObjectMeta stores object metadata
type ObjectMeta struct {
	// Name represents the name of the resource
	Name string `yaml:"name,omitempty"`
	// Labels are Map of string keys and values that can be used to organize and categorize (scope and select) objects.
	Labels map[string]string `yaml:"labels,omitempty"`
}

// Kind stores the kind of the file
type Kind string

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1alpha1"}
)
