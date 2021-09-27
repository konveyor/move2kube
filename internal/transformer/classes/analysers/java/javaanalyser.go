// /*
//  *  Copyright IBM Corporation 2021
//  *
//  *  Licensed under the Apache License, Version 2.0 (the "License");
//  *  you may not use this file except in compliance with the License.
//  *  You may obtain a copy of the License at
//  *
//  *        http://www.apache.org/licenses/LICENSE-2.0
//  *
//  *  Unless required by applicable law or agreed to in writing, software
//  *  distributed under the License is distributed on an "AS IS" BASIS,
//  *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  *  See the License for the specific language governing permissions and
//  *  limitations under the License.
//  */

package java

// import (
// 	"encoding/xml"
// 	"io/ioutil"
// 	"os"
// 	"path/filepath"
// 	"regexp"
// 	"strings"

// 	"github.com/konveyor/move2kube/environment"
// 	"github.com/konveyor/move2kube/internal/common"
// 	collectiontypes "github.com/konveyor/move2kube/types/collection"
// 	irtypes "github.com/konveyor/move2kube/types/ir"
// 	"github.com/konveyor/move2kube/types/source/maven"
// 	"github.com/konveyor/move2kube/types/source/springboot"
// 	transformertypes "github.com/konveyor/move2kube/types/transformer"
// 	"github.com/konveyor/move2kube/types/transformer/artifacts"
// 	"github.com/sirupsen/logrus"
// )

// const (
// 	gradleBuildFileName    string = "build.gradle"
// 	gradleSettingsFileName string = "settings.gradle"
// )

// const (
// 	javaServiceConfigType transformertypes.ConfigType = "JavaService"
// )

// const (
// 	mavenPomXML                   transformertypes.PathType = "MavenPomXML"
// 	springbootApplicationFilePath transformertypes.PathType = "SpringbootApplicationFile"
// )

// // JavaBuildTool defines the build types understood by java transformer
// type JavaBuildTool = string

// const (
// 	buildToolMaven  JavaBuildTool = "maven"
// 	buildToolGradle JavaBuildTool = "gradle"
// )

// // JavaAnalyser implements Transformer interface
// type JavaAnalyser struct {
// 	Config     transformertypes.Transformer
// 	Env        *environment.Environment
// 	JavaConfig JavaYamlConfig
// }

// type JavaYamlConfig struct {
// 	JavaVersion string `yaml:"defaultJavaVersion"`
// }

// // JavaConfig defines JavaConfig properties
// type JavaConfig struct {
// 	BuildTool   JavaBuildTool `yaml:"buildTool,omitempty"`
// 	JavaVersion string        `yaml:"javaVersion,omitempty"`
// }

// // SpringBootConfig defines SpringBootConfig properties
// type SpringBootConfig struct {
// 	Profiles []string `yaml:"profiles,omitempty"`
// }

// // JavaBuildTemplateConfig defines JavaBuildTemplateConfig properties
// type JavaBuildTemplateConfig struct {
// 	JavaInstallPackageName string `yaml:"javaInstallPackageName,omitempty"`
// }

// // ConfigurationFromBuildTool defines Configuration properties
// type ConfigurationFromBuildTool struct {
// 	BuildTool                     JavaBuildTool           `yaml:"buildTool,omitempty"` // Maven or Gradle
// 	HasModules                    bool                    `yaml:"hasModules,omitempty"`
// 	IsSpringboot                  bool                    `yaml:"isSpringboot,omitempty"`
// 	IsTomcatProvided              bool                    `yaml:"isTomcatProvided,omitempty"`
// 	Packaging                     string                  `yaml:"packaging,omitempty"`
// 	JavaVersion                   string                  `yaml:"javaVersion,omitempty"`
// 	TomcatVersion                 string                  `yaml:"tomcatVersion,omitempty"`
// 	Name                          string                  `yaml:"name,omitempty"`
// 	ArtifactID                    string                  `yaml:"artifactId,omitempty"`
// 	Version                       string                  `yaml:"version,omitempty"`
// 	FileSuffix                    string                  `yaml:"fileSuffix,omitempty"`
// 	Profiles                      []string                `yaml:"profiles,omitempty"`
// 	SpringbootConfigFromBuildTool JavaConfigFromBuildTool `yaml:"springbootConfigFromBuildTool,omitempty"`
// }

// // JavaConfigFromBuildTool defines JavaConfigFromBuildTool properties
// type JavaConfigFromBuildTool struct {
// 	IsTomcatProvided bool     `yaml:"isTomcatProvided,omitempty"`
// 	Profiles         []string `yaml:"profiles,omitempty"`
// }

// // Init Initializes the transformer
// func (t *JavaAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
// 	t.Config = tc
// 	t.Env = env
// 	t.JavaConfig = JavaYamlConfig{}
// 	err = common.GetObjFromInterface(t.Config.Spec.Config, &t.JavaConfig)
// 	if err != nil {
// 		logrus.Errorf("unable to load config for Transformer %+v into %T : %s", t.Config.Spec.Config, t.JavaConfig, err)
// 		return err
// 	}
// 	return nil
// }

// // GetConfig returns the transformer config
// func (t *JavaAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
// 	return t.Config, t.Env
// }

// // BaseDirectoryDetect runs detect in base directory
// func (t *JavaAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
// 	return nil, nil, nil
// }

// // getFileLines gets the lines from a file as a list of strings
// func getFileLines(filePath string) ([]string, error) {

// 	// Implementation using ioutil.ReadFile
// 	fileContent, err := ioutil.ReadFile(filePath)
// 	if err != nil {
// 		return nil, err
// 	}
// 	fileContentAsString := string(fileContent) // originally is an array of bytes
// 	fileLines := strings.Split(fileContentAsString, "\n")
// 	return fileLines, nil

// }

// // getGradleData extracts info from Gradle files
// func getGradleData(buildGradlePath string, settingsGradlePath string) (configuration ConfigurationFromBuildTool, err error) {

// 	buildGradleLines, err := getFileLines(buildGradlePath)
// 	if err != nil {
// 		logrus.Errorf("failed getting lines from file: %s", buildGradlePath)
// 	}

// 	openBlock := false
// 	openMultilineCommentBlock := false

// 	blockParams := map[string][]string{}
// 	var singleParams []string
// 	var blockContent []string
// 	var blockName string
// 	for _, line := range buildGradleLines {

// 		if strings.Contains(line, "/*") {
// 			openMultilineCommentBlock = true
// 		}

// 		if strings.Contains(line, "*/") && openMultilineCommentBlock {
// 			openMultilineCommentBlock = false
// 			continue
// 		}

// 		if openMultilineCommentBlock {
// 			continue
// 		}

// 		if strings.Contains(line, "{") {
// 			openBlock = true
// 			blockName = strings.Replace(line, "{", "", 1)
// 			blockName = strings.TrimSpace(blockName)
// 			blockContent = nil
// 		}

// 		if openBlock && strings.Contains(line, "}") {
// 			if blockContent != nil {
// 				blockParams[blockName] = blockContent
// 			}
// 			openBlock = false
// 			blockContent = nil
// 		}

// 		if openBlock && !strings.Contains(line, "}") && !strings.Contains(line, "{") {
// 			line = strings.TrimSpace(line)
// 			blockContent = append(blockContent, line)
// 		}

// 		if !openBlock && !strings.Contains(line, "}") && !strings.Contains(line, "{") && line != "" && strings.TrimSpace(line) != "" {
// 			singleParams = append(singleParams, line)
// 		}
// 	}

// 	// (Optional) Data extraction from settings.gradle
// 	settingsGradleLines, err := getFileLines(settingsGradlePath)
// 	if err != nil {
// 		logrus.Errorf("failed getting lines from file: %s", settingsGradlePath)
// 	}

// 	name := ""
// 	artifactId := ""
// 	for _, sl := range settingsGradleLines {
// 		sl = strings.TrimSpace(sl)
// 		if strings.Contains(sl, "rootProject.name") {
// 			slSplitted := strings.Split(sl, "=")
// 			if len(slSplitted) == 2 {
// 				name = slSplitted[1]
// 				artifactId = slSplitted[1]
// 			}
// 		}
// 	}

// 	// Collect Modules
// 	var modules []string
// 	version := ""
// 	javaVersion := ""

// 	for _, sp := range singleParams {
// 		spSplitted := strings.Split(sp, " ")
// 		if strings.Contains(sp, "include") {

// 			module := spSplitted[len(spSplitted)-1]
// 			modules = append(modules, module)
// 		}

// 		if spSplitted[0] == "version" && len(spSplitted) == 2 {
// 			version = spSplitted[1]
// 		}

// 		if strings.Contains(sp, "sourceCompatibility") {
// 			spSplittedEq := strings.Split(sp, "=")
// 			if len(spSplittedEq) == 2 {
// 				javaVersion = spSplittedEq[1]
// 			}
// 		}
// 	}

// 	hasModules := false
// 	if len(modules) > 0 {
// 		hasModules = true
// 	}

// 	isSpringboot := false
// 	isTomcatProvided := false
// 	packaging := ""

// 	for blockId, blockContent := range blockParams {

// 		switch blockId {
// 		case "dependencies":
// 			for _, dependency := range blockContent {
// 				if strings.Contains(dependency, "org.springframework.boot") {
// 					isSpringboot = true
// 				}

// 				if strings.Contains(dependency, "providedRuntime 'org.springframework.boot:spring-boot-starter-tomcat'") {
// 					isTomcatProvided = true
// 				}
// 			}
// 		case "war":
// 			packaging = "war"
// 		}
// 	}

// 	conf := ConfigurationFromBuildTool{
// 		BuildTool:        buildToolGradle,
// 		HasModules:       hasModules,
// 		IsSpringboot:     isSpringboot,
// 		IsTomcatProvided: isTomcatProvided,
// 		Packaging:        packaging,
// 		JavaVersion:      javaVersion,
// 		Name:             name,
// 		ArtifactID:       artifactId,
// 		Version:          version,
// 	}
// 	return conf, nil
// }

// // readPropertiesFile reads a .properties file
// func readPropertiesFile(propertiesFilePath string) (result map[string]string, err error) {

// 	res := map[string]string{}

// 	propertiesFileLines, err := getFileLines(propertiesFilePath)
// 	if err != nil {
// 		logrus.Errorf("Failed getting lines from file: %s", propertiesFilePath)
// 		return res, err
// 	}

// 	for _, line := range propertiesFileLines {
// 		if line != "" && strings.Contains(line, "=") {
// 			lineContent := strings.Split(line, "=")
// 			if len(lineContent) == 2 {
// 				res[lineContent[0]] = lineContent[1]
// 			}
// 		}
// 	}
// 	return res, err
// }

// // getMavenData extracts data from maven files
// func getMavenData(pomXMLPath string) (configuration ConfigurationFromBuildTool, err error) {

// 	// filled with previously declared xml
// 	pomStr, err := ioutil.ReadFile(pomXMLPath)
// 	if err != nil {
// 		logrus.Errorf("Could not read the pom.xml file: %s", err)
// 		return ConfigurationFromBuildTool{}, err
// 	}

// 	// Load pom from string
// 	var pom maven.Pom
// 	if err := xml.Unmarshal([]byte(pomStr), &pom); err != nil {
// 		logrus.Errorf("unable to unmarshal pom file. Reason: %s", err)
// 		return ConfigurationFromBuildTool{}, err
// 	}

// 	hasModules := false
// 	if pom.Modules != nil && len(*(pom.Modules)) != 0 {
// 		hasModules = true
// 	}

// 	isSpringboot := false
// 	isTomcatProvided := false

// 	if pom.Dependencies == nil {
// 		logrus.Debugf("POM file at %s does not contain a dependencies block", pomXMLPath)
// 	} else {
// 		for _, dependency := range *pom.Dependencies {

// 			if strings.Contains(dependency.GroupID, "org.springframework.boot") {
// 				isSpringboot = true
// 			}

// 			if strings.Contains(dependency.ArtifactID, "spring-boot-starter-tomcat") && dependency.Scope == "provided" {
// 				isTomcatProvided = true
// 			}
// 		}
// 	}

// 	packaging := ""
// 	if pom.Packaging == "" {
// 		logrus.Debugf("Pom at %s does not contain a Packaging block", pomXMLPath)
// 	} else {
// 		packaging = pom.Packaging
// 		logrus.Debugf("Packaging: %s", packaging)
// 	}

// 	// Collect java / tomcat version fom the Properties block
// 	javaVersion := ""
// 	tomcatVersion := ""
// 	if pom.Properties == nil {
// 		logrus.Debugf("Pom at %s  does not contain a Properties block", pomXMLPath)
// 	} else {

// 		for k, v := range pom.Properties.Entries {
// 			switch k {
// 			// Only for springboot apps
// 			case "java.version":
// 				javaVersion = v
// 			case "tomcat.version":
// 				tomcatVersion = v
// 			// Non springboot apps:
// 			case "maven.compiler.target":
// 				if javaVersion == "" {
// 					javaVersion = v
// 				}
// 			}
// 		}
// 	}

// 	fileSuffix := ""
// 	if pom.Build.Plugins == nil {
// 		logrus.Debugf("Pom at %s does not contain a Build->Plugins block", pomXMLPath)
// 	} else {
// 		for _, bp := range *pom.Build.Plugins {
// 			//logrus.Debugf("s:", bp)
// 			if bp.Configuration.Classifier != "" {
// 				fileSuffix = bp.Configuration.Classifier
// 			}
// 		}
// 	}

// 	profiles := []string{}
// 	if pom.Profiles == nil {
// 		logrus.Debugf("Pom at %s does not contain a Profiles block", pomXMLPath)
// 	} else {
// 		for _, pr := range *pom.Profiles {
// 			profiles = append(profiles, pr.ID)
// 		}
// 	}

// 	conf := ConfigurationFromBuildTool{
// 		BuildTool:        buildToolMaven,
// 		HasModules:       hasModules,
// 		IsSpringboot:     isSpringboot,     //->  SpringbootConfigFromBuildTool struct
// 		IsTomcatProvided: isTomcatProvided, // x
// 		Packaging:        packaging,
// 		JavaVersion:      javaVersion,
// 		Name:             pom.Name,
// 		ArtifactID:       pom.ArtifactID,
// 		Version:          pom.Version,
// 		FileSuffix:       fileSuffix,
// 		TomcatVersion:    tomcatVersion,
// 		Profiles:         profiles, // x
// 		SpringbootConfigFromBuildTool: JavaConfigFromBuildTool{
// 			IsTomcatProvided: isTomcatProvided,
// 			Profiles:         profiles,
// 		},
// 	}
// 	return conf, nil
// }

// //getFilesByRegExp returns files that match a reg exp
// func getFilesByRegExp(inputPath string, re string) ([]string, error) {
// 	var files []string
// 	if info, err := os.Stat(inputPath); os.IsNotExist(err) {
// 		logrus.Warnf("Error in walking through files due to : %q", err)
// 		return files, err
// 	} else if !info.IsDir() {
// 		logrus.Warnf("The path %q is not a directory.", inputPath)
// 	}

// 	reg, err2 := regexp.Compile(re)
// 	if err2 != nil {
// 		logrus.Warnf("Could not compile regular expression: %s", re)
// 		return files, err2
// 	}

// 	err := filepath.Walk(inputPath, func(path string, info os.FileInfo, err error) error {
// 		if err != nil && path == inputPath { // if walk for root search path return gets error
// 			// then stop walking and return this error
// 			return err
// 		}
// 		if err != nil {
// 			logrus.Warnf("Skipping path %q due to error: %q", path, err)
// 			return nil
// 		}
// 		// Skip directories
// 		if info.IsDir() {
// 			return nil
// 		}
// 		fname := filepath.Base(path)

// 		if reg.MatchString(fname) {
// 			files = append(files, path)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		logrus.Warnf("Error in walking through files due to : %s", err)
// 		return files, err
// 	}
// 	//logrus.Debugf("No of files with %s names identified : %d", names, len(files))
// 	return files, nil
// }

// // DirectoryDetect runs detect in each sub directory
// func (t *JavaAnalyser) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
// 	destEntries, err := ioutil.ReadDir(dir)
// 	if err != nil {
// 		logrus.Errorf("Unable to process directory %s : %s", dir, err)
// 		return nil, nil, err
// 	}
// 	mavenFound := false
// 	gradleFound := false
// 	for _, de := range destEntries {
// 		if de.Name() == maven.PomXMLFileName {
// 			mavenFound = true
// 			continue //break
// 		}
// 		if de.Name() == gradleBuildFileName {
// 			gradleFound = true
// 			continue
// 		}
// 	}

// 	// If there are not build config files, we stop
// 	if !mavenFound && !gradleFound {
// 		return nil, nil, nil
// 	}

// 	var config ConfigurationFromBuildTool
// 	if mavenFound {
// 		mavenConfig, err := getMavenData(filepath.Join(dir, maven.PomXMLFileName))
// 		if err != nil {
// 			logrus.Errorf("Unable to load data from maven file %s", filepath.Join(dir, maven.PomXMLFileName))
// 		} else {
// 			config = mavenConfig
// 		}

// 	} else { // This hierarchy is by design. We are more confident on the maven extraction
// 		if gradleFound {
// 			gradleConfig, err := getGradleData(filepath.Join(dir, gradleBuildFileName), filepath.Join(dir, gradleSettingsFileName))
// 			if err != nil {
// 				logrus.Errorf("Unable to load data from gradle file %s", filepath.Join(dir, gradleBuildFileName))
// 			} else {
// 				config = gradleConfig
// 			}
// 		}
// 	}

// 	buildTool := config.BuildTool

// 	if config.HasModules {
// 		logrus.Debugf("Ignoring configuration at %s as it has modules", dir)
// 		return nil, nil, nil
// 	}

// 	// Collect java / tomcat version fom the Properties block
// 	javaVersion := ""
// 	if config.JavaVersion == "" { // default case
// 		javaVersion = "1.8"
// 	} else {
// 		javaVersion = config.JavaVersion
// 	}

// 	// Check if the application uses an embeded server or not.
// 	// This is based on having tomcat as `provided` and packaging as `war`
// 	// Initialize flag as false
// 	isServerEmbedded := false

// 	isPackagingWAR := false
// 	if config.Packaging == "war" {
// 		isPackagingWAR = true
// 	}

// 	// here we asses for both conditions
// 	isServerEmbedded = !(config.IsTomcatProvided && isPackagingWAR)

// 	// in the case we have standanlone java maven application (no springboot),
// 	// we assume the server is not embedded
// 	if !config.IsSpringboot {
// 		isServerEmbedded = false
// 	}

// 	// If the server is not embedded, we check if it is open-liberty or jboss/wildfly
// 	appServer := ""
// 	if !isServerEmbedded {
// 		// Server is not embedded. What type of server app are we using?

// 		// Search for server.xml files
// 		serverXMLfiles, err := common.GetFilesByName(dir, []string{"server.xml"})
// 		if err != nil {
// 			logrus.Debugf("Cannot get server.xml files: %s", err)
// 		}

// 		// Current assumption: if there is at least one server.xml file, -> open-liberty
// 		if len(serverXMLfiles) > 0 {
// 			appServer = "openliberty/open-liberty"
// 		} else {
// 			appServer = "jboss/wildfly"
// 		}
// 	}
// 	logrus.Debugf("App server: %s", appServer)

// 	// Check compatible image for the application server
// 	var appServerCandidateImages []collectiontypes.ImageInfoSpec

// 	if appServer != "" {
// 		if config.JavaVersion == "" { // default case
// 			javaVersion = "1.8"
// 		}

// 		mappingPath := filepath.Join(t.Env.GetEnvironmentContext(), "mappings/java2images_tags.yaml")
// 		images2Data := collectiontypes.NewImagesInfo()
// 		if err := common.ReadMove2KubeYaml(mappingPath, &images2Data); err != nil {
// 			logrus.Debugf("Could not load mapping at %s", mappingPath)
// 		}

// 		for _, im := range images2Data.Spec {
// 			if im.Params["javaVersion"] == javaVersion && im.Params["serverApp"] == appServer {
// 				appServerCandidateImages = append(appServerCandidateImages, im)
// 			}
// 		}
// 	}

// 	for _, e := range appServerCandidateImages {
// 		logrus.Debugf("e: %s", e.Tags)
// 	}

// 	appServerImage := ""
// 	if len(appServerCandidateImages) > 0 {
// 		appServerImage = appServerCandidateImages[0].Tags[0]
// 	}
// 	logrus.Debugf("app server image %s", appServerImage)

// 	appPropfiles, err := common.GetFilesByName(dir, []string{"", "application.properties"})
// 	if err != nil {
// 		logrus.Debugf("Cannot get application files: %s", err)
// 	}
// 	logrus.Debugf("App prop files %s", appPropfiles)

// 	// Java images for build and deploy

// 	// build
// 	javaPackageNamesMappingPath := filepath.Join(t.Env.GetEnvironmentContext(), "mappings/java_version2package_name.json")
// 	var javaPackageNamesMapping map[string]string
// 	if err := common.ReadJSON(javaPackageNamesMappingPath, &javaPackageNamesMapping); err != nil {
// 		logrus.Debugf("Could not load mapping at %s", javaPackageNamesMappingPath)
// 	}

// 	javaPackageName := ""
// 	if val, ok := javaPackageNamesMapping[javaVersion]; ok {
// 		javaPackageName = val
// 	}

// 	// Get app file and app name
// 	appName := ""
// 	if config.Name != "" {
// 		appName = config.Name
// 	} else if config.ArtifactID != "" {
// 		appName = config.ArtifactID
// 	} else {
// 		appName = filepath.Base(dir)
// 	}

// 	appName = common.NormalizeForServiceName(appName)
// 	appName = strings.Replace(appName, " ", "-", 1)

// 	appFile := ""
// 	if config.ArtifactID != "" {
// 		appFile = config.ArtifactID
// 	} else {
// 		if config.Name != "" {
// 			appFile = config.Name
// 		}
// 	}
// 	if appFile != "" {
// 		if config.Version != "" {
// 			appFile = appFile + "-" + config.Version
// 		}

// 		if config.FileSuffix != "" {
// 			appFile = appFile + "-" + config.FileSuffix
// 		}

// 		if config.Packaging != "" {
// 			appFile = appFile + "." + config.Packaging
// 		} else {
// 			appFile = appFile + ".jar"
// 		}
// 	}

// 	deploymentFile := ""
// 	if config.ArtifactID != "" {
// 		deploymentFile = config.ArtifactID
// 	}
// 	if deploymentFile != "" {
// 		if config.Packaging != "" {
// 			deploymentFile = deploymentFile + "." + config.Packaging
// 		} else {
// 			deploymentFile = deploymentFile + ".jar"
// 		}
// 	}

// 	// Collect application.yml/yaml files
// 	appfiles, err := common.GetFilesByName(dir, []string{"application.yaml", "application.yml"})
// 	if err != nil {
// 		logrus.Debugf("Cannot get application files: %s", err)
// 	}

// 	// General query for springboot config files
// 	springConfigFiles, err := getFilesByRegExp(dir, "application.*.(yaml|yml|properties)")
// 	if err != nil {
// 		logrus.Debugf("Cannot get application files: %s", err)
// 	}
// 	logrus.Debugf("testfiles %s", springConfigFiles)

// 	// Remove not relevant files
// 	springConfigFilesFiltered := []string{}
// 	for _, scf := range springConfigFiles {
// 		if !strings.Contains(scf, "test/") && !strings.Contains(scf, "secret/") {
// 			// in the case of secret, we probably need it anyway
// 			springConfigFilesFiltered = append(springConfigFilesFiltered, scf)
// 		}
// 	}

// 	// Divide the remaining list into yaml and properties
// 	springConfigFilesYaml := []string{}
// 	springConfigFilesProperties := []string{}

// 	for _, scf := range springConfigFilesFiltered {
// 		// regex for this?
// 		segments := strings.Split(scf, ".")
// 		extension := segments[len(segments)-1]

// 		if extension == "yml" || extension == "yaml" {
// 			springConfigFilesYaml = append(springConfigFilesYaml, scf)
// 		} else if extension == "properties" {
// 			springConfigFilesProperties = append(springConfigFilesProperties, scf)
// 		}
// 	}

// 	propFilesMap := map[string]map[string]string{}
// 	profiles := []string{}

// 	for _, propFile := range springConfigFilesProperties {
// 		propFileContent, err := readPropertiesFile(propFile)
// 		if err != nil {
// 			logrus.Debugf("Could not process .properties file %s", propFile)
// 		} else {
// 			fileName := filepath.Base(propFile)
// 			if strings.Contains(fileName, "-") {
// 				parts := strings.Split(fileName, "-")
// 				profile := strings.Replace(parts[1], ".properties", "", 1)
// 				profiles = append(profiles, profile)
// 			}

// 			propFilesMap[propFile] = propFileContent
// 		}
// 	}

// 	yamlFilesMap := map[string]springboot.SpringApplicationYaml{}
// 	for _, yamlFile := range springConfigFilesYaml {

// 		var springApplicationYaml springboot.SpringApplicationYaml
// 		err = common.ReadYaml(yamlFile, &springApplicationYaml)
// 		if err != nil {
// 			logrus.Debugf("Could not load application file %s", yamlFile)
// 			continue
// 		}
// 		if (springApplicationYaml == springboot.SpringApplicationYaml{}) {
// 			logrus.Debugf("No information found in application file %s", yamlFile)
// 			continue
// 		}
// 		yamlFilesMap[yamlFile] = springApplicationYaml
// 	}

// 	// original code
// 	validSpringbootFiles := []string{}
// 	ports := []int{}
// 	for _, appfile := range appfiles {
// 		var springApplicationYaml springboot.SpringApplicationYaml
// 		err = common.ReadYaml(appfile, &springApplicationYaml)
// 		if err != nil {
// 			logrus.Debugf("Could not load application file %s", appfile)
// 			continue
// 		}
// 		if (springApplicationYaml == springboot.SpringApplicationYaml{}) {
// 			logrus.Debugf("No information found in application file %s", appfile)
// 			continue
// 		}
// 		validSpringbootFiles = append(validSpringbootFiles, appfile)

// 		if springApplicationYaml.Spring.SpringApplication.Name != "" {
// 			appName = springApplicationYaml.Spring.SpringApplication.Name
// 		}

// 		if springApplicationYaml.Server.Port != 0 {
// 			ports = append(ports, springApplicationYaml.Server.Port)
// 		}
// 	}

// 	// If we couldnt find a java version up to this point , we use default from config
// 	if javaVersion == "" {
// 		javaVersion = t.JavaConfig.JavaVersion
// 	}

// 	ct := transformertypes.TransformerPlan{
// 		Mode:              transformertypes.ModeContainer,
// 		ArtifactTypes:     []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
// 		BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
// 		Configs: map[transformertypes.ConfigType]interface{}{
// 			javaServiceConfigType: JavaConfig{
// 				ServiceName:    appName,
// 				Ports:          ports,
// 				JavaVersion:    javaVersion,
// 				AppFile:        appFile,
// 				DeploymentFile: deploymentFile,
// 				BuildTool:      buildTool,
// 				Profiles:       profiles,
// 			}},
// 		Paths: map[transformertypes.PathType][]string{
// 			mavenPomXML:                   {filepath.Join(dir, maven.PomXMLFileName)},
// 			artifacts.ProjectPathPathType: {dir},
// 			springbootApplicationFilePath: validSpringbootFiles,
// 		},
// 	}

// 	return map[string]transformertypes.ServicePlan{appName: {ct}}, nil, nil
// }

// // Transform transforms the artifacts
// func (t *JavaAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
// 	pathMappings := []transformertypes.PathMapping{}
// 	createdArtifacts := []transformertypes.Artifact{}
// 	for _, a := range newArtifacts {
// 		if a.Artifact != artifacts.ServiceArtifactType {
// 			continue
// 		}

// 		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
// 		if err != nil {
// 			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
// 		}
// 		var sConfig JavaConfig
// 		err = a.GetConfig(javaServiceConfigType, &sConfig)
// 		if err != nil {
// 			logrus.Errorf("unable to load config for Transformer into %T : %s", sConfig, err)
// 			continue
// 		}
// 		var seConfig artifacts.ServiceConfig
// 		err = a.GetConfig(artifacts.ServiceConfigType, &seConfig)
// 		if err != nil {
// 			logrus.Errorf("unable to load config for Transformer into %T : %s", seConfig, err)
// 			continue
// 		}
// 		sImageName := artifacts.ImageName{}
// 		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
// 		if err != nil {
// 			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
// 		}
// 		if sImageName.ImageName == "" {
// 			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
// 		}

// 		// License
// 		strLicense, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license"))
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		// Build

// 		buildSegment := ""
// 		if sConfig.BuildTool == "maven" {
// 			buildSegment = "Dockerfile.maven-build"
// 		} else if sConfig.BuildTool == "gradle" {
// 			buildSegment = "Dockerfile.gradle-build"
// 		} else {
// 			logrus.Errorf("Unable to set the buildSegment file")
// 			continue
// 		}

// 		strBuild, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, buildSegment))
// 		if err != nil {
// 			return nil, nil, err
// 		}

// 		// Runtime
// 		// Pablo: we will split this part.
// 		//runtimeSegment := "Dockerfile.springboot-embedded" // default
// 		//if sConfig.ApplicationServer == "jboss/wildfly" {
// 		//	runtimeSegment = "Dockerfile.springboot-wildfly-jboss-runtime"
// 		//} else if sConfig.ApplicationServer == "openliberty/open-liberty" {
// 		//	runtimeSegment = "Dockerfile.springboot-open-liberty-runtime"
// 		//}

// 		//strRuntime, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, runtimeSegment))
// 		//if err != nil {
// 		//	return nil, nil, err
// 		//}

// 		var outputPath = filepath.Join(t.Env.TempPath, "Dockerfile.template")
// 		//template := string(strLicense) + "\n" + string(strBuild) + "\n" + string(strRuntime)
// 		// Pablo: Just condider license and build
// 		template := string(strLicense) + "\n" + string(strBuild)
// 		err = ioutil.WriteFile(outputPath, []byte(template), 0644)
// 		if err != nil {
// 			logrus.Errorf("Could not write the single generated Dockerfile template: %s", err)
// 		}

// 		port := 8080
// 		if len(sConfig.Ports) > 0 {
// 			port = sConfig.Ports[0]
// 		}

// 		dfp := filepath.Join(common.DefaultSourceDir, relSrcPath, "Dockerfile")
// 		pathMappings = append(pathMappings, transformertypes.PathMapping{
// 			Type:     transformertypes.TemplatePathMappingType,
// 			SrcPath:  outputPath,
// 			DestPath: dfp,
// 			TemplateConfig: JavaBuildTemplateConfig{
// 				JavaInstallPackageName: sConfig.JavaPackageName,
// 				AppServerImage:         sConfig.ApplicationServerImage,
// 				Port:                   port,
// 				AppFile:                sConfig.AppFile,
// 				DeploymentFile:         sConfig.DeploymentFile,
// 			},
// 		}, transformertypes.PathMapping{
// 			Type:     transformertypes.SourcePathMappingType,
// 			SrcPath:  "",
// 			DestPath: common.DefaultSourceDir,
// 		})

// 		/*
// 			// not using it
// 			p := transformertypes.Artifact{
// 				Name:     sImageName.ImageName,
// 				Artifact: artifacts.DockerfileArtifactType,
// 				Paths: map[string][]string{
// 					artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
// 					artifacts.DockerfilePathType:  {dfp},
// 				},
// 				Configs: map[string]interface{}{
// 					artifacts.ImageNameConfigType: sImageName,
// 				},
// 			}

// 			// not using it
// 			dfs := transformertypes.Artifact{
// 				Name:     sConfig.ServiceName,
// 				Artifact: artifacts.DockerfileForServiceArtifactType,
// 				Paths: map[string][]string{
// 					artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
// 					artifacts.DockerfilePathType:  {dfp},
// 				},
// 				Configs: map[string]interface{}{
// 					artifacts.ImageNameConfigType: sImageName,
// 					artifacts.ServiceConfigType:   sConfig,
// 				},
// 			}
// 		*/

// 		//new
// 		buildArtifact := transformertypes.Artifact{
// 			Name:     sImageName.ImageName + "-build",
// 			Artifact: artifacts.JarArtifactType,
// 			//In here we store the current path of the current Dockerfile template
// 			Paths: map[string][]string{
// 				artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
// 				artifacts.DockerfilePathType:  {dfp},
// 			},
// 			Configs: map[string]interface{}{
// 				"targetAppData": JavaBuildTemplateConfig{
// 					JavaInstallPackageName: sConfig.JavaPackageName,
// 					AppServerImage:         sConfig.ApplicationServerImage,
// 					Port:                   port,
// 					AppFile:                sConfig.AppFile,
// 					DeploymentFile:         sConfig.DeploymentFile,
// 					AppServer:              sConfig.ApplicationServer,
// 					BuildPath:              dfp,
// 					BuildOutputPath:        outputPath,
// 				},
// 			},
// 		}
// 		// new
// 		/*
// 			runtimeDataArtifact := transformertypes.Artifact{
// 				Name:     sImageName.ImageName + "-runtime",
// 				Artifact: artifacts.RuntimeType,
// 				//In here we store the current path of the current Dockerfile template

// 				Configs: map[string]interface{}{
// 					"targetAppData": SpringbootTemplateConfig{
// 						JavaPackageName: sConfig.JavaPackageName,
// 						AppServerImage:  sConfig.ApplicationServerImage,
// 						Port:            port,
// 						AppFile:         sConfig.AppFile,
// 						DeploymentFile:  sConfig.DeploymentFile,
// 					},
// 				},
// 			}
// 		*/

// 		//createdArtifacts = append(createdArtifacts, p, dfs) // original
// 		createdArtifacts = append(createdArtifacts, buildArtifact)

// 	}

// 	return pathMappings, createdArtifacts, nil

// 	// changes:
// 	// - just artifacts: jar/war/ + params to run it
// }
