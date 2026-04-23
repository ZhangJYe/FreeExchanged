# Phase 4: Hive Offline Warehouse

## Scope

This phase adds a Hive offline analytics layer for Kafka event history. It does not put Hive on the online API path.

Added files:

- `analytics/hive/events.hql`
- `deploy/k8s/analytics/hive.yaml`

## Tables

Raw external tables:

- `freeexchanged.article_events_raw`
- `freeexchanged.interaction_events_raw`

Typed views:

- `freeexchanged.article_events`
- `freeexchanged.interaction_events`

The raw tables store one JSON event per line. Views parse fields with `get_json_object`.

## Deploy

```bash
kubectl delete job/hive-schema-init -n freeexchanged --ignore-not-found
kubectl apply -f deploy/k8s/analytics/hive.yaml
kubectl rollout status deployment/hive-server2 -n freeexchanged --timeout=180s
kubectl wait --for=condition=complete job/hive-schema-init -n freeexchanged --timeout=180s
```

HiveServer2 access:

```bash
kubectl port-forward svc/hive-server2 10000:10000 -n freeexchanged
```

Example query:

```bash
beeline -u jdbc:hive2://localhost:10000 -e "SELECT event_type, COUNT(*) FROM freeexchanged.interaction_events GROUP BY event_type"
```

## Data Ingestion

This phase creates the warehouse endpoint and schema. The next production step is to land Kafka events into the raw table locations with one of:

- Flink FileSink to object storage or HDFS.
- Kafka Connect S3/HDFS sink.
- Batch export jobs for backfill.

For the current local K8s skeleton, keep Hive as an offline validation surface and avoid making API services depend on Hive.

## Production Notes

- The manifest uses embedded Derby for the metastore, suitable only for development.
- Production should use an external metastore database such as MySQL or PostgreSQL.
- Warehouse storage should move from a single PVC to HDFS, S3, or another durable object store.
- If query concurrency grows, add Trino or Spark SQL instead of exposing HiveServer2 directly to users.
