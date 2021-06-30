#   Copyright IBM Corporation 2020
#
#   Licensed under the Apache License, Version 2.0 (the "License");
#   you may not use this file except in compliance with the License.
#   You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#   Unless required by applicable law or agreed to in writing, software
#   distributed under the License is distributed on an "AS IS" BASIS,
#   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#   See the License for the specific language governing permissions and
#   limitations under the License.

FROM registry.access.redhat.com/ubi8/ubi:latest AS build_base
RUN yum install -y {{ .java_version}}
RUN yum install -y maven
COPY . /app
WORKDIR /app
RUN mvn install

FROM {{ .jboss_image}}
{{range $port := .port}}EXPOSE {{$port}}
{{end}}
# Copy the war file to the deployments folder
COPY --from=build_base  /app/target/{{ .app_file }} {{ .deployment_path }}{{ .deployment_file }}

