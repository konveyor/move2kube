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
RUN yum install -y java-1.8.0-openjdk-devel
RUN yum install -y wget
RUN yum install -y unzip
RUN wget https://mirrors.estointernet.in/apache//ant/binaries/apache-ant-1.10.8-bin.zip -P /tmp
RUN unzip -d /opt/apache-ant /tmp/apache-ant-1.10.8-bin.zip
ENV PATH="$PATH:/opt/apache-ant/apache-ant-1.10.8/bin/"
COPY . /{{ .app_name }}
WORKDIR /{{ .app_name }}
RUN {{ .ant_cmd }}

FROM registry.access.redhat.com/jboss-eap-6/eap64-openshift:latest
EXPOSE {{ .port }}
COPY --from=build_base /{{ .app_name }}/output/{{ .app_name }}.ear /opt/eap/standalone/deployments/
