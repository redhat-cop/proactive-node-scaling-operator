FROM registry.access.redhat.com/ubi9/ubi-minimal
WORKDIR /
COPY config/templates /templates
COPY bin/manager .
COPY bin/kube-rbac-proxy .
USER 65532:65532

ENTRYPOINT ["/manager"]
