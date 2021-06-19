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

	"github.com/konveyor/move2kube/api"
	cmdcommon "github.com/konveyor/move2kube/cmd/common"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type transformFlags struct {
	cmdcommon.TransformFlags
	qadisablecli bool
	qaport       int
}

const (
	qadisablecliFlag = "qadisablecli"
	qaportFlag       = "qaport"
)

func transformHandler(cmd *cobra.Command, flags transformFlags) {
	// Setup
	var err error

	if flags.Planfile, err = filepath.Abs(flags.Planfile); err != nil {
		logrus.Fatalf("Failed to make the plan file path %q absolute. Error: %q", flags.Planfile, err)
	}
	if flags.Srcpath != "" {
		if flags.Srcpath, err = filepath.Abs(flags.Srcpath); err != nil {
			logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.Srcpath, err)
		}
	}
	if flags.Outpath, err = filepath.Abs(flags.Outpath); err != nil {
		logrus.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.Outpath, err)
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
		logrus.Debugf("No plan file found.")
		if cmd.Flags().Changed(cmdcommon.PlanFlag) {
			logrus.Fatalf("Error while accessing plan file at path %s Error: %q", flags.Planfile, err)
		}
		if !cmd.Flags().Changed(cmdcommon.SourceFlag) {
			logrus.Fatalf("Invalid usage. Must specify either path to a plan file or path to directory containing source code.")
		}

		// Global settings
		cmdcommon.CheckSourcePath(flags.Srcpath)
		flags.Outpath = filepath.Join(flags.Outpath, flags.Name)
		cmdcommon.CheckOutputPath(flags.Outpath, flags.Overwrite)
		if flags.Srcpath == flags.Outpath || common.IsParent(flags.Outpath, flags.Srcpath) || common.IsParent(flags.Srcpath, flags.Outpath) {
			logrus.Fatalf("The source path %s and output path %s overlap.", flags.Srcpath, flags.Outpath)
		}
		if err := os.MkdirAll(flags.Outpath, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.Outpath, err)
		}
		qaengine.StartEngine(flags.Qaskip, flags.qaport, flags.qadisablecli)
		qaengine.SetupConfigFile(filepath.Join(flags.Outpath, common.ConfigFile), flags.Setconfigs, flags.Configs, flags.PreSets)
		qaengine.SetupWriteCacheFile(filepath.Join(flags.Outpath, common.QACacheFile))
		if err := qaengine.WriteStoresToDisk(); err != nil {
			logrus.Warnf("Failed to write the stores to disk. Error: %q", err)
		}

		logrus.Debugf("Creating a new plan.")
		p = api.CreatePlan(flags.Srcpath, flags.ConfigurationsPath, flags.Name)
	} else {
		logrus.Infof("Detected a plan file at path %s. Will transform using this plan.", flags.Planfile)
		if p, err = plan.ReadPlan(flags.Planfile); err != nil {
			logrus.Fatalf("Unable to read the plan at path %s Error: %q", flags.Planfile, err)
		}
		if len(p.Spec.Services) == 0 {
			logrus.Debugf("Plan : %+v", p)
			logrus.Fatalf("Failed to find any services. Aborting.")
		}
		if cmd.Flags().Changed(cmdcommon.NameFlag) {
			p.Name = flags.Name
		}
		if cmd.Flags().Changed(cmdcommon.SourceFlag) {
			if err := p.SetRootDir(flags.Srcpath); err != nil {
				logrus.Fatalf("Failed to set the root directory to %q Error: %q", flags.Srcpath, err)
			}
		}
		if cmd.Flags().Changed(cmdcommon.ConfigurationsFlag) {
			if flags.ConfigurationsPath != "" {
				p.Spec.ConfigurationsDir = flags.ConfigurationsPath
			}
		}

		// Global settings
		cmdcommon.CheckSourcePath(p.Spec.RootDir)
		common.CheckAndCopyConfigurations(p.Spec.ConfigurationsDir)
		flags.Outpath = filepath.Join(flags.Outpath, p.Name)
		cmdcommon.CheckOutputPath(flags.Outpath, flags.Overwrite)
		if p.Spec.RootDir == flags.Outpath || common.IsParent(flags.Outpath, p.Spec.RootDir) || common.IsParent(p.Spec.RootDir, flags.Outpath) {
			logrus.Fatalf("The source path %s and output path %s overlap.", p.Spec.RootDir, flags.Outpath)
		}
		if err := os.MkdirAll(flags.Outpath, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.Outpath, err)
		}
		qaengine.StartEngine(flags.Qaskip, flags.qaport, flags.qadisablecli)
		qaengine.SetupConfigFile(filepath.Join(flags.Outpath, common.ConfigFile), flags.Setconfigs, flags.Configs, flags.PreSets)
		qaengine.SetupWriteCacheFile(filepath.Join(flags.Outpath, common.QACacheFile))
		if err := qaengine.WriteStoresToDisk(); err != nil {
			logrus.Warnf("Failed to write the stores to disk. Error: %q", err)
		}
	}
	p = api.CuratePlan(p)
	api.Transform(p, flags.Outpath)
	logrus.Infof("Transformed target artifacts can be found at [%s].", flags.Outpath)
}

func getTransformCommand() *cobra.Command {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := transformFlags{}
	transformCmd := &cobra.Command{
		Use:   "transform",
		Short: "Transform using move2kube plan",
		Long:  "Transform artifacts using move2kube plan",
		Run:   func(cmd *cobra.Command, _ []string) { transformHandler(cmd, flags) },
	}

	// Basic options
	transformCmd.Flags().StringVarP(&flags.Planfile, cmdcommon.PlanFlag, "p", common.DefaultPlanFile, "Specify a plan file to execute.")
	transformCmd.Flags().BoolVar(&flags.Overwrite, cmdcommon.OverwriteFlag, false, "Overwrite the output directory if it exists. By default we don't overwrite.")
	transformCmd.Flags().StringVarP(&flags.Srcpath, cmdcommon.SourceFlag, "s", "", "Specify source directory to transform. If you already have a m2k.plan then this will override the rootdir value specified in that plan.")
	transformCmd.Flags().StringVarP(&flags.Outpath, cmdcommon.OutputFlag, "o", ".", "Path for output. Default will be directory with the project name.")
	transformCmd.Flags().StringVarP(&flags.Name, cmdcommon.NameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	transformCmd.Flags().StringSliceVarP(&flags.Configs, cmdcommon.ConfigFlag, "f", []string{}, "Specify config file locations")
	transformCmd.Flags().StringSliceVarP(&flags.PreSets, cmdcommon.PreSetFlag, "r", []string{}, "Specify preset config to use")
	transformCmd.Flags().StringArrayVarP(&flags.Setconfigs, cmdcommon.SetConfigFlag, "k", []string{}, "Specify config key-value pairs")
	transformCmd.Flags().StringVarP(&flags.ConfigurationsPath, cmdcommon.ConfigurationsFlag, "c", "", "Specify directory where configurations are stored.")

	// Advanced options
	transformCmd.Flags().BoolVar(&flags.IgnoreEnv, cmdcommon.IgnoreEnvFlag, false, "Ignore data from local machine.")

	// Hidden options
	transformCmd.Flags().BoolVar(&flags.qadisablecli, qadisablecliFlag, false, "Enable/disable the QA Cli sub-system. Without this system, you will have to use the REST API to interact.")
	transformCmd.Flags().BoolVar(&flags.Qaskip, cmdcommon.QASkipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")
	transformCmd.Flags().IntVar(&flags.qaport, qaportFlag, 0, "Port for the QA service. By default it chooses a random free port.")

	must(transformCmd.Flags().MarkHidden(qadisablecliFlag))
	must(transformCmd.Flags().MarkHidden(qaportFlag))

	return transformCmd
}
