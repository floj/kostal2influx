#!/usr/bin/env bash
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null && pwd)"
appName=$(basename "$DIR")

docker pull golang:1-alpine
docker pull alpine:3
DOCKER_BUILDKIT=1 docker build -t "floj/$appName:latest" "$DIR"
