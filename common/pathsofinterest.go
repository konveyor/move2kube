package common

import (
	"path/filepath"

	"github.com/konveyor/move2kube-wasm/types"
)

const (
	// DefaultPlanFile is the default name for the plan file
	DefaultPlanFile = types.AppNameShort + ".plan"
	// DefaultConfigFilePath is the default config file path
	DefaultConfigFilePath = types.AppNameShort + "-default-config.yaml"
	// DefaultCustomizationDir is the default path for the customization directory
	DefaultCustomizationDir = types.AppNameShort + "-default-customizations"
	// TempDirPrefix defines the prefix of the temp directory
	TempDirPrefix = types.AppNameShort + "-"
	// AssetsDir defines the dir of the assets temp directory
	AssetsDir = types.AppNameShort + "assets"

	// ScriptsDir defines the directory where the output scripts are placed
	ScriptsDir = "scripts"
	// DefaultSourceDir defines the directory where the source files and folders are placed along with build scripts for each individual image
	DefaultSourceDir = "source"
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
	// RemoteTempPath defines where all remote sources data get stored during execution
	RemoteTempPath = TempDirPrefix + "remote-temp"
)
