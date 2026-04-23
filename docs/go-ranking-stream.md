# Go Ranking Stream Worker

The project keeps Kafka as the event bus and uses a Go worker for online ranking updates.

## Runtime Flow

```text
article-rpc
        |
article_outbox_events
        |
article-outbox
        |
      Kafka
        |
ranking-stream
        |
      Redis ranking:hot
        |
ranking-rpc

interaction-rpc
        |
interaction_states + interaction_outbox_events
        |
      Kafka
        |
ranking-stream
        |
      Redis ranking:hot
        |
ranking-rpc
```

`article-rpc` writes the article row and the `article.published` outbox row in one MySQL transaction. `interaction-rpc` first records `interaction_states` so like/unlike is idempotent, then writes an interaction outbox row only when the state actually changes. Outbox workers publish pending rows to Kafka and mark them sent only after Kafka accepts the message.

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
go run ./app/article/cmd/outbox -f app/article/cmd/outbox/etc/outbox.yaml
go run ./app/interaction/cmd/outbox -f app/interaction/cmd/outbox/etc/outbox.yaml
```

Local dependencies:

```bash
docker compose up -d redis kafka
```

## Kubernetes

The worker is deployed by:

```bash
kubectl apply -f deploy/k8s/app/article-outbox.yaml
kubectl apply -f deploy/k8s/app/interaction-outbox.yaml
kubectl apply -f deploy/k8s/app/ranking-stream.yaml
kubectl rollout status deployment/article-outbox -n freeexchanged
kubectl rollout status deployment/interaction-outbox -n freeexchanged
kubectl rollout status deployment/ranking-stream -n freeexchanged
```

`deploy/k8s/deploy.sh` applies `article-outbox`, `interaction-outbox`, and `ranking-stream` before `ranking-rpc`.
