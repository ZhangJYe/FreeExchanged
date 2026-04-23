#!/usr/bin/env bash
set -euo pipefail

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Please run as root so images can be imported into k3s." >&2
  exit 1
fi

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GO_DOCKERFILE="$ROOT_DIR/deploy/docker/go-service.Dockerfile"

build_go_image() {
  local image="$1"
  local cmd_path="$2"
  docker build -f "$GO_DOCKERFILE" --build-arg CMD_PATH="$cmd_path" -t "$image" "$ROOT_DIR"
  docker save "$image" | k3s ctr images import -
  docker rmi "$image" >/dev/null
}

build_web_image() {
  local image="$1"
  local app_base="${2:-/}"
  docker build --build-arg VITE_APP_BASE="$app_base" -t "$image" "$ROOT_DIR/web"
  docker save "$image" | k3s ctr images import -
  docker rmi "$image" >/dev/null
}

import_third_party_image() {
  local image="$1"
  docker pull "$image"
  docker save "$image" | k3s ctr images import -
}

build_go_image freeexchanged/gateway:k3s ./app/gateway
build_go_image freeexchanged/user-rpc:k3s ./app/user/cmd/rpc
build_go_image freeexchanged/interaction-rpc:k3s ./app/interaction/cmd/rpc
build_go_image freeexchanged/article-rpc:k3s ./app/article/cmd/rpc
build_go_image freeexchanged/rate-rpc:k3s ./app/rate/cmd/rpc
build_go_image freeexchanged/watchlist-rpc:k3s ./app/watchlist/cmd/rpc
build_go_image freeexchanged/ranking-rpc:k3s ./app/ranking/cmd/rpc
build_go_image freeexchanged/article-outbox:k3s ./app/article/cmd/outbox
build_go_image freeexchanged/interaction-outbox:k3s ./app/interaction/cmd/outbox
build_go_image freeexchanged/ranking-stream:k3s ./app/ranking/cmd/stream
build_go_image freeexchanged/ranking-rebuild:k3s ./app/ranking/cmd/rebuild
build_go_image freeexchanged/rate-job:k3s ./app/rate/cmd/job
build_web_image freeexchanged/web:k3s /free/
import_third_party_image mysql:8.0
import_third_party_image redis:alpine
import_third_party_image bitnami/kafka:3.7
import_third_party_image jaegertracing/all-in-one:1.57

echo "All images built and imported into k3s."
