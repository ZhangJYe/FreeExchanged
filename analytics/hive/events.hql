CREATE DATABASE IF NOT EXISTS freeexchanged;

CREATE EXTERNAL TABLE IF NOT EXISTS freeexchanged.article_events_raw (
  json STRING
)
STORED AS TEXTFILE
LOCATION '/opt/hive/warehouse/freeexchanged/article_events_raw';

CREATE EXTERNAL TABLE IF NOT EXISTS freeexchanged.interaction_events_raw (
  json STRING
)
STORED AS TEXTFILE
LOCATION '/opt/hive/warehouse/freeexchanged/interaction_events_raw';

CREATE VIEW IF NOT EXISTS freeexchanged.article_events AS
SELECT
  get_json_object(json, '$.event_id') AS event_id,
  get_json_object(json, '$.event_type') AS event_type,
  CAST(get_json_object(json, '$.version') AS INT) AS version,
  CAST(get_json_object(json, '$.article_id') AS BIGINT) AS article_id,
  get_json_object(json, '$.title') AS title,
  CAST(get_json_object(json, '$.author_id') AS BIGINT) AS author_id,
  CAST(get_json_object(json, '$.occurred_at') AS BIGINT) AS occurred_at
FROM freeexchanged.article_events_raw;

CREATE VIEW IF NOT EXISTS freeexchanged.interaction_events AS
SELECT
  get_json_object(json, '$.event_id') AS event_id,
  get_json_object(json, '$.event_type') AS event_type,
  CAST(get_json_object(json, '$.version') AS INT) AS version,
  CAST(get_json_object(json, '$.user_id') AS BIGINT) AS user_id,
  CAST(get_json_object(json, '$.article_id') AS BIGINT) AS article_id,
  CAST(get_json_object(json, '$.occurred_at') AS BIGINT) AS occurred_at
FROM freeexchanged.interaction_events_raw;
