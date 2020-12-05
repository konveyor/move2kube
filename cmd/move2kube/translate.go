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

package main

import (
	"os"
	"path/filepath"

	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/move2kube"
	"github.com/konveyor/move2kube/internal/qaengine"
	"github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type translateFlags struct {
	ignoreEnv    bool
	planfile     string
	outpath      string
	srcpath      string
	name         string
	curate       bool
	qadisablecli bool
	qaskip       bool
	qaport       int
	qacaches     []string
}

const (
	outpathFlag      = "outpath"
	curateFlag       = "curate"
	qadisablecliFlag = "qadisablecli"
	qaskipFlag       = "qaskip"
	qaportFlag       = "qaport"
	qacacheFlag      = "qacache"
)

func checkSourcePath(srcpath string) {
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

func checkOutputPath(outpath string) {
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

func createOutputDirectoryAndCacheFile(outpath string) {
	if err := os.MkdirAll(outpath, common.DefaultDirectoryPermission); err != nil {
		log.Fatalf("Failed to create the output directory at path %s Error: %q", outpath, err)
	}
	cacheFilePath := filepath.Join(outpath, common.QACacheFile)
	log.Debugf("Creating the cache file at path %s", cacheFilePath)
	if err := qaengine.SetWriteCache(cacheFilePath); err != nil {
		log.Warnf("Unable to write the cache file to path %q Error: %q", cacheFilePath, err)
	}
}

func translateHandler(cmd *cobra.Command, flags translateFlags) {
	// Setup
	var err error
	ignoreEnv := flags.ignoreEnv
	planfile := flags.planfile
	srcpath := flags.srcpath
	outpath := flags.outpath
	name := flags.name

	curate := flags.curate
	qadisablecli := flags.qadisablecli
	qaskip := flags.qaskip
	qaport := flags.qaport
	qacaches := flags.qacaches

	if planfile, err = filepath.Abs(planfile); err != nil {
		log.Fatalf("Failed to make the plan file path %q absolute. Error: %q", planfile, err)
	}
	if srcpath != "" {
		if srcpath, err = filepath.Abs(srcpath); err != nil {
			log.Fatalf("Failed to make the source directory path %q absolute. Error: %q", srcpath, err)
		}
	}
	if outpath, err = filepath.Abs(outpath); err != nil {
		log.Fatalf("Failed to make the output directory path %q absolute. Error: %q", outpath, err)
	}

	// Global settings
	common.IgnoreEnvironment = ignoreEnv
	qaengine.StartEngine(qaskip, qaport, qadisablecli)
	cachepaths := []string{}
	for i := len(qacaches) - 1; i >= 0; i-- {
		cachepaths = append(cachepaths, qacaches[i])
	}
	qaengine.AddCaches(cachepaths)

	// Parameter cleaning and curate plan
	var p plan.Plan
	fi, err := os.Stat(planfile)
	if err == nil && fi.IsDir() {
		planfile = filepath.Join(planfile, common.DefaultPlanFile)
		_, err = os.Stat(planfile)
	}
	if err != nil {
		log.Debugf("No plan file found.")
		if cmd.Flags().Changed(planFlag) || !cmd.Flags().Changed(sourceFlag) {
			// They specified path to a plan file which doesn't exist.
			// OR
			// They didn't specify path to a source directory.
			log.Fatalf("Error while accessing plan file at path %s Error: %q", planfile, err)
		}
		checkSourcePath(srcpath)
		log.Debugf("Creating a new plan.")
		p = move2kube.CreatePlan(srcpath, name)
		outpath = filepath.Join(outpath, p.Name)
		checkOutputPath(outpath)
		createOutputDirectoryAndCacheFile(outpath)
		p = move2kube.CuratePlan(p)
	} else {
		log.Infof("Detected a plan file at path %s. Will translate using this plan.", planfile)
		if p, err = plan.ReadPlan(planfile); err != nil {
			log.Fatalf("Unable to read the plan at path %s Error: %q", planfile, err)
		}
		if len(p.Spec.Inputs.Services) == 0 {
			log.Fatalf("Failed to find any services. Aborting.")
		}
		if cmd.Flags().Changed(nameFlag) {
			p.Name = name
		}
		if cmd.Flags().Changed(sourceFlag) {
			if err := p.SetRootDir(srcpath); err != nil {
				log.Fatalf("Failed to set the root directory to %q Error: %q", srcpath, err)
			}
		}
		checkSourcePath(p.Spec.Inputs.RootDir)
		outpath = filepath.Join(outpath, p.Name)
		checkOutputPath(outpath)
		createOutputDirectoryAndCacheFile(outpath)
		if curate {
			p = move2kube.CuratePlan(p)
		}
	}

	// Translate
	move2kube.Translate(p, outpath, qadisablecli)
	log.Infof("Translated target artifacts can be found at [%s].", outpath)
}

func init() {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := translateFlags{}
	translateCmd := &cobra.Command{
		Use:   "translate",
		Short: "Translate using move2kube plan",
		Long:  "Translate artifacts using move2kube plan",
		Run:   func(cmd *cobra.Command, _ []string) { translateHandler(cmd, flags) },
	}

	// Basic options
	translateCmd.Flags().StringVarP(&flags.planfile, planFlag, "p", common.DefaultPlanFile, "Specify a plan file to execute.")
	translateCmd.Flags().BoolVarP(&flags.curate, curateFlag, "c", false, "Specify whether to curate the plan with a q/a.")
	translateCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify source directory to translate. If you already have a m2k.plan then this will override the rootdir value specified in that plan.")
	translateCmd.Flags().StringVarP(&flags.outpath, outpathFlag, "o", ".", "Path for output. Default will be directory with the project name.")
	translateCmd.Flags().StringVarP(&flags.name, nameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	translateCmd.Flags().StringSliceVarP(&flags.qacaches, qacacheFlag, "q", []string{}, "Specify qa cache file locations")

	// Advanced options
	translateCmd.Flags().BoolVar(&flags.ignoreEnv, ignoreEnvFlag, false, "Ignore data from local machine.")

	// Hidden options
	translateCmd.Flags().BoolVar(&flags.qadisablecli, qadisablecliFlag, false, "Enable/disable the QA Cli sub-system. Without this system, you will have to use the REST API to interact.")
	translateCmd.Flags().BoolVar(&flags.qaskip, qaskipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")
	translateCmd.Flags().IntVar(&flags.qaport, qaportFlag, 0, "Port for the QA service. By default it chooses a random free port.")

	must(translateCmd.Flags().MarkHidden(qadisablecliFlag))
	must(translateCmd.Flags().MarkHidden(qaskipFlag))
	must(translateCmd.Flags().MarkHidden(qaportFlag))

	rootCmd.AddCommand(translateCmd)
}
