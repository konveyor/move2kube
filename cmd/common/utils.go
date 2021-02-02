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
	"os"

	log "github.com/sirupsen/logrus"
)

const (
	// SourceFlag is the name of the flag that contains path to the source folder
	SourceFlag = "source"
	// OutputFlag is the name of the flag that contains path to the output folder
	OutputFlag = "output"
	// QACacheFlag is the name of the flag that contains list of qacache files
	QACacheFlag = "qacache"
	// NameFlag is the name of the flag that contains the project name
	NameFlag = "name"
	// PlanFlag is the name of the flag that contains the path to the plan file
	PlanFlag = "plan"
	// IgnoreEnvFlag is the name of the flag that tells us whether to use data collected from the local machine
	IgnoreEnvFlag = "ignoreenv"
	// QASkipFlag is the name of the flag that let's you skip all the question answers
	QASkipFlag = "qaskip"
	// ConfigFlag is the name of the flag that contains list of config files
	ConfigFlag = "config"
	// SetConfigFlag is the name of the flag that contains list of key-value configs
	SetConfigFlag = "setconfig"
	// PreSetFlag is the name of the flag that contains list of preset configurations to use
	PreSetFlag = "preset"
)

//TranslateFlags to store values from command line paramters
type TranslateFlags struct {
	//IgnoreEnv tells us whether to use data collected from the local machine
	IgnoreEnv bool
	//Planfile is contains the path to the plan file
	Planfile string
	//Outpath contains path to the output folder
	Outpath string
	//SourceFlag contains path to the source folder
	Srcpath string
	//Name contains the project name
	Name string
	//Qacaches contains list of qacache files
	Qacaches []string
	//Configs contains list of config files
	Configs []string
	//Configs contains list of key-value configs
	Setconfigs []string
	//Qaskip let's you skip all the question answers
	Qaskip bool
	//PreSets contains list of preset configurations
	PreSets []string
}

// CheckSourcePath checks if the source path is an existing directory.
func CheckSourcePath(srcpath string) {
	fi, err := os.Stat(srcpath)
	if os.IsNotExist(err) {
		log.Fatalf("The given source directory %s does not exist. Error: %q", srcpath, err)
	}
	if err != nil {
		log.Fatalf("Error while accessing the given source directory %s Error: %q", srcpath, err)
	}
	if !fi.IsDir() {
		log.Fatalf("The given source path %s is a file. Expected a directory. Exiting.", srcpath)
	}
}

// CheckOutputPath checks if the output path is already in use.
func CheckOutputPath(outpath string) {
	fi, err := os.Stat(outpath)
	if os.IsNotExist(err) {
		log.Debugf("Translated artifacts will be written to %s", outpath)
		return
	}
	if err != nil {
		log.Fatalf("Error while accessing output directory at path %s Error: %q . Exiting", outpath, err)
	}
	if !fi.IsDir() {
		log.Fatalf("Output path %s is a file. Expected a directory. Exiting", outpath)
	}
	log.Infof("Output directory %s exists. The contents might get overwritten.", outpath)
}
