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

# Build App
FROM registry.access.redhat.com/ubi8/ubi:latest AS build_base
RUN yum install -y java-1.8.0-openjdk-devel
RUN yum install -y wget
RUN yum install -y unzip
RUN wget https://services.gradle.org/distributions/gradle-6.6-bin.zip -P /tmp
RUN unzip -d /opt/gradle /tmp/gradle-6.6-bin.zip
ENV PATH="$PATH:/opt/gradle/gradle-6.6/bin/"
COPY . /{{ .app_name }}
WORKDIR /{{ .app_name }}
RUN gradle build

# Run App
FROM registry.access.redhat.com/jboss-eap-6/eap64-openshift:latest
EXPOSE {{ .port }}
COPY --from=build_base /{{ .app_name }}/build/libs/* /opt/eap/standalone/deployments/