# Phase 2: Kafka Kubernetes Infrastructure

## Scope

This phase removes the Kubernetes RabbitMQ deployment path and replaces it with Kafka KRaft infrastructure for the current project environment.

Added resources:

- `deploy/k8s/infra/kafka.yaml`
- `deploy/k8s/infra/kafka-topics-job.yaml`

Removed resource:

- `deploy/k8s/infra/rabbitmq.yaml`

## Runtime Topology

```text
article-rpc ----------> article_outbox_events
interaction-rpc ------> interaction_outbox_events
outbox workers -------> Kafka topics
ranking-stream -------> Redis sorted set: ranking:hot
ranking-rpc ----------> Redis read API
```

## Kafka Settings

The manifest uses Bitnami Kafka in KRaft mode:

- Three broker/controllers in the standard K8s manifests.
- The K3s lite overlay intentionally scales Kafka down to one broker for small single-node demos.
- Internal service: `kafka-svc:9092`.
- Headless service: `kafka-headless` for StatefulSet identity.
- Persistent volume: `5Gi`.
- Topic initialization Job creates and verifies:
  - `article.events`
  - `interaction.events`
  - `ranking.dlq`
- Standard topic settings are three partitions, replication factor 3, and `min.insync.replicas=2`.

## Deployment Script Changes

`deploy/k8s/deploy.sh` now:

1. Deletes the old `kafka-topic-init` Job if present.
2. Applies infrastructure.
3. Waits for MySQL, Kafka, Redis, and topic initialization.
4. Runs DB migration.
5. Deploys application workloads.

## Production Notes

For larger production deployments, prefer one of these:

- Managed Kafka from the cloud provider.
- Strimzi Kafka Operator.
- A hardened self-managed StatefulSet with rack awareness, monitoring, backup, and tested partition reassignment runbooks.

Keep the app-facing contract stable: producers and consumers only depend on `kafka-svc:9092` and topic names.
