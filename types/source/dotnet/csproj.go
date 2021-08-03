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
	FourXPattern = regexp.MustCompile("v4*")
	WebLib       = regexp.MustCompile("^System.Web*")
	WebSLLib     = regexp.MustCompile("Silverlight.js")
)

const (
	ProjBlock               = "^Project"
	CsSln                   = ".sln"
	CsProj                  = ".csproj"
	DefaultBaseImageVersion = "4.8"
)

type CSProj struct {
	XMLName       xml.Name       `xml:"Project"`
	Sdk           string         `xml:"Sdk,attr"`
	PropertyGroup *PropertyGroup `xml:"PropertyGroup"`
	ItemGroups    []ItemGroup    `xml:"ItemGroup"`
}

type ItemGroup struct {
	XMLName    xml.Name    `xml:"ItemGroup"`
	References []Reference `xml:"Reference"`
	Contents   []Content   `xml:"Content"`
	None       []None   `xml:"None"`
}

type Reference struct {
	XMLName xml.Name `xml:"Reference"`
	Include string   `xml:"Include,attr"`
}

type Content struct {
	XMLName xml.Name `xml:"Content"`
	Include string   `xml:"Include,attr"`
}

type None struct {
	XMLName xml.Name `xml:"None"`
	Include string   `xml:"Include,attr"`
}

type PropertyGroup struct {
	XMLName                xml.Name    `xml:"PropertyGroup"`
	Condition              string      `xml:"Condition,attr"`
	TargetFrameworkVersion string      `xml:"TargetFrameworkVersion"`
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
