FROM golang:1.22 AS builder
WORKDIR /src
ARG TARGETARCH
ARG TARGETOS
ENV GOARCH=$TARGETARCH
ENV GOOS=$TARGETOS
ENV CGO_ENABLED=0
ENV GOBIN=/usr/local/bin
COPY . .
RUN mkdir build && \
    for file in *.go; do \
    echo Building $file; \
    go build -o build/${file%.go} $file; \
    done

FROM gcr.io/kaniko-project/executor:v1.23.2-debug AS kaniko

FROM alpine:latest

COPY --from=builder /src/build/* /bin/
COPY --from=kaniko /kaniko/ /kaniko/

RUN apk --update --no-cache add jq

ENV PATH $PATH:/usr/local/bin:/kaniko
ENV DOCKER_CONFIG /kaniko/.docker/
