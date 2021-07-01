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

package springboot

type SpringApplicationYaml struct {
	Spring Spring `yaml:"spring,omitempty"`
	Server Server `yaml:"server,omitempty"`
}

type Server struct {
	Port int `yaml:"port,omitempty"`
}

type Spring struct {
	SpringApplication SpringApplication `yaml:"application,omitempty"`
}
type SpringApplication struct {
	Name string `yaml:"name,omitempty"`
}
