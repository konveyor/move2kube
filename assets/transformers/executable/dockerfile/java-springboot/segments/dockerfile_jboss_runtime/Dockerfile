
FROM {{ .jboss_image}}
{{range $port := .port}}EXPOSE {{$port}}
{{end}}
# Copy the war file to the deployments folder
COPY --from=build_base  /app/target/{{ .app_file }} {{ .deployment_path }}{{ .deployment_file }}
