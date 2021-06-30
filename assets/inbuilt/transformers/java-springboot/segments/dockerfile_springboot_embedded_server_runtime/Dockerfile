
FROM {{ .java_image }}
COPY --from=build_base /app/target/*.jar app.jar
{{range $port := .port}}EXPOSE {{$port}}
{{end}}
CMD ["java", "-Xmx200m", "-jar", "/app.jar"]