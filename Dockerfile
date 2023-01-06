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
  GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
  go build -v -o /app/kostal-influx-bridge

FROM alpine:3
USER nobody
WORKDIR /
COPY --from=builder /app/kostal-influx-bridge .

CMD ["/kostal-influx-bridge"]
