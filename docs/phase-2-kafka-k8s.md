# Phase 2: Kafka Kubernetes Infrastructure

## Scope

This phase removes the Kubernetes RabbitMQ deployment and replaces it with a single-node Kafka KRaft deployment for the current project environment.

Added resources:

- `deploy/k8s/infra/kafka.yaml`
- `deploy/k8s/infra/kafka-topics-job.yaml`

Removed resource:

- `deploy/k8s/infra/rabbitmq.yaml`

## Runtime Topology

```text
article-rpc ----------> Kafka topic: article.events
interaction-rpc ------> Kafka topic: interaction.events
ranking-rpc ----------> Kafka consumer groups
ranking-rpc ----------> Redis sorted set: ranking:hot
```

## Kafka Settings

The manifest uses Bitnami Kafka in KRaft mode:

- One broker/controller for development and small demo clusters.
- Internal service: `kafka-svc:9092`.
- Headless service: `kafka-headless` for StatefulSet identity.
- Persistent volume: `5Gi`.
- Topic initialization Job creates:
  - `article.events`
  - `interaction.events`

## Deployment Script Changes

`deploy/k8s/deploy.sh` now:

1. Deletes the old `kafka-topic-init` Job if present.
2. Applies infrastructure.
3. Waits for MySQL, Kafka, Redis, and topic initialization.
4. Runs DB migration.
5. Deploys application workloads.

## Production Notes

The current Kafka manifest is intentionally small. For production, prefer one of these:

- Managed Kafka from the cloud provider.
- Strimzi Kafka Operator.
- A three-broker StatefulSet with proper storage, rack awareness, and replication factor greater than one.

Keep the app-facing contract stable: producers and consumers only depend on `kafka-svc:9092` and topic names.
