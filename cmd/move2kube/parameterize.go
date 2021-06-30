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

package main

import (
	"os"
	"path/filepath"

	cmdcommon "github.com/konveyor/move2kube/cmd/common"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/konveyor/move2kube/parameterizer"
	"github.com/konveyor/move2kube/qaengine"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type parameterizeFlags struct {
	// Outpath contains the path to the output folder
	Outpath string
	// SourceFlag contains path to the source folder
	Srcpath string
	// ConfigurationsPath contains path to the pack folder
	ConfigurationsPath string
	// Overwrite: if the output folder exists then it will be overwritten
	Overwrite bool
	// qadisablecli: part of hidden flags, used to select http server engine for QA
	qadisablecli bool
	// qaskip: used to select the default engine for QA
	qaskip bool
	// qaport: part of hidden flags, used to select the port on which the http server engine listens
	qaport int
}

func parameterizeHandler(_ *cobra.Command, flags parameterizeFlags) {
	var err error
	if flags.Srcpath, err = filepath.Abs(flags.Srcpath); err != nil {
		logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.Srcpath, err)
	}
	if flags.Outpath, err = filepath.Abs(flags.Outpath); err != nil {
		logrus.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.Outpath, err)
	}
	if flags.ConfigurationsPath, err = filepath.Abs(flags.ConfigurationsPath); err != nil {
		logrus.Fatalf("Failed to make the pack directory path %q absolute. Error: %q", flags.ConfigurationsPath, err)
	}

	cmdcommon.CheckSourcePath(flags.Srcpath)
	cmdcommon.CheckOutputPath(flags.Outpath, flags.Overwrite)
	if flags.Srcpath == flags.Outpath || common.IsParent(flags.Outpath, flags.Srcpath) || common.IsParent(flags.Srcpath, flags.Outpath) {
		logrus.Fatalf("The source path %s and output path %s overlap.", flags.Srcpath, flags.Outpath)
	}
	if err := os.MkdirAll(flags.Outpath, common.DefaultDirectoryPermission); err != nil {
		logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.Outpath, err)
	}

	// Initialize the QA engine
	qaengine.StartEngine(flags.qaskip, flags.qaport, flags.qadisablecli)
	qaengine.SetupConfigFile(filepath.Join(flags.Outpath, common.ConfigFile), []string{}, []string{}, []string{})
	qaengine.SetupWriteCacheFile(filepath.Join(flags.Outpath, common.QACacheFile))
	if err := qaengine.WriteStoresToDisk(); err != nil {
		logrus.Warnf("Failed to write the config and/or cache to disk. Error: %q", err)
	}
	// Parameterization
	filesWritten, err := parameterizer.Top(flags.Srcpath, flags.ConfigurationsPath, flags.Outpath)
	if err != nil {
		logrus.Fatalf("Failed to apply all the parameterizations. Error: %q", err)
	}
	logrus.Debugf("filesWritten: %+v", filesWritten)
	logrus.Infof("Parameterized artifacts can be found at [%s].", flags.Outpath)
}

func getParameterizeCommand() *cobra.Command {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := parameterizeFlags{}
	parameterizeCmd := &cobra.Command{
		Use:   "parameterize",
		Short: "Parameterize fields in k8s resources",
		Long:  "Parameterize fields in k8s resources",
		Run:   func(cmd *cobra.Command, _ []string) { parameterizeHandler(cmd, flags) },
	}

	// Basic options
	parameterizeCmd.Flags().StringVarP(&flags.Srcpath, cmdcommon.SourceFlag, "s", "", "Specify the directory containing the source code to parameterize.")
	parameterizeCmd.Flags().StringVarP(&flags.Outpath, cmdcommon.OutputFlag, "o", "", "Specify the directory where the output should be written.")
	parameterizeCmd.Flags().StringVarP(&flags.ConfigurationsPath, cmdcommon.ConfigurationsFlag, "c", "", "Specify directory where configurations are stored.")
	parameterizeCmd.Flags().BoolVar(&flags.Overwrite, cmdcommon.OverwriteFlag, false, "Overwrite the output directory if it exists. By default we don't overwrite.")

	// Hidden options
	parameterizeCmd.Flags().BoolVar(&flags.qadisablecli, qadisablecliFlag, false, "Enable/disable the QA Cli sub-system. Without this system, you will have to use the REST API to interact.")
	parameterizeCmd.Flags().BoolVar(&flags.qaskip, cmdcommon.QASkipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")
	parameterizeCmd.Flags().IntVar(&flags.qaport, qaportFlag, 0, "Port for the QA service. By default it chooses a random free port.")

	must(parameterizeCmd.MarkFlagRequired(cmdcommon.SourceFlag))
	must(parameterizeCmd.MarkFlagRequired(cmdcommon.OutputFlag))
	must(parameterizeCmd.MarkFlagRequired(cmdcommon.ConfigurationsFlag))

	must(parameterizeCmd.Flags().MarkHidden(qadisablecliFlag))
	must(parameterizeCmd.Flags().MarkHidden(qaportFlag))

	return parameterizeCmd
}
