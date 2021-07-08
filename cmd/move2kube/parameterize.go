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

	"github.com/konveyor/move2kube/api"
	"github.com/konveyor/move2kube/internal/common"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type parameterizeFlags struct {
	// outpath contains the path to the output folder
	outpath string
	// SourceFlag contains path to the source folder
	srcpath string
	// customizationsPath contains path to the pack folder
	customizationsPath string
	// overwrite: if the output folder exists then it will be overwritten
	overwrite bool
	qaflags
}

func parameterizeHandler(_ *cobra.Command, flags parameterizeFlags) {
	var err error
	if flags.srcpath, err = filepath.Abs(flags.srcpath); err != nil {
		logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.srcpath, err)
	}
	if flags.outpath, err = filepath.Abs(flags.outpath); err != nil {
		logrus.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.outpath, err)
	}
	if flags.customizationsPath, err = filepath.Abs(flags.customizationsPath); err != nil {
		logrus.Fatalf("Failed to make the pack directory path %q absolute. Error: %q", flags.customizationsPath, err)
	}

	checkSourcePath(flags.srcpath)
	checkOutputPath(flags.outpath, flags.overwrite)
	if flags.srcpath == flags.outpath || common.IsParent(flags.outpath, flags.srcpath) || common.IsParent(flags.srcpath, flags.outpath) {
		logrus.Fatalf("The source path %s and output path %s overlap.", flags.srcpath, flags.outpath)
	}
	if err := os.MkdirAll(flags.outpath, common.DefaultDirectoryPermission); err != nil {
		logrus.Fatalf("Failed to create the output directory at path %s Error: %q", flags.outpath, err)
	}
	startQA(flags.qaflags)

	// Parameterization
	filesWritten, err := api.Parameterize(flags.srcpath, flags.customizationsPath, flags.outpath)
	if err != nil {
		logrus.Fatalf("Failed to apply all the parameterizations. Error: %q", err)
	}
	logrus.Debugf("filesWritten: %+v", filesWritten)
	logrus.Infof("Parameterized artifacts can be found at [%s].", flags.outpath)
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
	parameterizeCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify the directory containing the source code to parameterize.")
	parameterizeCmd.Flags().StringVarP(&flags.outpath, outputFlag, "o", "", "Specify the directory where the output should be written.")
	parameterizeCmd.Flags().StringVarP(&flags.customizationsPath, customizationsFlag, "c", "", "Specify directory where customizations are stored.")
	parameterizeCmd.Flags().BoolVar(&flags.overwrite, overwriteFlag, false, "Overwrite the output directory if it exists. By default we don't overwrite.")
	parameterizeCmd.Flags().StringVar(&flags.configOut, configOutFlag, ".", "Specify config file output location")
	parameterizeCmd.Flags().StringVar(&flags.qaCacheOut, qaCacheOutFlag, ".", "Specify cache file output location")

	// Hidden options
	parameterizeCmd.Flags().BoolVar(&flags.qadisablecli, qadisablecliFlag, false, "Enable/disable the QA Cli sub-system. Without this system, you will have to use the REST API to interact.")
	parameterizeCmd.Flags().BoolVar(&flags.qaskip, qaSkipFlag, false, "Enable/disable the default answers to questions posed in QA Cli sub-system. If disabled, you will have to answer the questions posed by QA during interaction.")
	parameterizeCmd.Flags().IntVar(&flags.qaport, qaportFlag, 0, "Port for the QA service. By default it chooses a random free port.")

	must(parameterizeCmd.MarkFlagRequired(sourceFlag))
	must(parameterizeCmd.MarkFlagRequired(outputFlag))
	must(parameterizeCmd.MarkFlagRequired(customizationsFlag))

	must(parameterizeCmd.Flags().MarkHidden(qadisablecliFlag))
	must(parameterizeCmd.Flags().MarkHidden(qaportFlag))

	return parameterizeCmd
}
