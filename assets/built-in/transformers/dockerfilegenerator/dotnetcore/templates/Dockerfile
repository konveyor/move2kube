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

FROM mcr.microsoft.com/dotnet/sdk:{{ .DotNetVersion }} AS builder
WORKDIR /src
COPY . .
RUN mkdir app
RUN dotnet restore {{ .CsprojFilePath }}

{{- if .IsNodeJSProject}}
RUN curl https://deb.nodesource.com/setup_10.x -o setup_10.x && \
    bash setup_10.x && \
    apt-get install -y build-essential nodejs
{{- end }}

{{- if .PublishProfileFilePath }}
RUN dotnet publish {{ .CsprojFilePath }} /p:PublishProfile={{ .PublishProfileFilePath}} -o /src/app/publish
{{- else }}
RUN dotnet publish {{ .CsprojFilePath }} -c Release -o /src/app/publish
{{- end }}

# Run Stage
FROM mcr.microsoft.com/dotnet/aspnet:{{ .DotNetVersion }}
WORKDIR /app
{{- range $port := .Ports }}
EXPOSE {{ $port }}
{{- end }}
{{- if .HTTPPort }}
ENV ASPNETCORE_URLS=http://+:{{ .HTTPPort }}
{{- end}}
COPY --from=builder /src/app/publish .
CMD ["dotnet", "{{ .CsprojFileName }}.dll"]
