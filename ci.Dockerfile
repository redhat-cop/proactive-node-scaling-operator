FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /
COPY config/templates /templates
COPY bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
