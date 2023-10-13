# syntax = docker/dockerfile:1-experimental

FROM --platform=${BUILDPLATFORM} golang:1-alpine as builder
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

RUN apk add --no-cache git
COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download
COPY . /app
RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=0 GOEXPERIMENT=loopvar GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
  go build -ldflags '-s -w' -trimpath -v -o /app/kostal-influx-bridge

FROM alpine:3
RUN apk add --no-cache tini
COPY --from=builder /app/kostal-influx-bridge .
WORKDIR /
USER nobody
CMD ["/sbin/tini", "--", "/kostal-influx-bridge"]
