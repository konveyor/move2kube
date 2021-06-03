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
	"path/filepath"

	cmdcommon "github.com/konveyor/move2kube/cmd/common"
	"github.com/konveyor/move2kube/internal/newparameterizer"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type parameterizeFlags struct {
	// Outpath contains the path to the output folder
	Outpath string
	// SourceFlag contains path to the source folder
	Srcpath string
	// ParameterizerPaths contains a list of paths to paramterizer yamls or folders containing such yamls
	ParameterizerPaths []string
}

const (
	parameterizersFlag = "parameterizers"
)

func parameterizeHandler(_ *cobra.Command, flags parameterizeFlags) {
	var err error
	if flags.Srcpath, err = filepath.Abs(flags.Srcpath); err != nil {
		log.Fatalf("Failed to make the source directory path %q absolute. Error: %q", flags.Srcpath, err)
	}
	if flags.Outpath, err = filepath.Abs(flags.Outpath); err != nil {
		log.Fatalf("Failed to make the output directory path %q absolute. Error: %q", flags.Outpath, err)
	}
	normalizedParameterizerPaths, err := cmdcommon.NormalizePaths(flags.ParameterizerPaths, []string{".yaml"})
	if err != nil {
		log.Fatalf("Failed to clean the paths:\n%+v\nError: %q", flags.ParameterizerPaths, err)
	}
	filesWritten, err := newparameterizer.ParameterizeAllPaths(normalizedParameterizerPaths, flags.Srcpath, flags.Outpath)
	if err != nil {
		log.Fatalf("Failed to apply all the parameterizations. Error: %q", err)
	}
	log.Debugf("filesWritten: %+v", filesWritten)
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
	parameterizeCmd.Flags().StringVarP(&flags.Srcpath, cmdcommon.SourceFlag, "s", "", "Specify source directory to parameterize.")
	parameterizeCmd.Flags().StringVarP(&flags.Outpath, cmdcommon.OutputFlag, "o", "", "Output directory path.")
	parameterizeCmd.Flags().StringSliceVarP(&flags.ParameterizerPaths, parameterizersFlag, "p", []string{}, "Specify paths to the paramterizer yamls. Can be the path to a yaml file or the path to a folder containing the yamls.")

	must(parameterizeCmd.MarkFlagRequired(cmdcommon.SourceFlag))
	must(parameterizeCmd.MarkFlagRequired(cmdcommon.OutputFlag))
	must(parameterizeCmd.MarkFlagRequired(parameterizersFlag))

	return parameterizeCmd
}
