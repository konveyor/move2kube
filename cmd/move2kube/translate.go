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

	cmdcommon "github.com/konveyor/move2kube/cmd/common"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/move2kube"
	"github.com/konveyor/move2kube/internal/qaengine"
	"github.com/konveyor/move2kube/types/plan"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type translateFlags struct {
	cmdcommon.TranslateFlags
	curate       bool
	qadisablecli bool
	qaport       int
}

const (
	curateFlag       = "curate"
	qadisablecliFlag = "qadisablecli"
	qaportFlag       = "qaport"
)

func translateHandler(cmd *cobra.Command, flags translateFlags) {
	// Setup
	var err error

	if flags.Planfile, err = filepath.Abs(flags.Planfile); err != nil {
		log.Fatalf("Failed to make the plan file path %q absolute. Error: %q", flags.Planfile, err)
	}
	if flags.Srcpath != "" {
		if flags.Srcpath, err = filepath.Abs(flags.Srcpath); err != nil {
			log.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.Srcpath, err)
		}
	}
	if flags.Outpath, err = filepath.Abs(flags.Outpath); err != nil {
		log.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.Outpath, err)
	}

	// Global settings
	common.IgnoreEnvironment = flags.IgnoreEnv
	// Global settings

	// Parameter cleaning and curate plan
	var p plan.Plan
	fi, err := os.Stat(flags.Planfile)
	if err == nil && fi.IsDir() {
		flags.Planfile = filepath.Join(flags.Planfile, common.DefaultPlanFile)
		_, err = os.Stat(flags.Planfile)
	}
	if err != nil {
		log.Debugf("No plan file found.")
		if cmd.Flags().Changed(cmdcommon.PlanFlag) {
			log.Fatalf("Error while accessing plan file at path %s Error: %q", flags.Planfile, err)
		}
		if !cmd.Flags().Changed(cmdcommon.SourceFlag) {
			log.Fatalf("Invalid usage. Must specify either path to a plan file or path to directory containing source code.")
		}

		// Global settings
		cmdcommon.CheckSourcePath(flags.Srcpath)
		flags.Outpath = filepath.Join(flags.Outpath, flags.Name)
		cmdcommon.CheckOutputPath(flags.Outpath, flags.Overwrite)
		if flags.Srcpath == flags.Outpath || common.IsParent(flags.Outpath, flags.Srcpath) || common.IsParent(flags.Srcpath, flags.Outpath) {
			log.Fatalf("The source path %s and output path %s overlap.", flags.Srcpath, flags.Outpath)
		}
		if err := os.MkdirAll(flags.Outpath, common.DefaultDirectoryPermission); err != nil {
			log.Fatalf("Failed to create the output directory at path %s Error: %q", flags.Outpath, err)
		}
		qaengine.StartEngine(flags.Qaskip, flags.qaport, flags.qadisablecli)
		qaengine.SetupConfigFile(flags.Outpath, flags.Setconfigs, flags.Configs, flags.PreSets)
		qaengine.SetupCacheFile(flags.Outpath, flags.Qacaches)
		if err := qaengine.WriteStoresToDisk(); err != nil {
			log.Warnf("Failed to write the stores to disk. Error: %q", err)
		}
		// Global settings

		log.Debugf("Creating a new plan.")
		p = move2kube.CreatePlan(flags.Srcpath, flags.Name, true)
		p = move2kube.CuratePlan(p)
	} else {
		log.Infof("Detected a plan file at path %s. Will translate using this plan.", flags.Planfile)
		if p, err = plan.ReadPlan(flags.Planfile); err != nil {
			log.Fatalf("Unable to read the plan at path %s Error: %q", flags.Planfile, err)
		}
		if len(p.Spec.Inputs.Services) == 0 {
			if len(p.Spec.Inputs.K8sFiles) == 0 {
				log.Fatalf("Failed to find any services. Aborting.")
			} else {
				log.Infof("No services found. Proceeding for kubernetes artifacts translation.")
			}
		}
		if cmd.Flags().Changed(cmdcommon.NameFlag) {
			p.Name = flags.Name
		}
		if cmd.Flags().Changed(cmdcommon.SourceFlag) {
			if err := p.SetRootDir(flags.Srcpath); err != nil {
				log.Fatalf("Failed to set the root directory to %q Error: %q", flags.Srcpath, err)
			}
		}

		// Global settings
		cmdcommon.CheckSourcePath(p.Spec.Inputs.RootDir)
		flags.Outpath = filepath.Join(flags.Outpath, p.Name)
		cmdcommon.CheckOutputPath(flags.Outpath, flags.Overwrite)
		if p.Spec.Inputs.RootDir == flags.Outpath || common.IsParent(flags.Outpath, p.Spec.Inputs.RootDir) || common.IsParent(p.Spec.Inputs.RootDir, flags.Outpath) {
			log.Fatalf("The source path %s and output path %s overlap.", p.Spec.Inputs.RootDir, flags.Outpath)
		}
		if err := os.MkdirAll(flags.Outpath, common.DefaultDirectoryPermission); err != nil {
			log.Fatalf("Failed to create the output directory at path %s Error: %q", flags.Outpath, err)
		}
		qaengine.StartEngine(flags.Qaskip, flags.qaport, flags.qadisablecli)
		qaengine.SetupConfigFile(flags.Outpath, flags.Setconfigs, flags.Configs, flags.PreSets)
		qaengine.SetupCacheFile(flags.Outpath, flags.Qacaches)
		if err := qaengine.WriteStoresToDisk(); err != nil {
			log.Warnf("Failed to write the stores to disk. Error: %q", err)
		}
		// Global settings

		if flags.curate {
			p = move2kube.CuratePlan(p)
		}
	}

	// Translate
	normalizedTransformPaths, err := cmdcommon.NormalizePaths(flags.TransformPaths, []string{".star"})
	if err != nil {
		log.Fatalf("Failed to clean the paths:\n%+v\nError: %q", flags.TransformPaths, err)
	}
	move2kube.Translate(p, flags.Outpath, flags.qadisablecli, normalizedTransformPaths)
	log.Infof("Translated target artifacts can be found at [%s].", flags.Outpath)
}

func getTranslateCommand() *cobra.Command {
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
	translateCmd.Flags().StringVarP(&flags.Planfile, cmdcommon.PlanFlag, "p", common.DefaultPlanFile, "Specify a plan file to execute.")
	translateCmd.Flags().BoolVarP(&flags.curate, curateFlag, "c", false, "Specify whether to curate the plan with a q/a.")
	translateCmd.Flags().BoolVar(&flags.Overwrite, cmdcommon.OverwriteFlag, false, "Overwrite the output directory if it exists. By default we don't overwrite.")
	translateCmd.Flags().StringVarP(&flags.Srcpath, cmdcommon.SourceFlag, "s", "", "Specify source directory to translate. If you already have a m2k.plan then this will override the rootdir value specified in that plan.")
	translateCmd.Flags().StringVarP(&flags.Outpath, cmdcommon.OutputFlag, "o", ".", "Path for output. Default will be directory with the project name.")
	translateCmd.Flags().StringVarP(&flags.Name, cmdcommon.NameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	translateCmd.Flags().StringSliceVarP(&flags.Qacaches, cmdcommon.QACacheFlag, "q", []string{}, "Specify qa cache file locations")
	translateCmd.Flags().StringSliceVarP(&flags.Configs, cmdcommon.ConfigFlag, "f", []string{}, "Specify config file locations")
	translateCmd.Flags().StringSliceVarP(&flags.PreSets, cmdcommon.PreSetFlag, "r", []string{}, "Specify preset config to use")
	translateCmd.Flags().StringArrayVarP(&flags.Setconfigs, cmdcommon.SetConfigFlag, "k", []string{}, "Specify config key-value pairs")
	translateCmd.Flags().StringSliceVarP(&flags.TransformPaths, cmdcommon.TransformsFlag, "t", []string{}, "Specify paths to the transformation scripts to apply. Can be the path to a script or the path to a folder containing the scripts.")

	// Advanced options
	translateCmd.Flags().BoolVar(&flags.IgnoreEnv, cmdcommon.IgnoreEnvFlag, false, "Ignore data from local machine.")

	// Hidden options
	translateCmd.Flags().BoolVar(&flags.qadisablecli, qadisablecliFlag, false, "Enable/disable the QA Cli sub-system. Without this system, you will have to use the REST API to interact.")
	translateCmd.Flags().BoolVar(&flags.Qaskip, cmdcommon.QASkipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")
	translateCmd.Flags().IntVar(&flags.qaport, qaportFlag, 0, "Port for the QA service. By default it chooses a random free port.")

	must(translateCmd.Flags().MarkHidden(qadisablecliFlag))
	must(translateCmd.Flags().MarkHidden(qaportFlag))

	return translateCmd
}
