# FreeExchanged Kubernetes Deployment

This directory contains the Kubernetes manifests for the FreeExchanged backend stack.

## Components

- `namespace.yaml`: creates the `freeexchanged` namespace.
- `infra/`: MySQL, Redis, Kafka, Jaeger, Prometheus, Grafana, and runtime secrets.
- `app/`: gateway, RPC services, Go stream workers, the exchange-rate CronJob, and the database migration Job.
- `ops/`: on-demand maintenance jobs such as ranking rebuild.
- `deploy.sh`: applies resources in dependency order.

## Deployment Order

1. Apply namespace.
2. Apply infrastructure and wait for MySQL, Kafka, Redis, and Kafka topic initialization.
3. Run `db-migration` to apply the schema from `infra/mysql.yaml`.
4. Deploy application workloads and wait for all Deployments.

The migration Job is idempotent because the schema uses `CREATE TABLE IF NOT EXISTS`.
The exchange-rate worker runs as a CronJob every 15 minutes. Its K8s config sets `RunOnce: true`, so each CronJob execution fetches rates once and exits.

The default K8s manifests run Kafka as a three-broker KRaft cluster and Redis with AOF persistence on a PVC. The K3s lite overlay scales Kafka and stream workers down for single-node demos.

## Required Images

Build and push these images before running the manifests:

- `freeexchanged/gateway:v0.1.0`
- `freeexchanged/user-rpc:v0.1.0`
- `freeexchanged/article-rpc:v0.1.0`
- `freeexchanged/article-outbox:v0.1.0`
- `freeexchanged/interaction-rpc:v0.1.0`
- `freeexchanged/interaction-outbox:v0.1.0`
- `freeexchanged/ranking-stream:v0.1.0`
- `freeexchanged/ranking-rebuild:v0.1.0`
- `freeexchanged/ranking-rpc:v0.1.0`
- `freeexchanged/rate-rpc:v0.1.0`
- `freeexchanged/watchlist-rpc:v0.1.0`
- `freeexchanged/rate-job:v0.1.0`
- `freeexchanged/web:v0.1.0`

The CI workflow builds immutable `sha-<commit>` image tags and uploads rendered K8s manifests with those tags. For manual deploys, pass the tag explicitly:

```bash
IMAGE_TAG=sha-xxxxxxxxxxxx bash deploy/k8s/deploy.sh
```

## Rebuild Ranking

If Redis data is lost and `ranking:hot` needs to be rebuilt from MySQL state, run the maintenance Job:

```bash
kubectl delete job/ranking-rebuild -n freeexchanged --ignore-not-found
kubectl apply -f deploy/k8s/ops/ranking-rebuild-job.yaml
kubectl wait --for=condition=complete job/ranking-rebuild -n freeexchanged --timeout=180s
kubectl logs job/ranking-rebuild -n freeexchanged
```

The rebuild job recalculates the hot ranking from published articles and `interaction_states`, then rewrites `ranking:hot` in Redis.
If your environment uses immutable `sha-<commit>` images, apply the rendered manifest artifact from CI or replace the job image tag before running it.

## Secrets

Runtime secrets are not committed. Before running `deploy.sh`, export the required values or create `freeexchanged-secrets` with your own secret manager and run with `SKIP_SECRET_SYNC=1`.

Required keys:

- `MYSQL_ROOT_PASSWORD`
- `MYSQL_DATABASE`
- `MYSQL_DSN`
- `PASETO_ACCESS_SECRET`
- `GRAFANA_ADMIN_PASSWORD`

Example:

```bash
export MYSQL_ROOT_PASSWORD='...'
export MYSQL_DATABASE='freeexchanged'
export MYSQL_DSN='root:...@tcp(mysql-svc:3306)/freeexchanged?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai'
export PASETO_ACCESS_SECRET='base64-encoded-32-byte-key'
export GRAFANA_ADMIN_PASSWORD='...'
```

## Deploy

```bash
bash deploy/k8s/deploy.sh
```

For the K3s lite overlay, use the kustomization at `deploy/`:

```bash
kubectl kustomize deploy
kubectl apply -k deploy
```

If you only want to apply a single phase:

```bash
kubectl apply -f deploy/k8s/namespace.yaml
kubectl create secret generic freeexchanged-secrets -n freeexchanged \
  --from-literal=MYSQL_ROOT_PASSWORD="$MYSQL_ROOT_PASSWORD" \
  --from-literal=MYSQL_DATABASE="$MYSQL_DATABASE" \
  --from-literal=MYSQL_DSN="$MYSQL_DSN" \
  --from-literal=PASETO_ACCESS_SECRET="$PASETO_ACCESS_SECRET" \
  --from-literal=GRAFANA_ADMIN_PASSWORD="$GRAFANA_ADMIN_PASSWORD" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -f deploy/k8s/infra/
kubectl apply -f deploy/k8s/app/db-migration-job.yaml
kubectl apply -f deploy/k8s/app/
```

When rerunning the migration Job manually:

```bash
kubectl delete job/db-migration -n freeexchanged --ignore-not-found
kubectl apply -f deploy/k8s/app/db-migration-job.yaml
kubectl wait --for=condition=complete job/db-migration -n freeexchanged --timeout=180s
```

## Verify

```bash
kubectl get pods -n freeexchanged
kubectl get svc -n freeexchanged
kubectl get cronjob -n freeexchanged
kubectl logs job/kafka-topic-init -n freeexchanged
kubectl logs job/db-migration -n freeexchanged
kubectl rollout status deployment/gateway -n freeexchanged
kubectl port-forward svc/prometheus-svc 9090:9090 -n freeexchanged
```

Local access:

```bash
kubectl port-forward svc/gateway-svc 8888:8888 -n freeexchanged
kubectl port-forward svc/grafana-svc 3000:3000 -n freeexchanged
kubectl port-forward svc/jaeger-svc 16686:16686 -n freeexchanged
kubectl port-forward svc/prometheus-svc 9090:9090 -n freeexchanged
```

Prometheus scrapes the RPC services plus `ranking-stream`, `article-outbox`, and `interaction-outbox`, so you can inspect worker throughput and failure counters directly in the Prometheus UI.

## Production Hardening Backlog

- Add image provenance/signing and promotion gates.
- Replace raw Secret manifests with a secret-management controller.
- Add Ingress, TLS, and external DNS.
- Add HPA after observing CPU, memory, and request metrics.
- Move schema changes to a versioned migration tool once the schema starts changing frequently.
- Use managed MySQL, Redis, and Kafka if the target environment already provides them.
