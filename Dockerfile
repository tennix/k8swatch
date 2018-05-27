# Builder image
# FROM golang:1.9-alpine AS builder

# WORKDIR /go/src/github.com/tennix/k8swatch
# ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
# ADD . /go/src/github.com/tennix/k8swatch
# RUN go build -o bin/k8swatch main.go

# Executable image
FROM alpine:3.5

# COPY --from=builder /go/src/github.com/wyatt88/k8swatch/bin/k8swatch /k8swatch
ADD k8swatch /k8swatch

WORKDIR /

ENTRYPOINT ["/k8swatch"]
