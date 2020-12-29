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
RUN dnf install -y git make
ENV GOPATH=/go
RUN mkdir -p $GOPATH/src $GOPATH/bin && chmod -R 777 $GOPATH
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH

# Download Go.
ARG GO_VERSION=1.15
RUN curl -o go.tgz "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz"
RUN tar -xzf go.tgz && mv go /usr/local/

# Copy only go.mod, go.sum and download packages to allow better caching.
ENV WORKDIR=${GOPATH}/src/move2kube
WORKDIR ${WORKDIR}
COPY go.mod .
COPY go.sum .
RUN go mod download

# Install utils
RUN dnf install -y findutils

# Build
ARG VERSION=latest
COPY . .
RUN make build
RUN cp bin/move2kube /bin/move2kube


### Run image ###
FROM registry.fedoraproject.org/fedora:latest
RUN curl -o /usr/bin/operator-sdk -LJO 'https://github.com/operator-framework/operator-sdk/releases/download/v1.3.0/operator-sdk_linux_amd64' && chmod +x /usr/bin/operator-sdk

# Install utils
RUN dnf install -y findutils podman

COPY --from=build_base /bin/move2kube /bin/move2kube
VOLUME ["/workspace"]
#"/var/run/docker.sock" needs to be mounted for CNB containerization to be used.
# Start app
WORKDIR /workspace
CMD move2kube
