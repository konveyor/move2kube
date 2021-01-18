# Run App
FROM registry.access.redhat.com/jboss-eap-6/eap64-openshift:latest
EXPOSE {{ .port }}
COPY --from=build_base /{{ .app_name }}/target/* /opt/eap/standalone/deployments/