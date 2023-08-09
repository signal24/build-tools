FROM golang:1.20 AS builder
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

FROM gcr.io/kaniko-project/executor:v1.13.0-debug
COPY --from=builder /src/build/* /bin/
