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

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3-201
RUN microdnf update && microdnf install -y java-11-openjdk-devel wget tar && microdnf clean all
WORKDIR /usr/local
RUN wget https://downloads.apache.org/tomcat/tomcat-9/v9.0.39/bin/apache-tomcat-9.0.39.tar.gz && tar -zxf apache-tomcat-9.0.39.tar.gz && rm -f apache-tomcat-9.0.39.tar.gz && mv apache-tomcat-9.0.39 tomcat9
RUN useradd -r tomcat && chown -R tomcat:tomcat tomcat9
ENV CATALINA_PID='/usr/local/tomcat9/temp/tomcat.pid' CATALINA_HOME='/usr/local/tomcat9' CATALINA_BASE='/usr/local/tomcat9'
COPY --chown=tomcat:tomcat {{ .war_path }} "$CATALINA_BASE"/webapps/
USER tomcat:tomcat
EXPOSE {{ .port }}
CMD [ "/usr/local/tomcat9/bin/catalina.sh", "run" ]
