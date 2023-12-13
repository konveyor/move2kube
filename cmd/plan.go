/*
 *  Copyright IBM Corporation 2022
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
	"io/fs"
	"os"
	"os/signal"

	// "os/signal"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube-wasm/common"
	"github.com/mholt/archiver/v3"

	// "github.com/konveyor/move2kube/common/download"
	// "github.com/konveyor/move2kube/common/vcs"
	"github.com/konveyor/move2kube-wasm/lib"
	"github.com/konveyor/move2kube-wasm/qaengine"
	plantypes "github.com/konveyor/move2kube-wasm/types/plan"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type planFlags struct {
	progressServerPort    int
	planfile              string
	srcpath               string
	name                  string
	customizationsPath    string
	transformerSelector   string
	disableLocalExecution bool
	failOnEmptyPlan       bool
	//Configs contains a list of config files
	configs []string
	//Configs contains a list of key-value configs
	setconfigs []string
	//PreSets contains a list of preset configurations
	preSets []string
}

// func zip_helper(src, dst string) {
// 	// Open a zip archive for reading.
// 	r, err := zip.OpenReader(src)
// 	if err != nil {
// 		logrus.Fatalf("impossible to open zip reader: %s", err)
// 	}
// 	defer r.Close()

// 	// Iterate through the files in the archive,
// 	for k, f := range r.File {
// 		fmt.Printf("Unzipping %s:\n", f.Name)
// 		rc, err := f.Open()
// 		if err != nil {
// 			logrus.Fatalf("impossible to open file n°%d in archine: %s", k, err)
// 		}
// 		defer rc.Close()
// 		// define the new file path
// 		newFilePath := filepath.Join(dst, f.Name)

// 		// CASE 1 : we have a directory
// 		if f.FileInfo().IsDir() {
// 			// if we have a directory we have to create it
// 			err = os.MkdirAll(newFilePath, 0777)
// 			if err != nil {
// 				logrus.Fatalf("impossible to MkdirAll: %s", err)
// 			}
// 			// we can go to next iteration
// 			continue
// 		}

// 		// CASE 2 : we have a file
// 		// create new uncompressed file
// 		uncompressedFile, err := os.Create(newFilePath)
// 		if err != nil {
// 			logrus.Fatalf("impossible to create uncompressed: %s", err)
// 		}
// 		_, err = io.Copy(uncompressedFile, rc)
// 		if err != nil {
// 			logrus.Fatalf("impossible to copy file n°%d: %s", k, err)
// 		}
// 	}
// }

func planHandler(cmd *cobra.Command, flags planFlags) {
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
	planfile := flags.planfile
	srcpath := flags.srcpath
	name := flags.name
	//isRemotePath := vcs.IsRemotePath(srcpath)
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
		// if !download.IsRemotePath(c) {
		if c, err := filepath.Abs(c); err != nil {
			logrus.Fatalf("failed to make the config file path %s absolute. Error: %q", c, err)
		}
		flags.configs[i] = c
		// }
	}

	customizationsPath := flags.customizationsPath
	// Global settings
	common.DisableLocalExecution = flags.disableLocalExecution
	// Global settings

	planfile, err = filepath.Abs(planfile)
	if err != nil {
		logrus.Fatalf("Failed to make the plan file path %q absolute. Error: %q", planfile, err)
	}
	var fi fs.FileInfo
	// if srcpath != "" && !isRemotePath {
	if srcpath != "" {
		srcpath, err = filepath.Abs(srcpath)
		if err != nil {
			logrus.Fatalf("Failed to make the source directory path %q absolute. Error: %q", srcpath, err)
		}
		fi, err = os.Stat(srcpath)
		if err != nil {
			logrus.Fatalf("Unable to access source directory : %s", err)
		}
		if !fi.IsDir() {
			supported := false
			for _, ext := range supportedExtensions {
				if strings.HasSuffix(fi.Name(), ext) {
					// expand the archive
					archivePath := srcpath
					archiveExpandedPath := srcpath + "-expanded"
					if err := archiver.Unarchive(archivePath, archiveExpandedPath); err != nil {
						logrus.Fatalf("failed to expand the archive at path %s into path %s . Trying other formats. Error: %q", archivePath, archiveExpandedPath, err)
					}
					srcpath = archiveExpandedPath
					logrus.Infof("using '%s' as the source directory", srcpath)
					supported = true
					if err := os.WriteFile(filepath.Join(archiveExpandedPath, ".m2kignore"), []byte("."), common.DefaultFilePermission); err != nil {
						logrus.Fatalf("Could not write .m2kignore file. Error: %q", err)
					}
					break
				}
			}
			if !supported {
				logrus.Fatalf("The input path '%s' is a file, expected a directory", srcpath)
			}
		}
	}
	//{
	//	logrus.Infof("DEBUG before os.Stat on plan file")
	//	fs, err := os.ReadDir(".")
	//	if err != nil {
	//		panic(err)
	//	}
	//	for i, f := range fs {
	//		logrus.Infof("DEBUG file[%d] %+v", i, f)
	//	}
	//}
	//{
	//	logrus.Infof("DEBUG look at files in source directory")
	//	// fs, err := os.ReadDir(srcpath)
	//	// if err != nil {
	//	// 	panic(err)
	//	// }
	//	// for i, f := range fs {
	//	// 	logrus.Infof("file[%d] %+v", i, f)
	//	// }
	//	if err := filepath.Walk(srcpath, func(path string, info fs.FileInfo, err error) error {
	//		if err != nil {
	//			return fmt.Errorf("failed to filepath.Walk on file '%s'. error: %w", path, err)
	//		}
	//		logrus.Infof("DEBUG file[%s] %+v", path, info)
	//		if info.IsDir() {
	//			return nil
	//		}
	//		byt, err := os.ReadFile(path)
	//		if err != nil {
	//			return fmt.Errorf("failed to read the file '%s'. error: %w", path, err)
	//		}
	//		logrus.Infof("the file data is:\n%s", string(byt))
	//		return nil
	//	}); err != nil {
	//		logrus.Fatalf("failed to filepath.Walk on directory '%s'. error: %q", srcpath, err)
	//	}
	//}
	logrus.Infof("planfile: '%s'", planfile)
	fi, err = os.Stat(planfile)
	if err == nil && fi.IsDir() {
		planfile = filepath.Join(planfile, common.DefaultPlanFile)
		_, err = os.Stat(planfile)
	}
	if err != nil {
		if os.IsNotExist(err) {
			logrus.Warnf("the plan file doesn't exist")
			if strings.HasSuffix(planfile, string(os.PathSeparator)) {
				planfile = filepath.Join(planfile, common.DefaultPlanFile)
			} else if !strings.Contains(filepath.Base(planfile), ".") {
				planfile = filepath.Join(planfile, common.DefaultPlanFile)
			}
		} else {
			// logrus.Warnf("failed to stat. file info: %+v", fi)
			logrus.Fatalf("failed to access the plan file at path '%s' . Error: %q", planfile, err)
		}
	}

	qaengine.StartEngine(true, 0, true)
	qaengine.SetupConfigFile("", flags.setconfigs, flags.configs, flags.preSets, false)
	if flags.progressServerPort != 0 {
		startPlanProgressServer(flags.progressServerPort)
	}
	p, err := lib.CreatePlan(ctx, srcpath, "", customizationsPath, flags.transformerSelector, name)
	if err != nil {
		logrus.Fatalf("failed to create the plan. Error: %q", err)
	}
	if err = plantypes.WritePlan(planfile, p); err != nil {
		logrus.Fatalf("failed to write the plan to file at path %s . Error: %q", planfile, err)
	}
	logrus.Debugf("Plan : %+v", p)
	logrus.Infof("Plan can be found at [%s].", planfile)
	if len(p.Spec.Services) == 0 && len(p.Spec.InvokedByDefaultTransformers) == 0 {
		if flags.failOnEmptyPlan {
			logrus.Fatalf("Did not detect any services in the directory %s . Also we didn't find any default transformers to run.", srcpath)
		}
		logrus.Warnf("Did not detect any services in the directory %s . Also we didn't find any default transformers to run.", srcpath)
	}
	//{
	//	logrus.Infof("DEBUG after planning, list files")
	//	fs, err := os.ReadDir(".")
	//	if err != nil {
	//		panic(err)
	//	}
	//	for i, f := range fs {
	//		logrus.Infof("DEBUG file[%d] %+v", i, f)
	//	}
	//}
}

// GetPlanCommand returns a command to do the planning
func GetPlanCommand() *cobra.Command {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	viper.AutomaticEnv()

	flags := planFlags{}
	planCmd := &cobra.Command{
		Use:   "plan",
		Short: "Plan out a move",
		Long:  "Discover and create a plan file based on an input directory",
		Run:   func(cmd *cobra.Command, _ []string) { planHandler(cmd, flags) },
	}

	planCmd.Flags().StringVarP(&flags.srcpath, sourceFlag, "s", "", "Specify source directory or a git url (see https://move2kube.konveyor.io/concepts/git-support).")
	planCmd.Flags().StringVarP(&flags.planfile, planFlag, "p", common.DefaultPlanFile, "Specify a file path to save plan to.")
	planCmd.Flags().StringVarP(&flags.name, nameFlag, "n", common.DefaultProjectName, "Specify the project name.")
	planCmd.Flags().StringVarP(&flags.customizationsPath, customizationsFlag, "c", "", "Specify directory or a git url (see https://move2kube.konveyor.io/concepts/git-support) where customizations are stored. By default we look for "+common.DefaultCustomizationDir)
	planCmd.Flags().StringSliceVarP(&flags.configs, configFlag, "f", []string{}, "Specify config file locations. By default we look for "+common.DefaultConfigFilePath)
	planCmd.Flags().StringVarP(&flags.transformerSelector, transformerSelectorFlag, "t", "", "Specify the transformer selector.")
	planCmd.Flags().StringSliceVar(&flags.preSets, preSetFlag, []string{}, "Specify preset config to use.")
	planCmd.Flags().StringArrayVar(&flags.setconfigs, setConfigFlag, []string{}, "Specify config key-value pairs.")
	planCmd.Flags().IntVar(&flags.progressServerPort, planProgressPortFlag, 0, "Port for the plan progress server. If not provided, the server won't be started.")
	planCmd.Flags().BoolVar(&flags.disableLocalExecution, common.DisableLocalExecutionFlag, false, "Allow files to be executed locally.")
	planCmd.Flags().BoolVar(&flags.failOnEmptyPlan, common.FailOnEmptyPlan, false, "If true, planning will exit with a failure exit code if no services are detected (and no default transformers are found).")

	must(planCmd.Flags().MarkHidden(planProgressPortFlag))

	return planCmd
}
