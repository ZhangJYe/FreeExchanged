# Phase 1: Kafka Event Bus

## Scope

This phase replaces the application event bus from RabbitMQ to Kafka for the current online ranking flow.

Changed producers:

- `article-rpc` publishes article lifecycle events to `article.events`.
- `interaction-rpc` publishes like, unlike, and read events to `interaction.events`.

Changed consumer:

- `ranking-rpc` consumes both Kafka topics and updates Redis key `ranking:hot`.

## Topic Contract

Topics:

- `article.events`
- `interaction.events`

Article event example:

```json
{
  "event_id": "uuid",
  "event_type": "article.published",
  "version": 1,
  "article_id": 1001,
  "title": "example",
  "author_id": 2001,
  "occurred_at": 1710000000
}
```

Interaction event example:

```json
{
  "event_id": "uuid",
  "event_type": "interaction.like",
  "version": 1,
  "user_id": 2001,
  "article_id": 1001,
  "occurred_at": 1710000000
}
```

## Ranking Semantics

`ranking-rpc` keeps the existing Redis output:

- New article: `ZADD ranking:hot <occurred_at> <article_id>`
- Like: `ZINCRBY ranking:hot 10 <article_id>`
- Unlike: `ZINCRBY ranking:hot -10 <article_id>`
- Read: `ZINCRBY ranking:hot 1 <article_id>`

The consumer still accepts legacy event names (`publish`, `like`, `unlike`, `read`) so replay tooling can bridge old payloads if needed.

## Local Development

`docker-compose.yml` now starts Kafka on `127.0.0.1:9092`.

```bash
docker compose up -d mysql redis kafka consul prometheus grafana jaeger
```

The local Go configs use:

```yaml
Kafka:
  Brokers:
    - 127.0.0.1:9092
```

## Notes

- This phase keeps JSON payloads for readability and low migration cost.
- Consumer offset handling follows Kafka consumer groups.
- The next phase will replace the Kubernetes RabbitMQ manifest with Kafka infrastructure.
