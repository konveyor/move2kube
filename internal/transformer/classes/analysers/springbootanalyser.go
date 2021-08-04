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

package analysers

import (
	"encoding/xml"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/konveyor/move2kube/environment"
	"github.com/konveyor/move2kube/internal/common"
	collectiontypes "github.com/konveyor/move2kube/types/collection"
	irtypes "github.com/konveyor/move2kube/types/ir"
	"github.com/konveyor/move2kube/types/source/maven"
	"github.com/konveyor/move2kube/types/source/springboot"
	transformertypes "github.com/konveyor/move2kube/types/transformer"
	"github.com/konveyor/move2kube/types/transformer/artifacts"
	"github.com/sirupsen/logrus"
)

const pomXML string = "pom.xml"

const (
	springbootServiceConfigType transformertypes.ConfigType = "SpringbootService"
)

const (
	mavenPomXML         transformertypes.PathType = "MavenPomXML"
	applicationFilePath transformertypes.PathType = "SpringbootApplicationFile"
)

// SpringbootAnalyser implements Transformer interface
type SpringbootAnalyser struct {
	Config transformertypes.Transformer
	Env    *environment.Environment
}

// SpringbootConfig defines SpringbootConfig properties
type SpringbootConfig struct {
	ServiceName            string `yaml:"serviceName,omitempty"`
	Ports                  []int  `yaml:"ports,omitempty"`
	JavaVersion            string `yaml:"javaVersion,omitempty"`
	ApplicationServer      string `yaml:"applicationServer,omitempty"`
	ApplicationServerImage string `yaml:"applicationServerImage,omitempty"`
	JavaPackageName        string `yaml:"javaPackageName,omitempty"`
	AppFile                string `yaml:"appFile,omitempty"`
	DeploymentFile         string `yaml:"deploymentFile,omitempty"`
}

// SpringbootTemplateConfig defines SpringbootTemplateConfig properties
type SpringbootTemplateConfig struct {
	Port            int    `yaml:"port,omitempty"`
	JavaPackageName string `yaml:"javaPackageName,omitempty"`
	AppServerImage  string `yaml:"appServerImage,omitempty"`
	AppFile         string `yaml:"appFile,omitempty"`
	DeploymentFile  string `yaml:"deploymentFile,omitempty"`
}

// Init Initializes the transformer
func (t *SpringbootAnalyser) Init(tc transformertypes.Transformer, env *environment.Environment) (err error) {
	t.Config = tc
	t.Env = env
	return nil
}

// GetConfig returns the transformer config
func (t *SpringbootAnalyser) GetConfig() (transformertypes.Transformer, *environment.Environment) {
	return t.Config, t.Env
}

// BaseDirectoryDetect runs detect in base directory
func (t *SpringbootAnalyser) BaseDirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	return nil, nil, nil
}

// DirectoryDetect runs detect in each sub directory
func (t *SpringbootAnalyser) DirectoryDetect(dir string) (namedServices map[string]transformertypes.ServicePlan, unnamedServices []transformertypes.TransformerPlan, err error) {
	destEntries, err := ioutil.ReadDir(dir)
	if err != nil {
		logrus.Errorf("Unable to process directory %s : %s", dir, err)
		return nil, nil, err
	}
	pomFound := false
	for _, de := range destEntries {
		if de.Name() == pomXML {
			pomFound = true
			break
		}
	}

	if !pomFound {
		return nil, nil, nil
	}

	// filled with previously declared xml
	pomStr, err := ioutil.ReadFile(filepath.Join(dir, pomXML))
	if err != nil {
		logrus.Errorf("Could not read the pom.xml file: %s", err)
		return nil, nil, err
	}

	// Load pom from string
	var pom maven.Pom
	if err := xml.Unmarshal([]byte(pomStr), &pom); err != nil {
		logrus.Errorf("unable to unmarshal pom file. Reason: %s", err)
		return nil, nil, err
	}

	// Dont process if this is a root pom and there are submodules
	if pom.Modules != nil && len(*(pom.Modules)) != 0 {
		logrus.Debugf("Ignoring pom at %s as it has modules", dir)
		return nil, nil, nil
	}

	// Check the dependencies block in case it exists

	isSpringboot := false
	isTomcatProvided := false

	if pom.Dependencies == nil {
		logrus.Debugf("POM file at %s does not contain a dependencies block", dir)
	} else {

		for _, dependency := range *pom.Dependencies {

			if strings.Contains(dependency.GroupID, "org.springframework.boot") {
				isSpringboot = true
			}

			if strings.Contains(dependency.ArtifactID, "spring-boot-starter-tomcat") && dependency.Scope == "provided" {
				isTomcatProvided = true
			}
		}
	}

	logrus.Debugf("Is springboot app: ", isSpringboot)

	// Collect packaging from the packaging block
	packaging := ""
	if pom.Packaging == "" {
		logrus.Debugf("Pom at %s does not contain a Packaging block", dir)
	} else {
		packaging = pom.Packaging
		logrus.Debugf("Packaging: %s", packaging)
	}

	// Collect java / tomcat version fom the Properties block
	javaVersion := ""
	tomcatVersion := ""
	if pom.Properties == nil {
		logrus.Debugf("Pom at %s  does not contain a Properties block", dir)
	} else {

		for k, v := range pom.Properties.Entries {
			switch k {
			// Only for springboot apps
			case "java.version":
				javaVersion = v
			case "tomcat.version":
				tomcatVersion = v
			// Non springboot apps:
			case "maven.compiler.target":
				javaVersion = v
			}
		}
	}

	logrus.Debugf("Java version %s", javaVersion)
	logrus.Debugf("Tomcat version %s", tomcatVersion)

	// Check if the application uses an embeded server or not.
	// This is based on having tomcat as `provided` and packaging as `war`
	// Initialize flag as false
	isServerEmbedded := false

	isPackagingWAR := false
	if packaging == "war" {
		isPackagingWAR = true
	}

	// here we asses for both conditions
	isServerEmbedded = !(isTomcatProvided && isPackagingWAR)

	// in the case we have standanlone java maven application (no springboot),
	// we assume the server is not embedded
	if isSpringboot == false {
		isServerEmbedded = false
	}

	// If the server is not embedded, we check if it is open-liberty or jboss/wildfly
	appServer := ""
	if !isServerEmbedded {
		// Server is not embedded. What type of server app are we using?

		// Search for server.xml files
		serverXMLfiles, err := common.GetFilesByName(dir, []string{"server.xml"})
		if err != nil {
			logrus.Debugf("Cannot get server.xml files: %s", err)
		}

		// Current assumption: if there is at least one server.xml file, -> open-liberty
		if len(serverXMLfiles) > 0 {
			appServer = "openliberty/open-liberty"
		} else {
			appServer = "jboss/wildfly"
		}
	}
	logrus.Debugf("App server: %s", appServer)

	// Check compatible image for the application server
	var appServerCandidateImages []collectiontypes.ImageInfoSpec

	if appServer != "" {
		if javaVersion == "" { // default case
			javaVersion = "1.8"
		}

		mappingPath := filepath.Join(t.Env.GetEnvironmentContext(), "mappings/java2images_tags.yaml")
		images2Data := collectiontypes.NewImagesInfo()
		if err := common.ReadMove2KubeYaml(mappingPath, &images2Data); err != nil {
			logrus.Debugf("Could not load mapping at %s", mappingPath)
		}

		for _, im := range images2Data.Spec {
			if im.Params["javaVersion"] == javaVersion && im.Params["serverApp"] == appServer {
				appServerCandidateImages = append(appServerCandidateImages, im)
			}
		}
	}

	for _, e := range appServerCandidateImages {
		logrus.Debugf("e: %s", e.Tags)
	}

	appServerImage := ""
	if len(appServerCandidateImages) > 0 {
		appServerImage = appServerCandidateImages[0].Tags[0]
	}
	logrus.Debugf("app server image %s", appServerImage)

	appPropfiles, err := common.GetFilesByName(dir, []string{"", "application.properties"})
	if err != nil {
		logrus.Debugf("Cannot get application files: %s", err)
	}
	logrus.Debugf("App prop files %s", appPropfiles)

	// Java images for build and deploy

	// build
	javaPackageNamesMappingPath := filepath.Join(t.Env.GetEnvironmentContext(), "mappings/java_version2package_name.json")
	var javaPackageNamesMapping map[string]string
	if err := common.ReadJSON(javaPackageNamesMappingPath, &javaPackageNamesMapping); err != nil {
		logrus.Debugf("Could not load mapping at %s", javaPackageNamesMappingPath)
	}

	javaPackageName := ""
	if val, ok := javaPackageNamesMapping[javaVersion]; ok {
		javaPackageName = val
	}

	// Get app file and app name
	appName := ""
	appFile := ""
	if pom.Name != "" {
		appFile = pom.Name
		appName = pom.Name
	} else {
		if pom.ArtifactID != "" {
			appFile = pom.ArtifactID
		}
		appName = filepath.Base(dir)
	}
	if appFile != "" {
		if pom.Version != "" {
			appFile = appFile + "-" + pom.Version
		}

		if pom.Packaging != "" {
			appFile = appFile + "." + pom.Packaging
		} else {
			appFile = appFile + ".jar"
		}
	}

	// Get deployment file
	deploymentFile := ""
	if pom.ArtifactID != "" {
		deploymentFile = pom.ArtifactID
	}
	if deploymentFile != "" {
		if pom.Packaging != "" {
			deploymentFile = deploymentFile + "." + pom.Packaging
		} else {
			deploymentFile = deploymentFile + ".jar"
		}
	}

	// Collect application.yml/yaml files
	appfiles, err := common.GetFilesByName(dir, []string{"application.yaml", "application.yml"})
	if err != nil {
		logrus.Debugf("Cannot get application files: %s", err)
	}

	validSpringbootFiles := []string{}
	ports := []int{}

	for _, appfile := range appfiles {
		var springApplicationYaml springboot.SpringApplicationYaml
		err = common.ReadYaml(appfile, &springApplicationYaml)
		if err != nil {
			logrus.Debugf("Could not load application file %s", appfile)
			continue
		}
		if (springApplicationYaml == springboot.SpringApplicationYaml{}) {
			logrus.Debugf("No information found in application file %s", appfile)
			continue
		}
		validSpringbootFiles = append(validSpringbootFiles, appfile)

		if springApplicationYaml.Spring.SpringApplication.Name != "" {
			appName = springApplicationYaml.Spring.SpringApplication.Name
		}

		if springApplicationYaml.Server.Port != 0 {
			ports = append(ports, springApplicationYaml.Server.Port)
		}
	}

	// If we couldnt find a java version up to this point , we assume 1.8
	if javaVersion == "" {
		javaVersion = "1.8"
	}

	ct := transformertypes.TransformerPlan{
		Mode:              transformertypes.ModeContainer,
		ArtifactTypes:     []transformertypes.ArtifactType{irtypes.IRArtifactType, artifacts.ContainerBuildArtifactType},
		BaseArtifactTypes: []transformertypes.ArtifactType{artifacts.ContainerBuildArtifactType},
		Configs: map[transformertypes.ConfigType]interface{}{
			springbootServiceConfigType: SpringbootConfig{
				ServiceName:            appName,
				Ports:                  ports,
				JavaVersion:            javaVersion,
				ApplicationServer:      appServer,
				ApplicationServerImage: appServerImage,
				JavaPackageName:        javaPackageName,
				AppFile:                appFile,
				DeploymentFile:         deploymentFile,
			}},
		Paths: map[transformertypes.PathType][]string{
			mavenPomXML:                   {filepath.Join(dir, pomXML)},
			artifacts.ProjectPathPathType: {dir},
			applicationFilePath:           validSpringbootFiles,
		},
	}

	return map[string]transformertypes.ServicePlan{appName: {ct}}, nil, nil
}

// Transform transforms the artifacts
func (t *SpringbootAnalyser) Transform(newArtifacts []transformertypes.Artifact, oldArtifacts []transformertypes.Artifact) ([]transformertypes.PathMapping, []transformertypes.Artifact, error) {
	pathMappings := []transformertypes.PathMapping{}
	createdArtifacts := []transformertypes.Artifact{}
	for _, a := range newArtifacts {
		if a.Artifact != artifacts.ServiceArtifactType {
			continue
		}

		relSrcPath, err := filepath.Rel(t.Env.GetEnvironmentSource(), a.Paths[artifacts.ProjectPathPathType][0])
		if err != nil {
			logrus.Errorf("Unable to convert source path %s to be relative : %s", a.Paths[artifacts.ProjectPathPathType][0], err)
		}
		var sConfig SpringbootConfig
		err = a.GetConfig(springbootServiceConfigType, &sConfig)
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
		sImageName := artifacts.ImageName{}
		err = a.GetConfig(artifacts.ImageNameConfigType, &sImageName)
		if err != nil {
			logrus.Debugf("unable to load config for Transformer into %T : %s", sImageName, err)
		}
		if sImageName.ImageName == "" {
			sImageName.ImageName = common.MakeStringContainerImageNameCompliant(a.Name)
		}

		// License
		strLicense, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.license"))
		if err != nil {
			return nil, nil, err
		}

		// Build
		strBuild, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, "Dockerfile.maven-build"))
		if err != nil {
			return nil, nil, err
		}

		// Runtime
		runtimeSegment := "Dockerfile.springboot-embedded" // default
		if sConfig.ApplicationServer == "jboss/wildfly" {
			runtimeSegment = "Dockerfile.springboot-wildfly-jboss-runtime"
		} else if sConfig.ApplicationServer == "openliberty/open-liberty" {
			runtimeSegment = "Dockerfile.springboot-open-liberty-runtime"
		}

		strRuntime, err := ioutil.ReadFile(filepath.Join(t.Env.GetEnvironmentContext(), t.Env.RelTemplatesDir, runtimeSegment))
		if err != nil {
			return nil, nil, err
		}

		var outputPath = filepath.Join(t.Env.TempPath, "Dockerfile.template")
		template := string(strLicense) + "\n" + string(strBuild) + "\n" + string(strRuntime)
		err = ioutil.WriteFile(outputPath, []byte(template), 0644)
		if err != nil {
			logrus.Errorf("Could not write the single generated Dockerfile template: %s", err)
		}

		port := 8080
		if len(sConfig.Ports) > 0 {
			port = sConfig.Ports[0]
		}

		dfp := filepath.Join(common.DefaultSourceDir, relSrcPath, "Dockerfile")
		pathMappings = append(pathMappings, transformertypes.PathMapping{
			Type:     transformertypes.TemplatePathMappingType,
			SrcPath:  outputPath,
			DestPath: dfp,
			TemplateConfig: SpringbootTemplateConfig{
				JavaPackageName: sConfig.JavaPackageName,
				AppServerImage:  sConfig.ApplicationServerImage,
				Port:            port,
				AppFile:         sConfig.AppFile,
				DeploymentFile:  sConfig.DeploymentFile,
			},
		}, transformertypes.PathMapping{
			Type:     transformertypes.SourcePathMappingType,
			SrcPath:  "",
			DestPath: common.DefaultSourceDir,
		})

		p := transformertypes.Artifact{
			Name:     sImageName.ImageName,
			Artifact: artifacts.DockerfileArtifactType,
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
				artifacts.DockerfilePathType:  {dfp},
			},
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
			},
		}
		dfs := transformertypes.Artifact{
			Name:     sConfig.ServiceName,
			Artifact: artifacts.DockerfileForServiceArtifactType,
			Paths: map[string][]string{
				artifacts.ProjectPathPathType: {filepath.Dir(dfp)},
				artifacts.DockerfilePathType:  {dfp},
			},
			Configs: map[string]interface{}{
				artifacts.ImageNameConfigType: sImageName,
				artifacts.ServiceConfigType:   sConfig,
			},
		}
		createdArtifacts = append(createdArtifacts, p, dfs)
	}
	return pathMappings, createdArtifacts, nil
}
