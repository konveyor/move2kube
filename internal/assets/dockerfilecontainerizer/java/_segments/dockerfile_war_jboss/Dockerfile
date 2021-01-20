FROM registry.access.redhat.com/jboss-eap-6/eap64-openshift:1.9-32.1604502274
COPY {{ .war_path }} /opt/eap/standalone/deployments/
EXPOSE {{ .port }}