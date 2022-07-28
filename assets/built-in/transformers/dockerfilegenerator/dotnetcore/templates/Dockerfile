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

{{ if .IncludeBuildStage }}

# Build Stage
FROM mcr.microsoft.com/dotnet/sdk:{{ .BuildStageImageTag }} AS {{ .BuildContainerName }}

{{- if .IsNodeJSProject }}
RUN apt-get update -y && \
    apt-get install -y xz-utils && \
    wget https://nodejs.org/dist/v16.16.0/node-v16.16.0-linux-x64.tar.xz && \
    tar -xJf node-v16.16.0-linux-x64.tar.xz && \
    mv node-v16.16.0-linux-x64 /node && \
    rm node-v16.16.0-linux-x64.tar.xz
ENV PATH="$PATH:/node/bin"
{{- end }}

WORKDIR /src
COPY . .
RUN dotnet restore

{{- if .PublishProfilePath }}
RUN dotnet publish /p:PublishProfile={{ .PublishProfilePath }} --no-restore
{{- else }}
RUN dotnet publish -c release --no-restore
{{- end }}

{{- end }}


{{ if .IncludeRunStage }}

# Run Stage
FROM mcr.microsoft.com/dotnet/aspnet:{{ .RunStageImageTag }}
ENV DOTNET_GENERATE_ASPNET_CERTIFICATE=false
WORKDIR /app

{{- range .Ports }}
EXPOSE {{ . }}
{{- end }}

{{- range $k, $v := .EnvVariables }}
ENV {{ $k }} {{ $v }}
{{- end }}

COPY {{ if .BuildContainerName }}--from={{ .BuildContainerName }}{{ end }} {{ .CopyFrom }} ./
CMD ["dotnet", "{{ .EntryPointPath }}"]

{{- end }}
