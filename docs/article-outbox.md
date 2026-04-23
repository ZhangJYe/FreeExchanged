# Article Publish Outbox

Article publishing uses the transactional outbox pattern to avoid losing `article.published` events.

## Flow

```text
article-rpc
  |
  | MySQL transaction
  v
articles + article_outbox_events
  |
  | poll pending rows
  v
article-outbox
  |
  | publish and then mark sent
  v
Kafka article.events
```

## Failure Behavior

- If MySQL insert fails, the API returns an error and no event is recorded.
- If Kafka is unavailable, the outbox row stays pending and is retried with backoff.
- If the worker crashes after Kafka accepts the message but before `status=sent`, the event may be delivered again. Ranking handles `article.published` idempotently with Redis `ZADD` by article id.

## Local Run

```bash
go run ./app/article/cmd/outbox -f app/article/cmd/outbox/etc/outbox.yaml
```

## Kubernetes

`deploy/k8s/app/article-outbox.yaml` deploys two worker replicas. Each worker claims rows with `status=processing`, `locked_by`, and `locked_until` before publishing, so replicas do not intentionally process the same pending row. Delivery remains at-least-once if Kafka succeeds but the sent marker update fails.
