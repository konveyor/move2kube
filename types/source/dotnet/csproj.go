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

package dotnet

import (
	"encoding/xml"
	"io"
	"regexp"
)

var (
	// Version4And3_5 is the regex to match against specific Dot Net framework versions
	Version4And3_5 = regexp.MustCompile(`(v4|v3\.5)`)
	// WebLib is the key library used in web applications
	WebLib = regexp.MustCompile(`^System\.Web`)
	// AspNetWebLib is the key library used in asp net web applications
	AspNetWebLib = regexp.MustCompile("AspNet")
	// WebSLLib is the key library used in web sliverlight applications
	WebSLLib = regexp.MustCompile(`Silverlight\.js`)
	// ProjBlockRegex pattern
	ProjBlockRegex = regexp.MustCompile(`(?m)^Project\([^)]+\)[^,]+,\s*\"([^"]+)\"`)
)

const (
	// VISUAL_STUDIO_SOLUTION_FILE_EXT is the file extension for a file Visual Studio uses to store project metadata
	// https://docs.microsoft.com/en-us/visualstudio/extensibility/internals/solution-dot-sln-file?view=vs-2022
	VISUAL_STUDIO_SOLUTION_FILE_EXT = ".sln"

	// DefaultBaseImageVersion is the default base image version tag
	DefaultBaseImageVersion = "4.8"
)

// CSProj defines the .csproj file
type CSProj struct {
	XMLName        xml.Name        `xml:"Project"`
	Sdk            string          `xml:"Sdk,attr"`
	PropertyGroups []PropertyGroup `xml:"PropertyGroup"`
	ItemGroups     []ItemGroup     `xml:"ItemGroup"`
}

// ItemGroup is defined in .csproj file to list items used in the project
type ItemGroup struct {
	XMLName           xml.Name           `xml:"ItemGroup"`
	PackageReferences []PackageReference `xml:"PackageReference"`
	References        []Reference        `xml:"Reference"`
	Contents          []Content          `xml:"Content"`
	None              []None             `xml:"None"`
}

// Reference is used in .csproj files to list dependencies
type Reference struct {
	XMLName xml.Name `xml:"Reference"`
	Include string   `xml:"Include,attr"`
}

// PackageReference is used in .csproj files to list dependencies
type PackageReference struct {
	XMLName xml.Name `xml:"PackageReference"`
	Include string   `xml:"Include,attr"`
	Version string   `xml:"Version,attr"`
}

// Content defined in .csproj to define items used in the project
type Content struct {
	XMLName xml.Name `xml:"Content"`
	Include string   `xml:"Include,attr"`
}

// None is defined in .csproj to define items used in the project
type None struct {
	XMLName xml.Name `xml:"None"`
	Include string   `xml:"Include,attr"`
}

// PropertyGroup is defined in .csproj file to list properties of the project
type PropertyGroup struct {
	XMLName                xml.Name    `xml:"PropertyGroup"`
	Condition              string      `xml:"Condition,attr"`
	TargetFramework        string      `xml:"TargetFramework"`
	TargetFrameworkVersion string      `xml:"TargetFrameworkVersion"`
	OutputPath             string      `xml:"OutputPath"`
	Properties             *Properties `xml:"properties,omitempty"`
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
