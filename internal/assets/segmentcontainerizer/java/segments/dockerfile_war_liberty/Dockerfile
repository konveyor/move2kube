FROM websphere-liberty:webProfile8
COPY --chown=1001:0 {{ .war_path }} /config/dropins/
EXPOSE {{ .port }}