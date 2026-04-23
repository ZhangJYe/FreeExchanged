#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NAMESPACE="freeexchanged"

echo "==> [1/4] Apply namespace"
kubectl apply -f "$SCRIPT_DIR/namespace.yaml"

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
kubectl apply -f "$SCRIPT_DIR/app/ranking-stream.yaml"
kubectl apply -f "$SCRIPT_DIR/app/ranking-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/article-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/watchlist-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/gateway.yaml"

echo "==> Waiting for application readiness"
kubectl rollout status deployment/rate-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/user-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/interaction-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/article-outbox -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/ranking-stream -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/ranking-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/article-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/watchlist-rpc -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/gateway -n "$NAMESPACE" --timeout=120s

echo ""
echo "Deployment completed."
echo "Gateway ClusterIP: $(kubectl get svc gateway-svc -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}'):8888"
echo "Grafana:           kubectl port-forward svc/grafana-svc 3000:3000 -n $NAMESPACE"
echo "Jaeger UI:         kubectl port-forward svc/jaeger-svc 16686:16686 -n $NAMESPACE"
