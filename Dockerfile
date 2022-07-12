#   Copyright IBM Corporation 2020, 2021
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
FROM registry.access.redhat.com/ubi8/ubi:latest AS builder
WORKDIR /temp

RUN yum -y install https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm
RUN dnf install -y git make
ENV GOPATH=/go
RUN mkdir -p $GOPATH/src $GOPATH/bin && chmod -R 777 $GOPATH
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH

# Download Go.
ARG GO_VERSION=1.18
ARG TARGETPLATFORM
RUN export ARCH=${TARGETPLATFORM//\//-} && curl -o go.tgz "https://dl.google.com/go/go${GO_VERSION}.${ARCH}.tar.gz" \
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
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
COPY --from=builder /bin/move2kube /bin/move2kube
VOLUME ["/workspace"]
#"/var/run/docker.sock" needs to be mounted for container based transformers to work with docker
# Start app
WORKDIR /workspace
CMD move2kube
