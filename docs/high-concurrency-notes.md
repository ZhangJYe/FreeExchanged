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
- Ranking consumers commit Kafka offsets only after Redis updates succeed.
- WebSocket broadcasts use per-connection buffered write queues and write deadlines. Slow clients are dropped instead of blocking all broadcasts.

## Next Useful Benchmarks

- Gateway HTTP QPS and P99 latency for login, publish, like, and top-ranking reads.
- Outbox drain throughput and `pending` row count under Kafka outage and recovery.
- Kafka consumer lag for `article.events` and `interaction.events`.
- Redis command latency for ranking ZSET updates under high interaction volume.
