# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/manager/main.go main.go
COPY pkg/ pkg/
COPY version/ version/
COPY vendor/ vendor/
# Build
ARG operator_version=dev
RUN CGO_ENABLED=0 \
    GO111MODULE=on \
    go build \
    -ldflags "-X \"github.com/1Password/onepassword-operator/version.Version=$operator_version\"" \
    -mod vendor \
    -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER nonroot:nonroot
COPY deploy/connect/ deploy/connect/

ENTRYPOINT ["/manager"]
