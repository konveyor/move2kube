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
FROM registry.access.redhat.com/ubi8/ubi:latest AS builder
WORKDIR /temp
ENV GOPATH=/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
RUN curl -o go.tar.gz https://dl.google.com/go/go{{ .GoVersion }}.linux-amd64.tar.gz
RUN tar -xzf go.tar.gz && mv go /usr/local/
RUN yum install git make -y 
RUN mkdir -p $GOPATH/src $GOPATH/bin && chmod -R 777 $GOPATH
WORKDIR /{{ .AppName }}
COPY . .
RUN go build -o {{ .AppName }}
RUN cp ./{{ .AppName }} /bin/{{ .AppName }}

# Run App
FROM registry.access.redhat.com/ubi8/ubi-minimal:8.3-201
COPY --from=builder /bin/{{ .AppName }} /bin/{{ .AppName }}
{{- range $port := .Ports }}
EXPOSE {{ $port }}
{{- end }}
CMD ["{{ .AppName }}"]
