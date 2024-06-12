# syntax=docker/dockerfile:1.6

FROM golang:1.21-bullseye AS build

WORKDIR /workspace

RUN go install github.com/cosmtrek/air@v1.49.0

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -o /usr/local/bin/blobcache /workspace/cmd/main.go

CMD ["/usr/local/bin/blobcache"]


FROM ubuntu:22.04 AS release

COPY --from=build /workspace/blobcache /usr/local/bin

WORKDIR /workspace

CMD ["/usr/local/bin/blobcache"]
