# Build the manager binary
FROM golang:1.27.3 AS builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Build the metrics sidecar from pinned source with independently patched dependencies.
COPY tools/kube-rbac-proxy/ tools/kube-rbac-proxy/
RUN cd tools/kube-rbac-proxy && CGO_ENABLED=0 go build -mod=readonly -o /workspace/kube-rbac-proxy github.com/brancz/kube-rbac-proxy/cmd/kube-rbac-proxy

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -o manager main.go

# Use UBI 9 minimal to package the manager binary
FROM registry.access.redhat.com/ubi9/ubi-minimal
WORKDIR /
COPY --from=builder /workspace/manager .
COPY --from=builder /workspace/kube-rbac-proxy .
COPY config/templates /templates
USER 65532:65532

ENTRYPOINT ["/manager"]
