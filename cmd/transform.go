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

package cmd

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/konveyor/move2kube-wasm/common/download"
	//"github.com/konveyor/move2kube-wasm/common/vcs"
	"github.com/konveyor/move2kube-wasm/lib"
	"github.com/konveyor/move2kube-wasm/types/plan"
	"github.com/mholt/archiver/v3"
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
	// profilepath contains the path to the CPU profile file
	profilepath string
	// outpath contains the path to the output folder
	outpath string
	// SourceFlag contains path to the source folder
	srcpath string
	// name contains the project name
	name string
	// overwrite lets you overwrite the output directory if it exists
	overwrite bool
	// maxIterations is the maximum number of iterations to allow before aborting with an error
	maxIterations int
	// CustomizationsPaths contains the path to the customizations directory
	customizationsPath  string
	transformerSelector string
}

func transformHandler(cmd *cobra.Command, flags transformFlags) {
	if flags.profilepath != "" {
		if f, err := os.Create(flags.profilepath); err != nil {
			panic(err)
		} else if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
	}

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
	//TODO: WASI
	//isRemotePath := vcs.IsRemotePath(flags.srcpath)
	//if flags.srcpath != "" && !isRemotePath {
	if flags.srcpath != "" {
		if flags.srcpath, err = filepath.Abs(flags.srcpath); err != nil {
			logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.srcpath, err)
		}
	}
	//isRemoteOutPath := vcs.IsRemotePath(flags.outpath)
	//if !isRemoteOutPath {
	if flags.outpath, err = filepath.Abs(flags.outpath); err != nil {
		logrus.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.outpath, err)
	}
	//}
	// Check if the default customization folder exists in the working directory.
	// If not, skip the customization option
	if !cmd.Flags().Changed(customizationsFlag) {
		if _, err := os.Stat(common.DefaultCustomizationDir); err == nil {
			flags.customizationsPath = common.DefaultCustomizationDir
			// make all path(s) absolute
			flags.customizationsPath, err = filepath.Abs(flags.customizationsPath)
			if err != nil {
				logrus.Fatalf("Failed to make the customizations directory path %q absolute. Error: %q", flags.customizationsPath, err)
			}
		}
	}
	// Check if the default configuration file exists in the working directory.
	// If not, skip the configuration option
	if !cmd.Flags().Changed(configFlag) {
		if _, err := os.Stat(common.DefaultConfigFilePath); err == nil {
			flags.configs = []string{common.DefaultConfigFilePath}
		}
	}
	// make all path(s) absolute
	for i, c := range flags.configs {
		if !download.IsRemotePath(c) {
			if c, err := filepath.Abs(c); err != nil {
				logrus.Fatalf("failed to make the config file path %s absolute. Error: %q", c, err)
			}
			flags.configs[i] = c
		}
	}

	// Global settings
	common.IgnoreEnvironment = flags.ignoreEnv
	common.DisableLocalExecution = flags.disableLocalExecution
	// Global settings

	// Parameter cleaning and curate plan
	transformationPlan := plan.Plan{}
	preExistingPlan := false
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

		// Global settings
		//TODO: WASI
		//if !isRemoteOutPath {
		flags.outpath = filepath.Join(flags.outpath, flags.name)
		checkOutputPath(flags.outpath, flags.overwrite)
		//if flags.srcpath != "" && !isRemotePath {
		if flags.srcpath != "" {
			checkSourcePath(flags.srcpath)
			if flags.srcpath == flags.outpath || common.IsParent(flags.outpath, flags.srcpath) || common.IsParent(flags.srcpath, flags.outpath) {
				logrus.Fatalf("The source path %s and output path %s overlap.", flags.srcpath, flags.outpath)
			}
		}
		if err := os.MkdirAll(flags.outpath, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.outpath, err)
		}
		//}
		startQA(flags.qaflags)
		logrus.Debugf("Creating a new plan.")
		transformationPlan, err = lib.CreatePlan(ctx, flags.srcpath, flags.outpath, flags.customizationsPath, flags.transformerSelector, flags.name)
		if err != nil {
			logrus.Fatalf("failed to create the plan. Error: %q", err)
		}
		if len(transformationPlan.Spec.Services) == 0 && len(transformationPlan.Spec.InvokedByDefaultTransformers) == 0 {
			logrus.Debugf("Plan : %+v", transformationPlan)
			logrus.Fatalf("failed to find any services or default transformers. Aborting.")
		}
	} else {
		preExistingPlan = true
		logrus.Infof("Detected a plan file at path %s. Will transform using this plan.", flags.planfile)
		sourceDir := ""
		if cmd.Flags().Changed(sourceFlag) {
			sourceDir = flags.srcpath
			logrus.Warnf("Using the detected plan with specified source. If you did not want to use the plan file at %s, delete it and rerun the command.", flags.planfile)
		}
		if transformationPlan, err = plan.ReadPlan(flags.planfile, sourceDir); err != nil {
			logrus.Fatalf("Unable to read the plan at path %s Error: %q", flags.planfile, err)
		}
		if len(transformationPlan.Spec.Services) == 0 && len(transformationPlan.Spec.InvokedByDefaultTransformers) == 0 {
			logrus.Debugf("Plan : %+v", transformationPlan)
			logrus.Fatalf("Failed to find any services or default transformers. Aborting.")
		}
		if cmd.Flags().Changed(nameFlag) {
			transformationPlan.Name = flags.name
		}
		if cmd.Flags().Changed(customizationsFlag) {
			if flags.customizationsPath != "" {
				transformationPlan.Spec.CustomizationsDir = flags.customizationsPath
				logrus.Warnf("Using the detected plan with specified customization. This might result in undesired results if the customization is different from what was given to plan. If you did not want to use the plan file at %s, delete it and rerun the command.", flags.planfile)
			}
		}

		// Global settings
		//if transformationPlan.Spec.SourceDir != "" {
		//	checkSourcePath(transformationPlan.Spec.SourceDir)
		//}
		lib.CheckAndCopyCustomizations(transformationPlan.Spec.CustomizationsDir)

		//TODO: WASI
		//if !isRemoteOutPath {
		flags.outpath = filepath.Join(flags.outpath, transformationPlan.Name)
		checkOutputPath(flags.outpath, flags.overwrite)
		if transformationPlan.Spec.SourceDir != "" && (transformationPlan.Spec.SourceDir == flags.outpath || common.IsParent(flags.outpath, transformationPlan.Spec.SourceDir) || common.IsParent(transformationPlan.Spec.SourceDir, flags.outpath)) {
			logrus.Fatalf("The source path %s and output path %s overlap.", transformationPlan.Spec.SourceDir, flags.outpath)
		}
		if err := os.MkdirAll(flags.outpath, common.DefaultDirectoryPermission); err != nil {
			logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.outpath, err)
		}
		//}
		startQA(flags.qaflags)
	}
	if err := lib.Transform(ctx, transformationPlan, preExistingPlan, flags.outpath, flags.transformerSelector, flags.maxIterations); err != nil {
		logrus.Fatalf("failed to transform. Error: %q", err)
	}
	logrus.Infof("Transformed target artifacts can be found at [%s].", flags.outpath)
	{
		if err := archiver.Archive([]string{"myproject"}, "myproject.zip"); err != nil {
			logrus.Fatalf("Cannot archive myproject dir. Error: %q", err)
		}
	}
	{
		logrus.Infof("DEBUG after planning, list files")
		fs, err := os.ReadDir("myproject/source/language-platforms/")
		if err != nil {
			panic(err)
		}
		for i, f := range fs {
			logrus.Infof("DEBUG file[%d] %+v", i, f)
		}
	}
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
	transformCmd.Flags().StringVar(&flags.profilepath, profileFlag, "", "Path where the CPU profile file should be generated. By default we don't profile.")
	transformCmd.Flags().StringVarP(&flags.planfile, planFlag, "p", common.DefaultPlanFile, "Specify a plan file to execute.")
	transformCmd.Flags().BoolVar(&flags.overwrite, overwriteFlag, false, "Overwrite the output directory if it exists. By default we don't overwrite.")
	transformCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify source directory or a git url (see https://move2kube.konveyor.io/concepts/git-support) to transform. If you already have a m2k.plan then this will override the sourceDir value specified in that plan.")
	transformCmd.Flags().StringVarP(&flags.outpath, outputFlag, "o", ".", "Path for output or a git url (see https://move2kube.konveyor.io/concepts/git-support). Default will be directory with the project name.")
	transformCmd.Flags().StringVarP(&flags.name, nameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	transformCmd.Flags().StringVar(&flags.configOut, configOutFlag, ".", "Specify config file output location.")
	transformCmd.Flags().StringVar(&flags.qaCacheOut, qaCacheOutFlag, ".", "Specify cache file output location.")
	transformCmd.Flags().StringSliceVarP(&flags.configs, configFlag, "f", []string{}, "Specify config file locations. By default we look for "+common.DefaultConfigFilePath)
	transformCmd.Flags().StringSliceVar(&flags.preSets, preSetFlag, []string{}, "Specify preset config to use.")
	transformCmd.Flags().BoolVar(&flags.persistPasswords, qaPersistPasswords, false, "Store passwords in the config and cache. By default passwords are not persisted.")
	transformCmd.Flags().StringArrayVar(&flags.setconfigs, setConfigFlag, []string{}, "Specify config key-value pairs.")
	transformCmd.Flags().StringVarP(&flags.customizationsPath, customizationsFlag, "c", "", "Specify directory or a git url (see https://move2kube.konveyor.io/concepts/git-support) where customizations are stored. By default we look for "+common.DefaultCustomizationDir)
	transformCmd.Flags().StringVarP(&flags.transformerSelector, transformerSelectorFlag, "t", "", "Specify the transformer selector.")
	transformCmd.Flags().BoolVar(&flags.qaskip, qaSkipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")

	// Advanced options
	transformCmd.Flags().BoolVar(&flags.ignoreEnv, ignoreEnvFlag, false, "Ignore data from local machine.")
	transformCmd.Flags().BoolVar(&flags.disableLocalExecution, common.DisableLocalExecutionFlag, false, "Allow files to be executed locally.")
	transformCmd.Flags().IntVar(&flags.maxIterations, maxIterationsFlag, -1, "The maximum number of iterations to allow. Negative value means infinite. Default is -1.")

	// Hidden options
	transformCmd.Flags().BoolVar(&flags.qadisablecli, qadisablecliFlag, false, "Enable/disable the QA Cli sub-system. Without this system, you will have to use the REST API to interact.")
	transformCmd.Flags().IntVar(&flags.qaport, qaportFlag, 0, "Port for the QA service. By default it chooses a random free port.")

	must(transformCmd.Flags().MarkHidden(qadisablecliFlag))
	must(transformCmd.Flags().MarkHidden(qaportFlag))

	return transformCmd
}
