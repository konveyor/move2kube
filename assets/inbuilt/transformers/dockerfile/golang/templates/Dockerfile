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
WORKDIR /temp
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
RUN curl -o go.tar.gz https://dl.google.com/go/go1.16.linux-amd64.tar.gz
RUN tar -xzf go.tar.gz && mv go /usr/local/
RUN yum install git make -y 
RUN mkdir -p $GOPATH/src $GOPATH/bin && chmod -R 777 $GOPATH
WORKDIR /{{ .app_name }}
COPY . .
RUN go build -o {{ .app_name }}
RUN cp ./{{ .app_name }} /bin/{{ .app_name }}

# Run App
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3-201
COPY --from=build_base /bin/{{ .app_name }} /bin/{{ .app_name }}
EXPOSE {{ .port }}
CMD ["{{ .app_name }}"]
