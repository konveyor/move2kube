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
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
)

type generateFlags struct {
	outputDir string
}

const docsTemplateStr = `---
layout: default
title: {{ .Title }}
permalink: /documentation/commands{{ .URL }}
parent: {{ .Parent }}
{{ .Extra -}}
---
{{ .Notes -}}

`

func generateHandler(flags generateFlags) {
	docsTemplate := template.Must(template.New("gen-docs").Parse(docsTemplateStr))
	filePrepender := func(filename string) string {
		data := struct {
			Title  string
			URL    string
			Parent string
			Extra  string
			Notes  string
		}{}
		commandsParent := "Move2Kube commands"
		data.Title = strings.TrimPrefix(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), types.AppName+"_")
		data.URL = "/" + data.Title
		data.Parent = commandsParent
		data.Extra = "grand_parent: Documentation\n"
		if data.Title == types.AppName {
			data.Title = commandsParent
			data.URL = ""
			data.Parent = "Documentation"
			data.Extra = "has_children: true\nnav_order: 1\nhas_toc: false\n"
			data.Notes = "\n###### This documentation was generated by running `" + types.AppName + " docs`\n\n"
		}
		output := bytes.Buffer{}
		if err := docsTemplate.Execute(&output, data); err != nil {
			logrus.Errorf("failed to generate the documentation for the filename %s using the template. Error: %q", filename, err)
			return ""
		}
		return output.String()
	}
	linkHandler := func(filename string) string {
		link := strings.TrimPrefix(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)), types.AppName+"_")
		if link == types.AppName {
			return "/documentation/commands"
		}
		return "/documentation/commands/" + link
	}
	logrus.Infof("generating the documentation for all the commands into the directory at the path: %s", flags.outputDir)
	if err := os.MkdirAll(flags.outputDir, common.DefaultDirectoryPermission); err != nil {
		logrus.Fatalf("error while making the output directory at path %s to store the documentation. Error: %q", flags.outputDir, err)
	}
	rootCmd := GetRootCmd()
	if err := doc.GenMarkdownTreeCustom(rootCmd, flags.outputDir, filePrepender, linkHandler); err != nil {
		logrus.Fatalf("error while generating documentation. Error: %q", err)
	}
}

// GetGenerateDocsCommand returns a command to generate the documentation for all the commands
func GetGenerateDocsCommand() *cobra.Command {
	viper.AutomaticEnv()
	flags := generateFlags{}
	generateDocsCmd := &cobra.Command{
		Hidden: true,
		Use:    "docs",
		Short:  "Generate the documentation for the commands",
		Long:   "Generate the documentation for the commands. The documentation is in markdown format.",
		Run:    func(_ *cobra.Command, __ []string) { generateHandler(flags) },
	}
	generateDocsCmd.Flags().StringVarP(&flags.outputDir, "output", "o", "commands", "Path to the directory where the documentation will be generated.")
	return generateDocsCmd
}
