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
	"strings"

	"github.com/konveyor/move2kube/internal/move2kube"
	"github.com/konveyor/move2kube/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type collectFlags struct {
	annotations string
	outpath     string
	srcpath     string
}

const (
	annotationsFlag = "annotations"
)

func collectHandler(flags collectFlags) {
	var err error
	annotations := flags.annotations
	outpath := flags.outpath
	srcpath := flags.srcpath

	if outpath != "" {
		if outpath, err = filepath.Abs(outpath); err != nil {
			log.Fatalf("Failed to make the output directory path %q absolute. Error: %q", outpath, err)
		}
	}
	if srcpath != "" {
		srcpath, err = filepath.Abs(srcpath)
		if err != nil {
			log.Fatalf("Failed to make the source directory path %q absolute. Error: %q", srcpath, err)
		}
		fi, err := os.Stat(srcpath)
		if os.IsNotExist(err) {
			log.Fatalf("Source directory does not exist: %s.", err)
		} else if err != nil {
			log.Fatalf("Error while accessing directory: %s. ", srcpath)
		} else if !fi.IsDir() {
			log.Fatalf("Source path is a file, expected directory: %s.", srcpath)
		}
	}
	outpath = filepath.Join(filepath.Clean(outpath), types.AppNameShort+"_collect")
	if annotations == "" {
		move2kube.Collect(srcpath, outpath, []string{})
	} else {
		move2kube.Collect(srcpath, outpath, strings.Split(annotations, ","))
	}
	log.Infof("Collect Output in [%s]. Copy this directory into the source directory to be used for planning.", outpath)
}

func init() {
	viper.AutomaticEnv()

	flags := collectFlags{}
	collectCmd := &cobra.Command{
		Use:   "collect",
		Short: "Collect and process metadata from multiple sources.",
		Long:  "Collect metadata from multiple sources (cluster, image repo etc.), filter and summarize it into a yaml.",
		Run:   func(*cobra.Command, []string) { collectHandler(flags) },
	}

	collectCmd.Flags().StringVarP(&flags.annotations, annotationsFlag, "a", "", "Specify annotations to select collector subset.")
	collectCmd.Flags().StringVarP(&flags.outpath, outpathFlag, "o", ".", "Specify output directory for collect.")
	collectCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify source directory for the artifacts to be considered while collecting.")

	rootCmd.AddCommand(collectCmd)
}
