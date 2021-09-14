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

import "encoding/xml"

// AppConfig defines the app.config
type AppConfig struct {
	XMLName        xml.Name       `xml:"configuration"`
	Model          ServiceModel   `xml:"system.serviceModel"`
	AppCfgSettings AppCfgSettings `xml:"appSettings"`
}

// ServiceModel defines list of service models
type ServiceModel struct {
	XMLName  xml.Name `xml:"system.serviceModel"`
	Services Services `xml:"services"`
}

// Services defines list of services
type Services struct {
	XMLName     xml.Name  `xml:"services"`
	ServiceList []Service `xml:"service"`
}

// Service defines a service property list
type Service struct {
	XMLName xml.Name `xml:"service"`
	Host    Host     `xml:"host"`
}

// Host defines a host exposed by service
type Host struct {
	XMLName       xml.Name      `xml:"host"`
	BaseAddresses BaseAddresses `xml:"baseAddresses"`
}

// BaseAddresses defines list of base addresses
type BaseAddresses struct {
	XMLName xml.Name  `xml:"baseAddresses"`
	AddList []AddKeys `xml:"add"`
}

// AppCfgSettings defines the settings
type AppCfgSettings struct {
	XMLName xml.Name  `xml:"appSettings"`
	AddList []AddKeys `xml:"add"`
}

// AddKeys defines the key list
type AddKeys struct {
	XMLName     xml.Name `xml:"add"`
	BaseAddress string   `xml:"baseAddress,attr"`
	Key         string   `xml:"key,attr"`
	Value       string   `xml:"value,attr"`
}
