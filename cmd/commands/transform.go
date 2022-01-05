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

package commands

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/lib"
	"github.com/konveyor/move2kube/types/plan"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type transformFlags struct {
	qaflags
	// ignoreEnv tells us whether to use data collected from the local machine
	ignoreEnv bool
	// disableLocalExecution disables execution of executables locally
	disableLocalExecution bool
	// planfile is contains the path to the plan file
	planfile string
	// outpath contains the path to the output folder
	outpath string
	// SourceFlag contains path to the source folder
	srcpath string
	// name contains the project name
	name string
	// overwrite lets you overwrite the output directory if it exists
	overwrite bool
	// CustomizationsPaths contains the path to the customizations directory
	customizationsPath  string
	transformerSelector string
}

func transformHandler(cmd *cobra.Command, flags transformFlags) {
	ctx, cancel := context.WithCancel(cmd.Context())
	logrus.AddHook(common.NewCleanupHook(cancel))
	logrus.AddHook(common.NewCleanupHook(lib.Destroy))
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	go func() {
		<-ctx.Done()
		lib.Destroy()
		stop()
		common.Interrupt()
	}()
	defer lib.Destroy()

	var err error
	if flags.planfile, err = filepath.Abs(flags.planfile); err != nil {
		logrus.Fatalf("Failed to make the plan file path %q absolute. Error: %q", flags.planfile, err)
	}
	if flags.srcpath != "" {
		if flags.srcpath, err = filepath.Abs(flags.srcpath); err != nil {
			logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.srcpath, err)
		}
	}
	if flags.outpath, err = filepath.Abs(flags.outpath); err != nil {
		logrus.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.outpath, err)
	}

	// Global settings
	common.IgnoreEnvironment = flags.ignoreEnv
	common.DisableLocalExecution = flags.disableLocalExecution
	// Global settings

	// Parameter cleaning and curate plan
	var p plan.Plan
	fi, err := os.Stat(flags.planfile)
	if err == nil && fi.IsDir() {
		flags.planfile = filepath.Join(flags.planfile, common.DefaultPlanFile)
		_, err = os.Stat(flags.planfile)
	}
	if err != nil {
		logrus.Debugf("No plan file found.")
		if cmd.Flags().Changed(planFlag) {
			logrus.Fatalf("Error while accessing plan file at path %s Error: %q", flags.planfile, err)
		}
		if !cmd.Flags().Changed(sourceFlag) {
			logrus.Fatalf("Invalid usage. Must specify either path to a plan file or path to directory containing source code.")
		}

		// Global settings
		checkSourcePath(flags.srcpath)
		flags.outpath = filepath.Join(flags.outpath, flags.name)
		checkOutputPath(flags.outpath, flags.overwrite)
		if flags.srcpath == flags.outpath || common.IsParent(flags.outpath, flags.srcpath) || common.IsParent(flags.srcpath, flags.outpath) {
			logrus.Fatalf("The source path %s and output path %s overlap.", flags.srcpath, flags.outpath)
		}
		if err := os.MkdirAll(flags.outpath, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.outpath, err)
		}
		startQA(flags.qaflags)
		logrus.Debugf("Creating a new plan.")
		p = lib.CreatePlan(ctx, flags.srcpath, flags.outpath, flags.customizationsPath, flags.transformerSelector, flags.name)
	} else {
		logrus.Infof("Detected a plan file at path %s. Will transform using this plan.", flags.planfile)
		rootDir := ""
		if cmd.Flags().Changed(sourceFlag) {
			rootDir = flags.srcpath
		}
		if p, err = plan.ReadPlan(flags.planfile, rootDir); err != nil {
			logrus.Fatalf("Unable to read the plan at path %s Error: %q", flags.planfile, err)
		}
		if len(p.Spec.Services) == 0 {
			logrus.Debugf("Plan : %+v", p)
			logrus.Fatalf("Failed to find any services. Aborting.")
		}
		if cmd.Flags().Changed(nameFlag) {
			p.Name = flags.name
		}
		if cmd.Flags().Changed(customizationsFlag) {
			if flags.customizationsPath != "" {
				p.Spec.CustomizationsDir = flags.customizationsPath
			}
		}

		// Global settings
		checkSourcePath(p.Spec.SourceDir)
		lib.CheckAndCopyCustomizations(p.Spec.CustomizationsDir)
		flags.outpath = filepath.Join(flags.outpath, p.Name)
		checkOutputPath(flags.outpath, flags.overwrite)
		if p.Spec.SourceDir == flags.outpath || common.IsParent(flags.outpath, p.Spec.SourceDir) || common.IsParent(p.Spec.SourceDir, flags.outpath) {
			logrus.Fatalf("The source path %s and output path %s overlap.", p.Spec.SourceDir, flags.outpath)
		}
		if err := os.MkdirAll(flags.outpath, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.outpath, err)
		}
		startQA(flags.qaflags)
	}
	p = lib.CuratePlan(p, flags.outpath, flags.transformerSelector)
	lib.Transform(ctx, p, flags.outpath)
	logrus.Infof("Transformed target artifacts can be found at [%s].", flags.outpath)
}

// GetTransformCommand returns a command to do the transformation
func GetTransformCommand() *cobra.Command {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := transformFlags{}
	transformCmd := &cobra.Command{
		Use:        "transform",
		Short:      "Transform using move2kube plan",
		Long:       "Transform artifacts using move2kube plan",
		Run:        func(cmd *cobra.Command, _ []string) { transformHandler(cmd, flags) },
		SuggestFor: []string{"translate"},
	}

	// Basic options
	transformCmd.Flags().StringVarP(&flags.planfile, planFlag, "p", common.DefaultPlanFile, "Specify a plan file to execute.")
	transformCmd.Flags().BoolVar(&flags.overwrite, overwriteFlag, false, "Overwrite the output directory if it exists. By default we don't overwrite.")
	transformCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify source directory to transform. If you already have a m2k.plan then this will override the rootdir value specified in that plan.")
	transformCmd.Flags().StringVarP(&flags.outpath, outputFlag, "o", ".", "Path for output. Default will be directory with the project name.")
	transformCmd.Flags().StringVarP(&flags.name, nameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	transformCmd.Flags().StringVar(&flags.configOut, configOutFlag, ".", "Specify config file output location.")
	transformCmd.Flags().StringVar(&flags.qaCacheOut, qaCacheOutFlag, ".", "Specify cache file output location.")
	transformCmd.Flags().StringSliceVarP(&flags.configs, configFlag, "f", []string{}, "Specify config file locations.")
	transformCmd.Flags().StringSliceVar(&flags.preSets, preSetFlag, []string{}, "Specify preset config to use.")
	transformCmd.Flags().StringArrayVar(&flags.setconfigs, setConfigFlag, []string{}, "Specify config key-value pairs.")
	transformCmd.Flags().StringVarP(&flags.customizationsPath, customizationsFlag, "c", "", "Specify directory where customizations are stored.")
	transformCmd.Flags().StringVarP(&flags.transformerSelector, transformerSelectorFlag, "t", "", "Specify the transformer selector.")
	transformCmd.Flags().BoolVar(&flags.qaskip, qaSkipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")

	// Advanced options
	transformCmd.Flags().BoolVar(&flags.ignoreEnv, ignoreEnvFlag, false, "Ignore data from local machine.")
	transformCmd.Flags().BoolVar(&flags.disableLocalExecution, common.DisableLocalExecutionFlag, false, "Allow files to be executed locally.")

	// Hidden options
	transformCmd.Flags().BoolVar(&flags.qadisablecli, qadisablecliFlag, false, "Enable/disable the QA Cli sub-system. Without this system, you will have to use the REST API to interact.")
	transformCmd.Flags().IntVar(&flags.qaport, qaportFlag, 0, "Port for the QA service. By default it chooses a random free port.")

	must(transformCmd.Flags().MarkHidden(qadisablecliFlag))
	must(transformCmd.Flags().MarkHidden(qaportFlag))

	return transformCmd
}
