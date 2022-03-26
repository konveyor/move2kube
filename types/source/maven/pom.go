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

package maven

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"

	"github.com/konveyor/move2kube/common"
	"github.com/sirupsen/logrus"
)

// PomXMLFileName represents the name of the POM File
const PomXMLFileName string = "pom.xml"

var (
	propVar = regexp.MustCompile(`\${(.+)}`)
)

// Pom defines pom.xml data
type Pom struct {
	XMLName                xml.Name                `xml:"project,omitempty"`
	ModelVersion           string                  `xml:"modelVersion,omitempty"`
	Parent                 *Parent                 `xml:"parent,omitempty"`
	GroupID                string                  `xml:"groupId,omitempty"`
	ArtifactID             string                  `xml:"artifactId,omitempty"`
	Version                string                  `xml:"version,omitempty"`
	Packaging              string                  `xml:"packaging,omitempty"`
	Name                   string                  `xml:"name,omitempty"`
	Description            string                  `xml:"description,omitempty"`
	URL                    string                  `xml:"url,omitempty"`
	InceptionYear          string                  `xml:"inceptionYear,omitempty"`
	Organization           *Organization           `xml:"organization,omitempty"`
	Licenses               *[]License              `xml:"licenses>license,omitempty"`
	Developers             *[]Developer            `xml:"developers>developer,omitempty"`
	Contributors           *[]Contributor          `xml:"contributors>contributor,omitempty"`
	MailingLists           *[]MailingList          `xml:"mailingLists>mailingList,omitempty"`
	Prerequisites          *Prerequisites          `xml:"prerequisites,omitempty"`
	Modules                *[]string               `xml:"modules>module,omitempty"`
	SCM                    *Scm                    `xml:"scm,omitempty"`
	IssueManagement        *IssueManagement        `xml:"issueManagement,omitempty"`
	CIManagement           *CIManagement           `xml:"ciManagement,omitempty"`
	DistributionManagement *DistributionManagement `xml:"distributionManagement,omitempty"`
	DependencyManagement   *DependencyManagement   `xml:"dependencyManagement,omitempty"`
	Dependencies           *[]Dependency           `xml:"dependencies>dependency,omitempty"`
	Repositories           *[]Repository           `xml:"repositories>repository,omitempty"`
	PluginRepositories     *[]PluginRepository     `xml:"pluginRepositories>pluginRepository,omitempty"`
	Build                  *Build                  `xml:"build,omitempty"`
	Reporting              *Reporting              `xml:"reporting,omitempty"`
	Profiles               *[]Profile              `xml:"profiles>profile,omitempty"`
	Properties             *Properties             `xml:"properties,omitempty"`
}

// Load loads a pom xml file
func (pom *Pom) Load(file string) error {
	err := common.ReadXML(file, pom)
	if err != nil {
		logrus.Errorf("Unable to unmarshal pom file (%s) : %s", file, err)
		return err
	}
	return nil
}

// GetProperty returns the property value of a property
func (pom *Pom) GetProperty(key string) (val string, err error) {
	if pom.Properties == nil {
		return "", fmt.Errorf("property not found in pom")
	}
	val, ok := pom.Properties.Entries[key]
	if !ok {
		return "", fmt.Errorf("property not found in pom")
	}
	if fullMatches := propVar.FindAllSubmatchIndex([]byte(val), -1); len(fullMatches) > 0 {
		newVal := ""
		prevMatch := 0
		for _, fullMatch := range fullMatches {
			newVal += val[prevMatch:fullMatch[0]]
			if len(fullMatch) <= 1 {
				logrus.Errorf("Unable to find variable name in pom property : %s", val)
				continue
			}
			propVal, err := pom.GetProperty(val[fullMatch[2]:fullMatch[3]])
			if err != nil {
				logrus.Errorf("Unable to read property in pom : %s", err)
				return "", err
			}
			newVal += propVal
			prevMatch = fullMatch[2]
		}
		val = newVal
	}
	return val, nil
}

// Entries defines a pom.xml entry
type Entries map[string]string

// Properties defines pom.xml properties
type Properties struct {
	Entries Entries
}

// UnmarshalXML unmarshals XML
func (p *Properties) UnmarshalXML(d *xml.Decoder, start xml.StartElement) (err error) {
	type entry struct {
		XMLName xml.Name
		Key     string `xml:"name,attr,omitempty"`
		Value   string `xml:",chardata"`
	}
	e := entry{}
	p.Entries = map[string]string{}
	for err = d.Decode(&e); err == nil; err = d.Decode(&e) {
		e.Key = e.XMLName.Local
		p.Entries[e.Key] = e.Value
	}
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}

// MarshalXML marshals XML
func (entries Entries) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	for key, value := range entries {
		t := xml.StartElement{Name: xml.Name{Space: "", Local: key}}
		tokens = append(tokens, t, xml.CharData(value), xml.EndElement{Name: t.Name})
	}
	tokens = append(tokens, xml.EndElement{Name: start.Name})
	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}
	// flush to ensure tokens are written
	return e.Flush()
}

// Parent defines a pom.xml parent
type Parent struct {
	GroupID      string `xml:"groupId,omitempty"`
	ArtifactID   string `xml:"artifactId,omitempty"`
	Version      string `xml:"version,omitempty"`
	RelativePath string `xml:"relativePath,omitempty"`
}

// Organization defines a pom.xml Organization
type Organization struct {
	Name string `xml:"name,omitempty"`
	URL  string `xml:"url,omitempty"`
}

// License defines a pom.xml License
type License struct {
	Name         string `xml:"name,omitempty"`
	URL          string `xml:"url,omitempty"`
	Distribution string `xml:"distribution,omitempty"`
	Comments     string `xml:"comments,omitempty"`
}

// Developer defines a pom.xml Developer
type Developer struct {
	ID              string     `xml:"id,omitempty"`
	Name            string     `xml:"name,omitempty"`
	Email           string     `xml:"email,omitempty"`
	URL             string     `xml:"url,omitempty"`
	Organization    string     `xml:"organization,omitempty"`
	OrganizationURL string     `xml:"organizationUrl,omitempty"`
	Roles           *[]string  `xml:"roles>role,omitempty"`
	Timezone        string     `xml:"timezone,omitempty"`
	Properties      Properties `xml:"properties,omitempty"`
}

// Contributor defines a pom.xml Contributor
type Contributor struct {
	Name            string     `xml:"name,omitempty"`
	Email           string     `xml:"email,omitempty"`
	URL             string     `xml:"url,omitempty"`
	Organization    string     `xml:"organization,omitempty"`
	OrganizationURL string     `xml:"organizationUrl,omitempty"`
	Roles           *[]string  `xml:"roles>role,omitempty"`
	Timezone        string     `xml:"timezone,omitempty"`
	Properties      Properties `xml:"properties,omitempty"`
}

// MailingList defines a pom.xml MailingList
type MailingList struct {
	Name          string    `xml:"name,omitempty"`
	Subscribe     string    `xml:"subscribe,omitempty"`
	Unsubscribe   string    `xml:"unsubscribe,omitempty"`
	Post          string    `xml:"post,omitempty"`
	Archive       string    `xml:"archive,omitempty"`
	OtherArchives *[]string `xml:"otherArchives>otherArchive,omitempty"`
}

// Prerequisites defines a pom.xml Prerequisites
type Prerequisites struct {
	Maven string `xml:"maven,omitempty"`
}

// Scm defines a pom.xml Scm
type Scm struct {
	Connection          string `xml:"connection,omitempty"`
	DeveloperConnection string `xml:"developerConnection,omitempty"`
	Tag                 string `xml:"tag,omitempty"`
	URL                 string `xml:"url,omitempty"`
}

// IssueManagement defines a pom.xml IssueManagement
type IssueManagement struct {
	System string `xml:"system,omitempty"`
	URL    string `xml:"url,omitempty"`
}

// CIManagement defines a pom.xml CIManagement
type CIManagement struct {
	System    string     `xml:"system,omitempty"`
	URL       string     `xml:"url,omitempty"`
	Notifiers []Notifier `xml:"notifiers>notifier,omitempty"`
}

// Notifier defines a pom.xml Notifier
type Notifier struct {
	Type          string     `xml:"type,omitempty"`
	SendOnError   bool       `xml:"sendOnError,omitempty"`
	SendOnFailure bool       `xml:"sendOnFailure,omitempty"`
	SendOnSuccess bool       `xml:"sendOnSuccess,omitempty"`
	SendOnWarning bool       `xml:"sendOnWarning,omitempty"`
	Address       string     `xml:"address,omitempty"`
	Configuration Properties `xml:"configuration,omitempty"`
}

// DistributionManagement defines a pom.xml DistributionManagement
type DistributionManagement struct {
	Repository         Repository `xml:"repository,omitempty"`
	SnapshotRepository Repository `xml:"snapshotRepository,omitempty"`
	Site               Site       `xml:"site,omitempty"`
	DownloadURL        string     `xml:"downloadUrl,omitempty"`
	Relocation         Relocation `xml:"relocation,omitempty"`
	Status             string     `xml:"status,omitempty"`
}

// Site defines a pom.xml Site
type Site struct {
	ID   string `xml:"id,omitempty"`
	Name string `xml:"name,omitempty"`
	URL  string `xml:"url,omitempty"`
}

// Relocation defines a pom.xml Relocation
type Relocation struct {
	GroupID    string `xml:"groupId,omitempty"`
	ArtifactID string `xml:"artifactId,omitempty"`
	Version    string `xml:"version,omitempty"`
	Message    string `xml:"message,omitempty"`
}

// DependencyManagement defines a pom.xml DependencyManagement
type DependencyManagement struct {
	Dependencies *[]Dependency `xml:"dependencies>dependency,omitempty"`
}

// Dependency defines a pom.xml Dependency
type Dependency struct {
	GroupID    string       `xml:"groupId,omitempty"`
	ArtifactID string       `xml:"artifactId,omitempty"`
	Version    string       `xml:"version,omitempty"`
	Type       string       `xml:"type,omitempty"`
	Classifier string       `xml:"classifier,omitempty"`
	Scope      string       `xml:"scope,omitempty"`
	SystemPath string       `xml:"systemPath,omitempty"`
	Exclusions *[]Exclusion `xml:"exclusions>exclusion,omitempty"`
	Optional   string       `xml:"optional,omitempty"`
}

// Exclusion defines a pom.xml Exclusion
type Exclusion struct {
	ArtifactID string `xml:"artifactId,omitempty"`
	GroupID    string `xml:"groupId,omitempty"`
}

// Repository defines a pom.xml Repository
type Repository struct {
	UniqueVersion bool              `xml:"uniqueVersion,omitempty"`
	Releases      *RepositoryPolicy `xml:"releases,omitempty"`
	Snapshots     *RepositoryPolicy `xml:"snapshots,omitempty"`
	ID            string            `xml:"id,omitempty"`
	Name          string            `xml:"name,omitempty"`
	URL           string            `xml:"url,omitempty"`
	Layout        string            `xml:"layout,omitempty"`
}

// RepositoryPolicy defines a pom.xml RepositoryPolicy
type RepositoryPolicy struct {
	Enabled        string `xml:"enabled,omitempty"`
	UpdatePolicy   string `xml:"updatePolicy,omitempty"`
	ChecksumPolicy string `xml:"checksumPolicy,omitempty"`
}

// PluginRepository defines a pom.xml PluginRepository
type PluginRepository struct {
	Releases  *RepositoryPolicy `xml:"releases,omitempty"`
	Snapshots *RepositoryPolicy `xml:"snapshots,omitempty"`
	ID        string            `xml:"id,omitempty"`
	Name      string            `xml:"name,omitempty"`
	URL       string            `xml:"url,omitempty"`
	Layout    string            `xml:"layout,omitempty"`
}

// BuildBase defines a pom.xml BuildBase
type BuildBase struct {
	DefaultGoal      string            `xml:"defaultGoal,omitempty"`
	Resources        *[]Resource       `xml:"resources>resource,omitempty"`
	TestResources    *[]Resource       `xml:"testResources>testResource,omitempty"`
	Directory        string            `xml:"directory,omitempty"`
	FinalName        string            `xml:"finalName,omitempty"`
	Filters          *[]string         `xml:"filters>filter,omitempty"`
	PluginManagement *PluginManagement `xml:"pluginManagement,omitempty"`
	Plugins          *[]Plugin         `xml:"plugins>plugin,omitempty"`
}

// Build defines a pom.xml Build
type Build struct {
	SourceDirectory       string       `xml:"sourceDirectory,omitempty"`
	ScriptSourceDirectory string       `xml:"scriptSourceDirectory,omitempty"`
	TestSourceDirectory   string       `xml:"testSourceDirectory,omitempty"`
	OutputDirectory       string       `xml:"outputDirectory,omitempty"`
	TestOutputDirectory   string       `xml:"testOutputDirectory,omitempty"`
	Extensions            *[]Extension `xml:"extensions>extension,omitempty"`
	BuildBase
}

// Extension defines a pom.xml Extension
type Extension struct {
	GroupID    string `xml:"groupId,omitempty"`
	ArtifactID string `xml:"artifactId,omitempty"`
	Version    string `xml:"version,omitempty"`
}

// Resource defines a pom.xml Resource
type Resource struct {
	TargetPath string    `xml:"targetPath,omitempty"`
	Filtering  string    `xml:"filtering,omitempty"`
	Directory  string    `xml:"directory,omitempty"`
	Includes   *[]string `xml:"includes>include,omitempty"`
	Excludes   *[]string `xml:"excludes>exclude,omitempty"`
}

// PluginManagement defines a pom.xml PluginManagement
type PluginManagement struct {
	Plugins *[]Plugin `xml:"plugins>plugin,omitempty"`
}

// Plugin defines a pom.xml Plugin
type Plugin struct {
	GroupID       string             `xml:"groupId,omitempty"`
	ArtifactID    string             `xml:"artifactId,omitempty"`
	Version       string             `xml:"version,omitempty"`
	Extensions    string             `xml:"extensions,omitempty"`
	Executions    *[]PluginExecution `xml:"executions>execution,omitempty"`
	Dependencies  *[]Dependency      `xml:"dependencies>dependency,omitempty"`
	Inherited     string             `xml:"inherited,omitempty"`
	Configuration Configuration      `xml:"configuration,omitempty"`
}

// Configuration defines a pom.xml Configuration
// TODO: change this to map[string]interface{} because different plugins have different configuration
// https://maven.apache.org/guides/mini/guide-configuring-plugins.html
type Configuration struct {
	Classifier            string    `xml:"classifier,omitempty"`
	Source                string    `xml:"source,omitempty"`
	Target                string    `xml:"target,omitempty"`
	ConfigurationProfiles *[]string `xml:"profiles>profile,omitempty"`
}

// PluginExecution defines a pom.xml PluginExecution
type PluginExecution struct {
	ID        string    `xml:"id,omitempty"`
	Phase     string    `xml:"phase,omitempty"`
	Goals     *[]string `xml:"goals>goal,omitempty"`
	Inherited string    `xml:"inherited,omitempty"`
}

// Reporting defines a pom.xml Reporting
type Reporting struct {
	ExcludeDefaults string             `xml:"excludeDefaults,omitempty"`
	OutputDirectory string             `xml:"outputDirectory,omitempty"`
	Plugins         *[]ReportingPlugin `xml:"plugins>plugin,omitempty"`
}

// ReportingPlugin defines a pom.xml ReportingPlugin
type ReportingPlugin struct {
	GroupID    string       `xml:"groupId,omitempty"`
	ArtifactID string       `xml:"artifactId,omitempty"`
	Version    string       `xml:"version,omitempty"`
	Inherited  string       `xml:"inherited,omitempty"`
	ReportSets *[]ReportSet `xml:"reportSets>reportSet,omitempty"`
}

// ReportSet defines a pom.xml ReportSet
type ReportSet struct {
	ID        string    `xml:"id,omitempty"`
	Reports   *[]string `xml:"reports>report,omitempty"`
	Inherited string    `xml:"inherited,omitempty"`
}

// Profile defines a pom.xml Profile
type Profile struct {
	ID                     string                  `xml:"id,omitempty"`
	Activation             *Activation             `xml:"activation,omitempty"`
	Build                  *BuildBase              `xml:"build,omitempty"`
	Modules                *[]string               `xml:"modules>module,omitempty"`
	DistributionManagement *DistributionManagement `xml:"distributionManagement,omitempty"`
	Properties             *Properties             `xml:"properties,omitempty"`
	DependencyManagement   *DependencyManagement   `xml:"dependencyManagement,omitempty"`
	Dependencies           *[]Dependency           `xml:"dependencies>dependency,omitempty"`
	Repositories           *[]Repository           `xml:"repositories>repository,omitempty"`
	PluginRepositories     *[]PluginRepository     `xml:"pluginRepositories>pluginRepository,omitempty"`
	Reporting              *Reporting              `xml:"reporting,omitempty"`
}

// Activation defines a pom.xml Activation
type Activation struct {
	ActiveByDefault bool                `xml:"activeByDefault,omitempty"`
	JDK             string              `xml:"jdk,omitempty"`
	OS              *ActivationOS       `xml:"os,omitempty"`
	Property        *ActivationProperty `xml:"property,omitempty"`
	File            *ActivationFile     `xml:"file,omitempty"`
}

// ActivationOS defines a pom.xml ActivationOS
type ActivationOS struct {
	Name    string `xml:"name,omitempty"`
	Family  string `xml:"family,omitempty"`
	Arch    string `xml:"arch,omitempty"`
	Version string `xml:"version,omitempty"`
}

// ActivationProperty defines a pom.xml ActivationProperty
type ActivationProperty struct {
	Name  string `xml:"name,omitempty"`
	Value string `xml:"value,omitempty"`
}

// ActivationFile defines a pom.xml ActivationFile
type ActivationFile struct {
	Missing string `xml:"missing,omitempty"`
	Exists  string `xml:"exists,omitempty"`
}
