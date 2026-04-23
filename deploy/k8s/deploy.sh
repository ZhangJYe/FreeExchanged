#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="freeexchanged"
SECRET_NAME="freeexchanged-secrets"
IMAGE_REPOSITORY_PREFIX="${IMAGE_REPOSITORY_PREFIX:-freeexchanged}"
IMAGE_TAG="${IMAGE_TAG:-v0.1.0}"
REQUIRED_SECRET_VARS=(
  MYSQL_ROOT_PASSWORD
  MYSQL_DATABASE
  MYSQL_DSN
  PASETO_ACCESS_SECRET
  GRAFANA_ADMIN_PASSWORD
)

sync_secret() {
  if [[ "${SKIP_SECRET_SYNC:-}" == "1" ]]; then
    kubectl get secret "$SECRET_NAME" -n "$NAMESPACE" >/dev/null
    return
  fi

  local missing=()
  for name in "${REQUIRED_SECRET_VARS[@]}"; do
    if [[ -z "${!name:-}" ]]; then
      missing+=("$name")
    fi
  done

  if (( ${#missing[@]} > 0 )); then
    echo "Missing required secret env vars: ${missing[*]}" >&2
    echo "Export them first, or set SKIP_SECRET_SYNC=1 if $SECRET_NAME is managed outside this script." >&2
    exit 1
  fi

  kubectl create secret generic "$SECRET_NAME" \
    -n "$NAMESPACE" \
    --from-literal=MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
    --from-literal=MYSQL_DATABASE="$MYSQL_DATABASE" \
    --from-literal=MYSQL_DSN="$MYSQL_DSN" \
    --from-literal=PASETO_ACCESS_SECRET="$PASETO_ACCESS_SECRET" \
    --from-literal=GRAFANA_ADMIN_PASSWORD="$GRAFANA_ADMIN_PASSWORD" \
    --dry-run=client -o yaml | kubectl apply -f -
}

set_app_images() {
  echo "==> Set application image tag: $IMAGE_TAG"
  kubectl set image deployment/gateway -n "$NAMESPACE" gateway="$IMAGE_REPOSITORY_PREFIX/gateway:$IMAGE_TAG"
  kubectl set image deployment/user-rpc -n "$NAMESPACE" user-rpc="$IMAGE_REPOSITORY_PREFIX/user-rpc:$IMAGE_TAG"
  kubectl set image deployment/interaction-rpc -n "$NAMESPACE" interaction-rpc="$IMAGE_REPOSITORY_PREFIX/interaction-rpc:$IMAGE_TAG"
  kubectl set image deployment/article-rpc -n "$NAMESPACE" article-rpc="$IMAGE_REPOSITORY_PREFIX/article-rpc:$IMAGE_TAG"
  kubectl set image deployment/article-outbox -n "$NAMESPACE" article-outbox="$IMAGE_REPOSITORY_PREFIX/article-outbox:$IMAGE_TAG"
  kubectl set image deployment/interaction-outbox -n "$NAMESPACE" interaction-outbox="$IMAGE_REPOSITORY_PREFIX/interaction-outbox:$IMAGE_TAG"
  kubectl set image deployment/ranking-stream -n "$NAMESPACE" ranking-stream="$IMAGE_REPOSITORY_PREFIX/ranking-stream:$IMAGE_TAG"
  kubectl set image deployment/ranking-rpc -n "$NAMESPACE" ranking-rpc="$IMAGE_REPOSITORY_PREFIX/ranking-rpc:$IMAGE_TAG"
  kubectl set image deployment/rate-rpc -n "$NAMESPACE" rate-rpc="$IMAGE_REPOSITORY_PREFIX/rate-rpc:$IMAGE_TAG"
  kubectl set image deployment/watchlist-rpc -n "$NAMESPACE" watchlist-rpc="$IMAGE_REPOSITORY_PREFIX/watchlist-rpc:$IMAGE_TAG"
  kubectl set image deployment/web -n "$NAMESPACE" web="$IMAGE_REPOSITORY_PREFIX/web:$IMAGE_TAG"
  kubectl set image cronjob/rate-job -n "$NAMESPACE" rate-job="$IMAGE_REPOSITORY_PREFIX/rate-job:$IMAGE_TAG"
}

echo "==> [1/4] Apply namespace"
kubectl apply -f "$SCRIPT_DIR/namespace.yaml"

echo "==> Sync runtime secrets"
sync_secret

echo "==> [2/4] Apply infrastructure"
kubectl delete job/kafka-topic-init -n "$NAMESPACE" --ignore-not-found
kubectl apply -f "$SCRIPT_DIR/infra/"

echo "==> Waiting for infrastructure readiness"
kubectl rollout status statefulset/mysql -n "$NAMESPACE" --timeout=180s
kubectl rollout status statefulset/kafka -n "$NAMESPACE" --timeout=180s
kubectl rollout status deployment/redis -n "$NAMESPACE" --timeout=90s
kubectl wait --for=condition=complete job/kafka-topic-init -n "$NAMESPACE" --timeout=180s

echo "==> [3/4] Run database migration"
kubectl delete job/db-migration -n "$NAMESPACE" --ignore-not-found
kubectl apply -f "$SCRIPT_DIR/app/db-migration-job.yaml"
kubectl wait --for=condition=complete job/db-migration -n "$NAMESPACE" --timeout=180s

echo "==> [4/4] Deploy application workloads"
kubectl apply -f "$SCRIPT_DIR/app/rate-job.yaml"
kubectl apply -f "$SCRIPT_DIR/app/rate-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/user-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/interaction-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/article-outbox.yaml"
kubectl apply -f "$SCRIPT_DIR/app/interaction-outbox.yaml"
kubectl apply -f "$SCRIPT_DIR/app/ranking-stream.yaml"
kubectl apply -f "$SCRIPT_DIR/app/ranking-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/article-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/watchlist-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/web.yaml"
kubectl apply -f "$SCRIPT_DIR/app/gateway.yaml"
set_app_images

echo "==> Waiting for application readiness"
kubectl rollout status deployment/rate-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/user-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/interaction-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/article-outbox -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/interaction-outbox -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/ranking-stream -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/ranking-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/article-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/watchlist-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/web -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/gateway -n "$NAMESPACE" --timeout=120s

echo ""
echo "Deployment completed."
echo "Web ingress:       route your domain to the ingress controller for /, /v1, and /ws"
echo "Gateway ClusterIP: $(kubectl get svc gateway-svc -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}'):8888"
echo "Grafana:           kubectl port-forward svc/grafana-svc 3000:3000 -n $NAMESPACE"
echo "Jaeger UI:         kubectl port-forward svc/jaeger-svc 16686:16686 -n $NAMESPACE"
