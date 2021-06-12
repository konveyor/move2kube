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

package common

import (
	"path/filepath"

	"github.com/konveyor/move2kube/types"
)

const (
	TempOutputRelPath = "." + types.AppNameShort
	// DefaultPlanFile defines default name for plan file
	DefaultPlanFile = types.AppNameShort + ".plan"
	// TempDirPrefix defines the prefix of the temp directory
	TempDirPrefix = types.AppNameShort + "-"
	// AssetsDir defines the dir of the assets temp directory
	AssetsDir = types.AppNameShort + "assets"
	// TranslateDir defines the dir of the assets temp directory
	TranslateDir = "translate"

	// ScriptsDir defines the directory where the output scripts are placed
	ScriptsDir = "scripts"
	// SourceDir defines the directory where the source files and folders are placed along with build scripts for each individual image
	SourceDir = "source"
	// DeployDir defines the directory where the deployment artifacts are placed
	DeployDir = "deploy"
	// CICDDir defines the directory where the deployment artifacts are placed
	CICDDir = "cicd"
	// HelmDir defines the directory where the helm charts are placed
	HelmDir = "helm-charts"
	// OCTemplatesDir defines the directory where the openshift templates are placed
	OCTemplatesDir = "openshift-templates"
)

var (
	// TempPath defines where all app data get stored during execution
	TempPath = TempDirPrefix + "temp"
	// AssetsPath defines where all assets get stored during execution
	AssetsPath = filepath.Join(TempPath, AssetsDir)
)
