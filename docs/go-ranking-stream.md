# Go Ranking Stream Worker

The project keeps Kafka as the event bus and uses a Go worker for online ranking updates.

## Runtime Flow

```text
article-rpc / interaction-rpc
        |
      Kafka
        |
ranking-stream
        |
      Redis ranking:hot
        |
ranking-rpc
```

## Why Not Flink/Hive Here

Flink and Hive are JVM-centric systems. They are useful for large-scale streaming and offline warehouse workloads, but they add Maven, JVM runtime, metastore, checkpoint, and operational overhead. The current online ranking logic only needs to consume Kafka events and update Redis, so a Go worker is a better fit.

## Offset Handling

`ranking-stream` uses manual Kafka offset handling:

1. Fetch message.
2. Apply Redis update.
3. Commit offset only after Redis succeeds.

Malformed or unsupported events are logged and committed so they do not block the stream. Redis failures are retried before committing.

## Local Run

```bash
go run ./app/ranking/cmd/stream -f app/ranking/cmd/stream/etc/stream.yaml
```

Local dependencies:

```bash
docker compose up -d redis kafka
```

## Kubernetes

The worker is deployed by:

```bash
kubectl apply -f deploy/k8s/app/ranking-stream.yaml
kubectl rollout status deployment/ranking-stream -n freeexchanged
```

`deploy/k8s/deploy.sh` applies this worker before `ranking-rpc`.
