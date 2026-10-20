FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY config/templates /templates
COPY bin/manager .
COPY bin/kube-rbac-proxy .
USER 65532:65532

ENTRYPOINT ["/manager"]
