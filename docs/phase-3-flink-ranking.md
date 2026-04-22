# Phase 3: Flink Ranking Job

## Scope

This phase adds a Flink DataStream job that consumes Kafka events and computes the same hot-ranking updates currently handled by `ranking-rpc`.

Added project:

- `streaming/flink-ranking-job`

Added Kubernetes manifest:

- `deploy/k8s/analytics/flink-ranking-job.yaml`

## Why Shadow Mode

The existing Go `ranking-rpc` Kafka consumer is still enabled. Running both writers against `ranking:hot` would double-count interaction events.

For that reason, the Flink manifest writes to:

```text
ranking:hot:flink
```

Use this key to validate output parity. After validation, switch over by:

1. Setting `ConsumerEnabled: false` in `deploy/k8s/app/ranking-rpc.yaml`.
2. Setting `RANKING_REDIS_KEY=ranking:hot` in `deploy/k8s/analytics/flink-ranking-job.yaml`.
3. Applying both manifests.

## Build Image

```bash
docker build -t freeexchanged/flink-ranking-job:latest streaming/flink-ranking-job
```

For production, replace `latest` with an immutable commit tag.

## Deploy

```bash
kubectl apply -f deploy/k8s/analytics/flink-ranking-job.yaml
kubectl rollout status deployment/flink-ranking-jobmanager -n freeexchanged
kubectl rollout status deployment/flink-ranking-taskmanager -n freeexchanged
```

Flink UI:

```bash
kubectl port-forward svc/flink-jobmanager 8081:8081 -n freeexchanged
```

## Event Mapping

- `article.published`: `ZADD <ranking-key> <occurred_at> <article_id>`
- `interaction.like`: `ZINCRBY <ranking-key> 10 <article_id>`
- `interaction.unlike`: `ZINCRBY <ranking-key> -10 <article_id>`
- `interaction.read`: `ZINCRBY <ranking-key> 1 <article_id>`

## Production Notes

- Redis writes are at-least-once. If exact counts become critical, introduce event IDs and idempotent state before writing to Redis.
- Checkpointing is enabled at 30 seconds.
- A production Flink deployment should use durable checkpoint storage and savepoints.
