
FROM {{ .openliberty_image}}
# E.g.: openliberty/open-liberty:kernel-java8-openj9-ubi

{{range $port := .port}}EXPOSE {{$port}}
{{end}}

USER root
COPY --chown=1001:0 src/main/liberty/config/server.xml /config/
COPY --chown=1001:0 target/*.jar /config/apps/
USER 1001

RUN configure.sh