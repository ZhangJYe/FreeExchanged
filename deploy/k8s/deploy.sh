#!/usr/bin/env bash
# 一键部署脚本：按依赖顺序部署所有资源
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> [1/3] 创建 Namespace"
kubectl apply -f "$SCRIPT_DIR/namespace.yaml"

echo "==> [2/3] 部署基础设施（MySQL/Redis/RabbitMQ/Jaeger/Prometheus/Grafana）"
kubectl apply -f "$SCRIPT_DIR/infra/"

echo "==> 等待 MySQL 和 Redis 就绪..."
kubectl rollout status statefulset/mysql -n freeexchanged --timeout=120s
kubectl rollout status deployment/redis   -n freeexchanged --timeout=60s
kubectl rollout status deployment/rabbitmq -n freeexchanged --timeout=90s

echo "==> [3/3] 部署应用层（RPC 服务 + Gateway）"
# 先启 rate-job 和 rpc 服务，最后启 gateway
kubectl apply -f "$SCRIPT_DIR/app/rate-job.yaml"
kubectl apply -f "$SCRIPT_DIR/app/rate-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/user-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/interaction-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/ranking-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/article-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/watchlist-rpc.yaml"
kubectl apply -f "$SCRIPT_DIR/app/gateway.yaml"

echo "==> 等待 Gateway 就绪..."
kubectl rollout status deployment/gateway -n freeexchanged --timeout=120s

echo ""
echo "部署完成！"
echo "Gateway ClusterIP: $(kubectl get svc gateway-svc -n freeexchanged -o jsonpath='{.spec.clusterIP}'):8888"
echo "Grafana:           kubectl port-forward svc/grafana-svc 3000:3000 -n freeexchanged"
echo "Jaeger UI:         kubectl port-forward svc/jaeger-svc 16686:16686 -n freeexchanged"
