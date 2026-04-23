# High-Concurrency Notes

This project now uses a Go-only event pipeline for write-heavy paths:

```text
gateway
  -> RPC write service
  -> MySQL state + outbox transaction
  -> horizontally scalable outbox workers
  -> Kafka topics
  -> ranking-stream consumer group
  -> Redis ZSET ranking
```

## Applied Optimizations

- Like/unlike is idempotent through `interaction_states` with a unique `(user_id, article_id)` key. Repeated like requests no longer create repeated score increments.
- Article and interaction events use transactional outbox tables. API success no longer depends on an in-process goroutine delivering Kafka before a crash.
- Outbox workers claim rows with `status=processing`, `locked_by`, and `locked_until`, so multiple replicas can drain the same table without selecting the same pending rows.
- Outbox workers refresh each row lease before publishing and verify the lease before marking a row sent or failed.
- Ranking consumers commit Kafka offsets only after Redis updates succeed.
- Ranking consumers retry transient Redis failures, then publish poison messages to `ranking.dlq` before committing.
- WebSocket broadcasts use per-connection buffered write queues and write deadlines. Slow clients are dropped instead of blocking all broadcasts.
- Redis uses AOF persistence with a PVC so the online ranking cache survives Pod restarts.
- The standard K8s Kafka manifest runs three brokers with replication factor 3 and min in-sync replicas 2; the K3s lite overlay intentionally scales this down for small single-node environments.
- Kafka topic initialization verifies existing topic partition and replication settings. If a production topic was created with replication factor 1, deployment fails instead of silently running with weak durability.
- Gateway like/unlike traffic is rate-limited in Redis per user, with IP fallback, so limits are shared across gateway replicas without one user consuming a Pod-wide bucket.

## Next Useful Benchmarks

- Gateway HTTP QPS and P99 latency for login, publish, like, and top-ranking reads.
- Outbox drain throughput and `pending` row count under Kafka outage and recovery.
- DLQ message rate and replay time after a Redis or consumer bug.
- Kafka consumer lag for `article.events` and `interaction.events`.
- Redis command latency for ranking ZSET updates under high interaction volume.
