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

package java

import (
	"bufio"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/konveyor/move2kube/common"
	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/types/source/maven"
	"github.com/konveyor/move2kube/types/source/springboot"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	// EurekaConfigType defines config type
	EurekaConfigType transformertypes.ConfigType = "ServiceModule"
)

const (
	// PomWithEureka defines path type
	PomWithEureka transformertypes.PathType = "PomWithEureka"
	// JavaWithFeign defines path type
	JavaWithFeign transformertypes.PathType = "JavaWithFeign"
	// JavaWithEureka defines path type
	JavaWithEureka transformertypes.PathType = "JavaWithEureka"
	// PropertiesWithEureka defines path type
	PropertiesWithEureka transformertypes.PathType = "PropertiesWithEureka"
	// PropertiesWithConfig defines path type
	PropertiesWithConfig transformertypes.PathType = "PropertiesWithConfig"
	// ConfigServerURL defines environment variable name
	ConfigServerURL string = "${CONFIG_SERVER_URL}"
)

var (
	eurekaRegex = regexp.MustCompile(".*EnableEurekaClient.*")
	feignRegex  = regexp.MustCompile(".*@FeignClient.*")
)

// EurekaReplaceEngine implements Transformer interface
type EurekaReplaceEngine struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// EurekaConfig defines spring boot properties
type EurekaConfig struct {
	ServiceName string `yaml:"serviceName,omitempty"`
	ServicePort int    `yaml:"servicePort,omitempty"`
}

func removeDuplicateValues(Slice []string) []string {
	keys := map[string]bool{}
	list := []string{}
	for _, entry := range Slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func buildNode(key, value string) []*yaml.Node {
	keyNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: key,
	}
	valueNode := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
	return []*yaml.Node{keyNode, valueNode}
}

// Init Initializes the transformer
func (t *EurekaReplaceEngine) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *EurekaReplaceEngine) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// DirectoryDetect runs detect in each sub directory
func (t *EurekaReplaceEngine) DirectoryDetect(dir string) (services map[string][]transformertypes.Artifact, err error) {
	destEntries, err := os.ReadDir(dir)
	var ec EurekaConfig
	var pomWithEureka []string
	var javaWithFeign []string
	var javaWithEureka []string
	var propertiesWithEureka []string
	var propertiesWithConfig []string

	if err != nil {
		logrus.Errorf("Unable to process directory %s : %s", dir, err)
	} else {
		for _, de := range destEntries {
			if de.Name() == maven.PomXMLFileName {

				// filled with previously declared xml
				pomStr, err := os.ReadFile(filepath.Join(dir, de.Name()))
				if err != nil {
					return nil, err
				}

				// Load pom from string
				var pom maven.Pom
				if err := xml.Unmarshal([]byte(pomStr), &pom); err != nil {
					logrus.Errorf("unable to unmarshal pom file. Reason: %s", err)
					return nil, err
				}

				// detect service module
				var isEurekaClient bool = false
				var isFeignClient bool = false
				if pom.Dependencies == nil {
					logrus.Debugf("Ignoring pom at %s as does not contain any dependencies", dir)
					return nil, nil
				}
				for _, dep := range *pom.Dependencies {
					if dep.ArtifactID == "spring-cloud-starter-netflix-eureka-client" {
						isEurekaClient = true
						pomWithEureka = append(pomWithEureka, filepath.Join(dir, de.Name()))
					}
					if dep.ArtifactID == "spring-cloud-starter-openfeign" {
						isFeignClient = true
					}
				}

				if !isEurekaClient && !isFeignClient {
					return nil, nil
				}

				// capture port and app name
				yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
				if err != nil {
					logrus.Errorf("Unable to get all yaml files : %s", err)
				}
				for _, path := range yamlpaths {
					sb := springboot.SpringApplicationYaml{}
					if err := common.ReadYaml(path, &sb); err != nil {
						continue
					}
					ec.ServiceName = sb.Spring.SpringApplication.Name
					ec.ServicePort = sb.Server.Port
					break
				}

				// record files to put it in plan phase
				if isEurekaClient {
					// record properties file
					yamlpaths, err := common.GetFilesByExt(dir, []string{".yaml", ".yml"})
					if err != nil {
						logrus.Errorf("Unable to fetch yaml files at path %s Error: %q", dir, err)
						return nil, err
					}
					for _, path := range yamlpaths {
						t := yaml.Node{}
						sourceYaml, _ := os.ReadFile(path)
						_ = yaml.Unmarshal(sourceYaml, &t)
						for _, n := range t.Content {
							for _, n1 := range n.Content {
								if n1.Value == "eureka" {
									propertiesWithEureka = append(propertiesWithEureka, path)
								}
								for _, n2 := range n1.Content {
									if n2.Value == "config" {
										propertiesWithConfig = append(propertiesWithConfig, path)
									}
								}
							}
						}

					}

					// record java files
					javapaths, err := common.GetFilesByExt(dir, []string{".java"})
					if err != nil {
						logrus.Errorf("Unable to get all java files : %s", err)
					}
					for _, path := range javapaths {
						file, err := os.Open(path)
						if err != nil {
							logrus.Errorf("failed opening file: %s", err)
							continue
						}

						scanner := bufio.NewScanner(file)
						scanner.Split(bufio.ScanLines)
						var txtlines []string
						for scanner.Scan() {
							txtlines = append(txtlines, scanner.Text())
						}
						defer file.Close()
						for _, eachline := range txtlines {
							match := eurekaRegex.MatchString(eachline)
							if match {
								javaWithEureka = append(javaWithEureka, path)
							}
							match = feignRegex.MatchString(eachline)
							if match {
								javaWithFeign = append(javaWithFeign, path)
							}
						}
					}
				}

				// remove duplicate entries and empty entries
				pomWithEureka = removeDuplicateValues(pomWithEureka)
				javaWithFeign = removeDuplicateValues(javaWithFeign)
				javaWithEureka = removeDuplicateValues(javaWithEureka)
				propertiesWithEureka = removeDuplicateValues(propertiesWithEureka)
				propertiesWithConfig = removeDuplicateValues(propertiesWithConfig)
				transformerpaths := map[transformertypes.PathType][]string{}

				if len(pomWithEureka) > 0 {
					transformerpaths[PomWithEureka] = pomWithEureka
				}
				if len(javaWithFeign) > 0 {
					transformerpaths[JavaWithFeign] = javaWithFeign
				}
				if len(javaWithEureka) > 0 {
					transformerpaths[JavaWithEureka] = javaWithEureka
				}
				if len(propertiesWithEureka) > 0 {
					transformerpaths[PropertiesWithEureka] = propertiesWithEureka
				}
				if len(propertiesWithConfig) > 0 {
					transformerpaths[PropertiesWithConfig] = propertiesWithConfig
				}
				ct := transformertypes.Artifact{
					Configs: map[transformertypes.ConfigType]interface{}{
						EurekaConfigType: ec,
					},
					Paths: transformerpaths,
				}
				return map[string][]transformertypes.Artifact{ec.ServiceName: {ct}}, nil
			}

		}
	}

	return nil, nil
}

// Transform transforms the artifacts
func (t *EurekaReplaceEngine) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	for _, a := range newArtifacts {
		var feignclients []string
		var sConfig EurekaConfig
		err := a.GetConfig(EurekaConfigType, &sConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
			continue
		}

		var seConfig artifacts.ServiceConfig
		err = a.GetConfig(artifacts.ServiceConfigType, &seConfig)
		if err != nil {
			logrus.Errorf("unable to load config for Transformer into %T : %s", seConfig, err)
			continue
		}

		pomWithEureka := a.Paths[PomWithEureka]
		javaWithFeign := a.Paths[JavaWithFeign]
		javaWithEureka := a.Paths[JavaWithEureka]
		propertiesWithEureka := a.Paths[PropertiesWithEureka]
		propertiesWithConfig := a.Paths[PropertiesWithConfig]

		for _, path := range pomWithEureka {
			// filled with previously declared xml
			pomStr, err := os.ReadFile(path)
			if err != nil {
				return nil, nil, err
			}

			// Load pom from string
			var pom maven.Pom
			if err := xml.Unmarshal([]byte(pomStr), &pom); err != nil {
				logrus.Errorf("unable to unmarshal pom file. Reason: %s", err)
				return nil, nil, err
			}
			// remove it from pom.xml
			eurekaIndex := 0
			if pom.Dependencies == nil {
				logrus.Debugf("Ignoring pom at %s as does not contain any dependencies", path)
				return nil, nil, nil
			}
			for i, dep := range *pom.Dependencies {
				if dep.ArtifactID == "spring-cloud-starter-netflix-eureka-client" {
					eurekaIndex = i
				}
			}
			*pom.Dependencies = append((*pom.Dependencies)[:eurekaIndex], (*pom.Dependencies)[eurekaIndex+1:]...)

			// write modified pom.xml to file
			pombytes, err := xml.MarshalIndent(pom, "  ", "    ")
			if err != nil {
				logrus.Errorf("Unable to convert struct to bytes Reason: %s", err)
				return nil, nil, err
			}
			err = os.WriteFile(path, pombytes, 0644)
			if err != nil {
				return nil, nil, err
			}

		}

		for _, path := range javaWithEureka {
			// remove it from java files
			file, err := os.Open(path)
			if err != nil {
				logrus.Errorf("failed opening file: %s", err)
				continue
			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			var txtlines []string
			var modifiedtxtlines []string
			for scanner.Scan() {
				txtlines = append(txtlines, scanner.Text())
			}
			defer file.Close()

			for _, eachline := range txtlines {
				match := eurekaRegex.MatchString(eachline)
				if match {
					eachline = " "
				}
				modifiedtxtlines = append(modifiedtxtlines, eachline)
			}
			file, err = os.Create(path)
			if err != nil {
				logrus.Errorf("failed to write to file: %s", err)
				continue
			}
			defer file.Close()
			w := bufio.NewWriter(file)
			for _, line := range modifiedtxtlines {
				fmt.Fprintln(w, line)
			}
			w.Flush()

		}

		for _, path := range javaWithFeign {
			// remove it from java files
			file, err := os.Open(path)

			if err != nil {
				logrus.Errorf("failed opening file: %s", err)
				continue
			}

			scanner := bufio.NewScanner(file)
			scanner.Split(bufio.ScanLines)
			var txtlines []string
			var modifiedtxtlines []string
			for scanner.Scan() {
				txtlines = append(txtlines, scanner.Text())
			}
			defer file.Close()

			for _, eachline := range txtlines {
				match := feignRegex.MatchString(eachline)
				if match {
					eachline = eachline[:len(eachline)-1]
					split := strings.Split(eachline, "(")
					arguments := split[1]
					split = strings.Split(arguments, ",")
					servicename := "default"
					for _, el := range split {
						keyvalue := strings.Split(el, "=")
						keyvalue[0] = strings.TrimSpace(keyvalue[0])
						keyvalue[1] = strings.TrimSpace(keyvalue[1])
						if keyvalue[0] == "name" {
							servicename = keyvalue[1][1 : len(keyvalue[1])-1]
						}
					}
					servicename = strings.ToUpper(servicename)
					feignclients = append(feignclients, (servicename + "_URL"))
					eachline = eachline + ", url = \"${" + servicename + "_URL" + "}\")"
				}

				modifiedtxtlines = append(modifiedtxtlines, eachline)
			}

			file, err = os.Create(path)
			if err != nil {
				logrus.Errorf("failed to write to file: %s", err)
				continue
			}
			defer file.Close()
			w := bufio.NewWriter(file)
			for _, line := range modifiedtxtlines {
				fmt.Fprintln(w, line)
			}
			w.Flush()
		}

		for _, path := range propertiesWithEureka {
			// remove it from spring config file
			t := yaml.Node{}
			sourceYaml, _ := os.ReadFile(path)
			_ = yaml.Unmarshal(sourceYaml, &t)
			// delete eureka node
			eurekaNodeIndex := -1
			for _, n := range t.Content {
				for index, n1 := range n.Content {
					if n1.Value == "eureka" {
						eurekaNodeIndex = index
						n.Content = append(n.Content[:eurekaNodeIndex], n.Content[eurekaNodeIndex+2:]...)
					}
				}
			}

			if eurekaNodeIndex != -1 {
				out, _ := yaml.Marshal(&t)
				_ = os.WriteFile(path, out, 0644)
			}

			t = yaml.Node{}
			sourceYaml, _ = os.ReadFile(path)
			_ = yaml.Unmarshal(sourceYaml, &t)

			for _, n := range t.Content {
				for _, f := range feignclients {
					envvar := "${" + f + "}"
					n.Content = append(n.Content, buildNode(f, envvar)...)
				}
			}
			out, _ := yaml.Marshal(&t)
			_ = os.WriteFile(path, out, 0644)

		}
		for _, path := range propertiesWithConfig {
			// access config through url (remove eureka)
			t := yaml.Node{}
			sourceYaml, _ := os.ReadFile(path)
			_ = yaml.Unmarshal(sourceYaml, &t)
			insert := false
			for _, n := range t.Content {
				for _, n1 := range n.Content {
					for _, n2 := range n1.Content {
						if insert {
							n2.Content = append(n2.Content, buildNode("uri", ConfigServerURL)...)
							break
						}
						if n2.Value == "config" {
							insert = true
						}
					}
				}
			}

			out, _ := yaml.Marshal(&t)
			_ = os.WriteFile(path, out, 0644)
		}

		// copy project code from source to destination
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			SrcPath:  "",
			DestPath: common.DefaultSourceDir,
		})

		// for every pomWithEureka file copy to destination
		for _, path := range pomWithEureka {
			relPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), path)
			if err != nil {
				logrus.Errorf("Unable to convert source path %s to be relative : %s", path, err)
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.ModifiedSourcePathMappingType,
				SrcPath:  path,
				DestPath: filepath.Join(common.DefaultSourceDir, relPath),
			})
		}

		// for every javaWithEureka file copy to destination
		for _, path := range javaWithEureka {
			relPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), path)
			if err != nil {
				logrus.Errorf("Unable to convert source path %s to be relative : %s", path, err)
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.ModifiedSourcePathMappingType,
				SrcPath:  path,
				DestPath: filepath.Join(common.DefaultSourceDir, relPath),
			})
		}

		// for every javaWithFeign file copy to destination
		for _, path := range javaWithFeign {
			relPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), path)
			if err != nil {
				logrus.Errorf("Unable to convert source path %s to be relative : %s", path, err)
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.ModifiedSourcePathMappingType,
				SrcPath:  path,
				DestPath: filepath.Join(common.DefaultSourceDir, relPath),
			})
		}

		// for every propertiesWithEureka file copy to destination
		for _, path := range propertiesWithEureka {
			relPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), path)
			if err != nil {
				logrus.Errorf("Unable to convert source path %s to be relative : %s", path, err)
			}
			pathMappings = append(pathMappings, transformertypes.PathMapping{
				Type:     transformertypes.ModifiedSourcePathMappingType,
				SrcPath:  path,
				DestPath: filepath.Join(common.DefaultSourceDir, relPath),
			})
		}
	}
	return pathMappings, nil, nil
}
