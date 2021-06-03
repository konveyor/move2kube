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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var verbose bool

func translateHandler(cmd *cobra.Command, flags cmdcommon.TranslateFlags) {
	// Setup
	var err error

	// These are just the defaults used in the move2kube translate command
	qadisablecli := false // kubectl-translate does not support REST based access.
	ignoreEnv := false    // Since kubectl is always supposed to run in the local machine, it will always use environment related info
	qaport := 0           // setting 0, since kubectl-translate does not support REST API based access this value never gets used

	if flags.Srcpath, err = filepath.Abs(flags.Srcpath); err != nil {
		log.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.Srcpath, err)
	}
	if flags.Outpath, err = filepath.Abs(flags.Outpath); err != nil {
		log.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.Outpath, err)
	}

	// Global settings
	common.IgnoreEnvironment = ignoreEnv
	cmdcommon.CheckSourcePath(flags.Srcpath)
	flags.Outpath = filepath.Join(flags.Outpath, flags.Name)
	cmdcommon.CheckOutputPath(flags.Outpath, flags.Overwrite)
	if flags.Srcpath == flags.Outpath || common.IsParent(flags.Outpath, flags.Srcpath) || common.IsParent(flags.Srcpath, flags.Outpath) {
		log.Fatalf("The source path %s and output path %s overlap.", flags.Srcpath, flags.Outpath)
	}
	if err := os.MkdirAll(flags.Outpath, common.DefaultDirectoryPermission); err != nil {
		log.Fatalf("Failed to create the output directory at path %s Error: %q", flags.Outpath, err)
	}
	qaengine.StartEngine(flags.Qaskip, qaport, qadisablecli)
	qaengine.SetupConfigFile(flags.Outpath, flags.Setconfigs, flags.Configs, flags.PreSets)
	qaengine.SetupCacheFile(flags.Outpath, flags.Qacaches)
	if err := qaengine.WriteStoresToDisk(); err != nil {
		log.Warnf("Failed to write the stores to disk. Error: %q", err)
	}

	// Plan
	plan := move2kube.CreatePlan(flags.Srcpath, flags.Name, true)
	plan = move2kube.CuratePlan(plan)

	// Translate
	normalizedTransformPaths, err := cmdcommon.NormalizePaths(flags.TransformPaths, []string{".star"})
	if err != nil {
		log.Fatalf("Failed to clean the paths:\n%+v\nError: %q", flags.TransformPaths, err)
	}
	move2kube.Translate(plan, flags.Outpath, qadisablecli, normalizedTransformPaths)
	log.Infof("Translated target artifacts can be found at [%s].", flags.Outpath)
}

func main() {
	// Setup
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := cmdcommon.TranslateFlags{}
	translateCmd := &cobra.Command{
		Use:   "kubectl translate [ -o <output directory> ] [ -n <project name> ] [ -q <list of qacache files> ] -s <source directory>",
		Short: "Translate creates all the resources required for deploying your application into kubernetes, including containerisation and kubernetes resources.",
		Long: `Translate creates all resources required for deploying your application into kubernetes, including containerisation and kubernetes resources.
It supports translating from docker swarm/docker-compose, cloud foundry apps and even other non-containerized applications.
Even if the app does not use any of the above, or even if it is not containerized it can still be translated.

This plugin is a small feature of a more flexible CLI tool called Move2Kube https://github.com/konveyor/move2kube
For more documentation and support for this plugin and Move2Kube, visit https://move2kube.konveyor.io/
`,
		Run: func(cmd *cobra.Command, _ []string) {
			if verbose {
				log.SetLevel(log.DebugLevel)
			}
			translateHandler(cmd, flags)
		},
	}

	// Options inherited from move2kube root command
	translateCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Basic options
	translateCmd.Flags().StringVarP(&flags.Srcpath, cmdcommon.SourceFlag, "s", ".", "Specify source directory to translate. If you already have a m2k.plan then this will override the rootdir value specified in that plan.")
	translateCmd.Flags().StringVarP(&flags.Outpath, cmdcommon.OutputFlag, "o", ".", "Path for output. Default will be directory with the project name.")
	translateCmd.Flags().StringVarP(&flags.Name, cmdcommon.NameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	translateCmd.Flags().StringSliceVarP(&flags.Qacaches, cmdcommon.QACacheFlag, "q", []string{}, "Specify qa cache file locations")
	translateCmd.Flags().StringSliceVarP(&flags.Configs, cmdcommon.ConfigFlag, "f", []string{}, "Specify config file locations")
	translateCmd.Flags().StringSliceVarP(&flags.PreSets, cmdcommon.PreSetFlag, "r", []string{}, "Specify preset config to use")
	translateCmd.Flags().BoolVar(&flags.Qaskip, cmdcommon.QASkipFlag, false, "Enable/disable the default answers to questions posed in QA sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")
	translateCmd.Flags().BoolVarP(&flags.Overwrite, cmdcommon.OverwriteFlag, "", false, "Overwrite the output directory if it exists. By default we don't overwrite.")
	translateCmd.Flags().StringArrayVarP(&flags.Setconfigs, cmdcommon.SetConfigFlag, "k", []string{}, "Specify config key-value pairs")
	translateCmd.Flags().StringSliceVarP(&flags.TransformPaths, cmdcommon.TransformsFlag, "t", []string{}, "Specify paths to the transformation scripts to apply. Can be the path to a script or the path to a folder containing the scripts.")

	must(translateCmd.MarkFlagRequired(cmdcommon.SourceFlag))

	translateCmd.AddCommand(cmdcommon.GetVersionCommand())

	// Run
	assetsPath, tempPath, err := common.CreateAssetsData()
	if err != nil {
		log.Fatalf("Unable to create the assets directory. Error: %q", err)
	}
	common.TempPath = tempPath
	common.AssetsPath = assetsPath
	defer os.RemoveAll(tempPath)
	if err := translateCmd.Execute(); err != nil {
		log.Fatalf("Error: %q", err)
	}
}
