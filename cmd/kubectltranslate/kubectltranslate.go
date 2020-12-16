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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cmdcommon "github.com/konveyor/move2kube/cmd/common"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/internal/move2kube"
	"github.com/konveyor/move2kube/internal/qaengine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type translateFlags struct {
	outpath  string
	srcpath  string
	name     string
	qaskip   bool
	qacaches []string
}

var verbose bool

func translateHandler(cmd *cobra.Command, flags translateFlags) {
	// Setup
	var err error
	srcpath := flags.srcpath
	outpath := flags.outpath
	name := flags.name
	qacaches := flags.qacaches
	qaskip := flags.qaskip

	// These are just the defaults used in the move2kube translate command
	qadisablecli := false // kubectl-translate does not support REST based access.
	ignoreEnv := false    // Since kubectl is always supposed to run in the local machine, it will always use environment related info
	qaport := 0           // setting 0, since kubectl-translate does not support REST API based access this value never gets used

	if srcpath, err = filepath.Abs(srcpath); err != nil {
		log.Fatalf("Failed to make the source directory path %q absolute. Error: %q", srcpath, err)
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

	// Plan
	cmdcommon.CheckSourcePath(srcpath)
	plan := move2kube.CreatePlan(srcpath, name)
	outpath = filepath.Join(outpath, plan.Name)
	cmdcommon.CheckOutputPath(outpath)
	cmdcommon.CreateOutputDirectoryAndCacheFile(outpath)
	plan = move2kube.CuratePlan(plan)

	// Translate
	move2kube.Translate(plan, outpath, qadisablecli)
	log.Infof("Translated target artifacts can be found at [%s].", outpath)
}

func main() {
	// Setup
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	commandUsedToInvoke := "kubectl-translate"
	if strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-") {
		commandUsedToInvoke = "kubectl translate"
	}

	flags := translateFlags{}
	translateCmd := &cobra.Command{
		Use:   fmt.Sprintf("%s [ -o <output directory> ] [ -n <project name> ] [ -q <list of qacache files> ] -s <source directory>", commandUsedToInvoke),
		Short: "Translate containerizes your app and creates k8s resources to get your app running on k8s.",
		Long: `Translate containerizes your app and creates k8s resources to get your app running on k8s.
It supports translating docker-compose, docker swarm, and cloud foundry apps.
Even if the app does not use any of the above, or even if it is not containerized
it can still be translated.

This plugin is a small feature of a more flexible CLI tool called Move2Kube https://github.com/konveyor/move2kube
For more documentation and support for this plugin and Move2Kube, visit https://konveyor.io/move2kube
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
	translateCmd.Flags().StringVarP(&flags.srcpath, cmdcommon.SourceFlag, "s", ".", "Specify source directory to translate. If you already have a m2k.plan then this will override the rootdir value specified in that plan.")
	translateCmd.Flags().StringVarP(&flags.outpath, cmdcommon.OutputFlag, "o", ".", "Path for output. Default will be directory with the project name.")
	translateCmd.Flags().StringVarP(&flags.name, cmdcommon.NameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	translateCmd.Flags().StringSliceVarP(&flags.qacaches, cmdcommon.QACacheFlag, "q", []string{}, "Specify qa cache file locations")
	translateCmd.Flags().BoolVar(&flags.qaskip, cmdcommon.QASkipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")

	must(translateCmd.MarkFlagRequired(cmdcommon.SourceFlag))

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
