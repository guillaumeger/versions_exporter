FROM scratch
ADD versions_exporter /
ADD fixtures/config.yaml /
ENV VERSIONS_EXPORTER_LOGLEVEL=debug 
ENV VERSIONS_EXPORTER_CONFIG_FILE="/config.yaml" 
EXPOSE 8080
ENTRYPOINT ["/versions_exporter"]