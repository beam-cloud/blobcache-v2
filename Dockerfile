# syntax=docker/dockerfile:1.6

FROM golang:1.23.4-bullseye AS build

WORKDIR /workspace

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    fuse3 libfuse2 libfuse3-dev  && \
    rm -rf /var/lib/apt/lists/*

RUN go install github.com/cosmtrek/air@v1.49.0

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o /usr/local/bin/blobcache /workspace/cmd/main.go

RUN curl -L https://beam-runner-python-deps.s3.amazonaws.com/juicefs -o /usr/local/bin/juicefs && chmod +x /usr/local/bin/juicefs
RUN curl -L https://beam-runner-python-deps.s3.amazonaws.com/mount-s3 -o /usr/local/bin/mount-s3 && chmod +x /usr/local/bin/mount-s3

CMD ["/usr/local/bin/blobcache"]


FROM ubuntu:22.04 AS release

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    fuse3 libfuse2 libfuse3-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=build /usr/local/bin/juicefs /usr/local/bin/juicefs
COPY --from=build /usr/local/bin/mount-s3 /usr/local/bin/mount-s3
COPY --from=build /usr/local/bin/blobcache /usr/local/bin/blobcache

WORKDIR /workspace

CMD ["/usr/local/bin/blobcache"]
