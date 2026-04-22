package com.freeexchanged.analytics;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.Serializable;
import java.time.Instant;
import java.util.Objects;
import org.apache.flink.api.common.eventtime.WatermarkStrategy;
import org.apache.flink.api.common.serialization.SimpleStringSchema;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.connector.kafka.source.KafkaSource;
import org.apache.flink.connector.kafka.source.enumerator.initializer.OffsetsInitializer;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.environment.StreamExecutionEnvironment;
import org.apache.flink.streaming.api.functions.sink.RichSinkFunction;
import redis.clients.jedis.Jedis;

public class RankingJob {
  private static final String ARTICLE_TOPIC = "article.events";
  private static final String INTERACTION_TOPIC = "interaction.events";

  public static void main(String[] args) throws Exception {
    String bootstrapServers = env("KAFKA_BOOTSTRAP_SERVERS", "kafka-svc:9092");
    String groupId = env("KAFKA_GROUP_ID", "flink-ranking-consumer");
    String redisHost = env("REDIS_HOST", "redis-svc");
    int redisPort = Integer.parseInt(env("REDIS_PORT", "6379"));
    String rankingKey = env("RANKING_REDIS_KEY", "ranking:hot:flink");

    StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();
    env.enableCheckpointing(30000);

    KafkaSource<String> source = KafkaSource.<String>builder()
        .setBootstrapServers(bootstrapServers)
        .setTopics(ARTICLE_TOPIC, INTERACTION_TOPIC)
        .setGroupId(groupId)
        .setStartingOffsets(OffsetsInitializer.latest())
        .setValueOnlyDeserializer(new SimpleStringSchema())
        .build();

    DataStream<RankingEvent> events = env
        .fromSource(source, WatermarkStrategy.noWatermarks(), "freeexchanged-kafka-events")
        .map(RankingEvent::fromJson)
        .filter(Objects::nonNull);

    events.addSink(new RedisRankingSink(redisHost, redisPort, rankingKey)).name("redis-ranking-sink");

    env.execute("freeexchanged-ranking-job");
  }

  private static String env(String key, String defaultValue) {
    String value = System.getenv(key);
    return value == null || value.isBlank() ? defaultValue : value;
  }

  static final class RankingEvent implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final ObjectMapper MAPPER = new ObjectMapper();

    final String eventType;
    final long articleId;
    final long score;
    final long delta;
    final boolean absoluteScore;

    private RankingEvent(String eventType, long articleId, long score, long delta, boolean absoluteScore) {
      this.eventType = eventType;
      this.articleId = articleId;
      this.score = score;
      this.delta = delta;
      this.absoluteScore = absoluteScore;
    }

    static RankingEvent fromJson(String json) throws Exception {
      JsonNode root = MAPPER.readTree(json);
      String eventType = root.path("event_type").asText("");
      long articleId = root.path("article_id").asLong(0);
      if (articleId <= 0) {
        return null;
      }

      switch (eventType) {
        case "article.published":
        case "publish":
          long occurredAt = root.path("occurred_at").asLong(Instant.now().getEpochSecond());
          return new RankingEvent(eventType, articleId, occurredAt, 0, true);
        case "interaction.like":
        case "like":
          return new RankingEvent(eventType, articleId, 0, 10, false);
        case "interaction.unlike":
        case "unlike":
          return new RankingEvent(eventType, articleId, 0, -10, false);
        case "interaction.read":
        case "read":
          return new RankingEvent(eventType, articleId, 0, 1, false);
        default:
          return null;
      }
    }
  }

  static final class RedisRankingSink extends RichSinkFunction<RankingEvent> implements Serializable {
    private static final long serialVersionUID = 1L;
    private final String host;
    private final int port;
    private final String rankingKey;
    private transient Jedis jedis;

    RedisRankingSink(String host, int port, String rankingKey) {
      this.host = host;
      this.port = port;
      this.rankingKey = rankingKey;
    }

    @Override
    public void open(Configuration parameters) {
      this.jedis = new Jedis(host, port);
    }

    @Override
    public void invoke(RankingEvent value, Context context) {
      String articleId = Long.toString(value.articleId);
      if (value.absoluteScore) {
        jedis.zadd(rankingKey, value.score, articleId);
      } else {
        jedis.zincrby(rankingKey, value.delta, articleId);
      }
    }

    @Override
    public void close() {
      if (jedis != null) {
        jedis.close();
      }
    }
  }
}
