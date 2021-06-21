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

# Builder image
FROM registry.fedoraproject.org/fedora:latest AS build_base
WORKDIR /temp
RUN dnf install -y git make findutils upx \
    && dnf clean all \
    && rm -rf /var/cache/yum
ENV GOPATH=/go
RUN mkdir -p $GOPATH/src $GOPATH/bin && chmod -R 777 $GOPATH
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH

# Download Go.
ARG GO_VERSION=1.16
RUN curl -o go.tgz "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz" \
    && tar -xzf go.tgz \
    && mv go /usr/local/ \
    && rm go.tgz

# Copy only go.mod, go.sum and download packages to allow better caching.
ENV WORKDIR=${GOPATH}/src/move2kube
WORKDIR ${WORKDIR}
COPY go.mod .
COPY go.sum .
RUN go mod download

# Build
ARG VERSION=latest
COPY . .
RUN make build
RUN cp bin/move2kube /bin/move2kube


### Run image ###
FROM registry.fedoraproject.org/fedora:latest
RUN curl -o /usr/local/bin/operator-sdk -LJO 'https://github.com/operator-framework/operator-sdk/releases/download/v1.3.0/operator-sdk_linux_amd64' \
    && chmod +x /usr/local/bin/operator-sdk
# Install utils
RUN dnf install -y findutils podman \
    && dnf clean all \
    && rm -rf /var/cache/yum
COPY containerconfig/* /etc/containers/


###################################################
#### Setting up environment for Containerizers ####
###################################################

ENV M2KCONTAINERIZER_ENV_AVAILABLE=true

# Install Java, python and utils
RUN dnf install -y \
    java-1.8.0-openjdk \
    java-1.8.0-openjdk-devel \
    unzip \
    python38 \
    && dnf clean all \
    && rm -rf /var/cache/yum \
    which
ENV JAVA_HOME /usr/lib/jvm/java-1.8.0-openjdk/

# Downloading and installing Maven
ENV MAVEN_VERSION 3.6.3
ENV BASE_URL https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries
RUN mkdir -p /usr/share/maven /usr/share/maven/ref \
  && echo "Downloading maven" \
  && curl -fsSL -o /tmp/apache-maven.tar.gz ${BASE_URL}/apache-maven-${MAVEN_VERSION}-bin.tar.gz \
  && echo "Unziping maven" \
  && tar -xzf /tmp/apache-maven.tar.gz -C /usr/share/maven --strip-components=1 \
  && echo "Cleaning and setting links" \
  && rm -f /tmp/apache-maven.tar.gz \
  && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn
ENV MAVEN_HOME /usr/share/maven
ENV MAVEN_CONFIG "$HOME/.m2"

# Downloading and installing Gradle
ENV GRADLE_VERSION 4.0.1
ENV GRADLE_HOME /usr/bin/gradle
ENV GRADLE_USER_HOME /cache
ENV PATH $PATH:$GRADLE_HOME/bin
ENV GRADLE_BASE_URL https://services.gradle.org/distributions
RUN mkdir -p /usr/share/gradle /usr/share/gradle/ref \
  && echo "Downloading gradle hash" \
  && curl -fsSL -o /tmp/gradle.zip ${GRADLE_BASE_URL}/gradle-${GRADLE_VERSION}-bin.zip \
  && echo "Unziping gradle" \
  && unzip -d /usr/share/gradle /tmp/gradle.zip \
  && echo "Cleaning and setting links" \
  && rm -f /tmp/gradle.zip \
  && ln -s /usr/share/gradle/gradle-${GRADLE_VERSION} /usr/bin/gradle

####################################################
# Setup Move2Kube
####################################################

COPY --from=build_base /bin/move2kube /bin/move2kube
VOLUME ["/workspace"]
#"/var/run/docker.sock" needs to be mounted for CNB containerization to use docker
# Start app
WORKDIR /workspace
CMD move2kube
