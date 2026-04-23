# FreeExchanged Kubernetes Deployment

This directory contains the Kubernetes manifests for the FreeExchanged backend stack.

## Components

- `namespace.yaml`: creates the `freeexchanged` namespace.
- `infra/`: MySQL, Redis, Kafka, Jaeger, Prometheus, Grafana, and runtime secrets.
- `app/`: gateway, RPC services, the exchange-rate CronJob, and the database migration Job.
- `analytics/`: optional Flink and Hive analytics workloads.
- `deploy.sh`: applies resources in dependency order.

## Deployment Order

1. Apply namespace.
2. Apply infrastructure and wait for MySQL, Kafka, Redis, and Kafka topic initialization.
3. Run `db-migration` to apply the schema from `infra/mysql.yaml`.
4. Deploy application workloads and wait for all Deployments.

The migration Job is idempotent because the schema uses `CREATE TABLE IF NOT EXISTS`.
The exchange-rate worker runs as a CronJob every 15 minutes. Its K8s config sets `RunOnce: true`, so each CronJob execution fetches rates once and exits.

## Required Images

Build and push these images before running the manifests:

- `freeexchanged/gateway:latest`
- `freeexchanged/user-rpc:latest`
- `freeexchanged/article-rpc:latest`
- `freeexchanged/interaction-rpc:latest`
- `freeexchanged/ranking-rpc:latest`
- `freeexchanged/rate-rpc:latest`
- `freeexchanged/watchlist-rpc:latest`
- `freeexchanged/rate-job:latest`

For production, replace `latest` with immutable version tags.

## Secrets

`infra/secrets.yaml` is suitable for local or demo clusters only. Before production deployment, replace these values or manage them with External Secrets, Sealed Secrets, SOPS, or the cloud provider secret manager.

Required keys:

- `MYSQL_ROOT_PASSWORD`
- `MYSQL_DATABASE`
- `MYSQL_DSN`
- `PASETO_ACCESS_SECRET`
- `GRAFANA_ADMIN_PASSWORD`

## Deploy

```bash
bash deploy/k8s/deploy.sh
```

If you only want to apply a single phase:

```bash
kubectl apply -f deploy/k8s/namespace.yaml
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
```

Local access:

```bash
kubectl port-forward svc/gateway-svc 8888:8888 -n freeexchanged
kubectl port-forward svc/grafana-svc 3000:3000 -n freeexchanged
kubectl port-forward svc/jaeger-svc 16686:16686 -n freeexchanged
```

## Production Hardening Backlog

- Add CI image build and manifest tag replacement.
- Replace raw Secret manifests with a secret-management controller.
- Add Ingress, TLS, and external DNS.
- Add HPA after observing CPU, memory, and request metrics.
- Move schema changes to a versioned migration tool once the schema starts changing frequently.
- Use managed MySQL, Redis, and Kafka if the target environment already provides them.
- Move Hive warehouse storage to HDFS or object storage before production analytics usage.
