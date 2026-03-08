# Phase 6: Jaeger 链路追踪实战指南 (从0到1)

> "Prometheus 告诉你哪里慢，Jaeger 告诉你为什么慢。"

本阶段我们将接入 Jaeger 分布式链路追踪，实现对每一个请求的全链路可视化。

---

## 1. 原理篇

### 1.1 什么是链路追踪？

在微服务架构中，一个用户请求可能经过多个服务：

```
用户请求
  → Gateway (鉴权、路由)
    → Ranking RPC (业务逻辑)
      → Redis (数据读取)
```

**没有链路追踪**：你只知道"这个请求花了 50ms"，但不知道这 50ms 分布在哪里。

**有了链路追踪**：你能看到：
```
Gateway 总耗时: 50ms
  ├─ 鉴权中间件:    2ms
  ├─ gRPC 调用:    45ms
  │    ├─ 序列化:   1ms
  │    ├─ 网络传输: 2ms
  │    └─ RPC处理: 42ms
  │         └─ Redis ZRevRange: 40ms  ← 瓶颈在这里！
  └─ 序列化返回:    3ms
```

### 1.2 核心概念 (面试必背)

| 概念 | 含义 | 类比 |
|:---|:---|:---|
| **Trace** | 一次完整的请求链路 | 一次快递从发货到收货的全程 |
| **Span** | 链路中的一个操作单元 | 快递的每一个中转站 |
| **TraceID** | 唯一标识一次请求 | 快递单号 |
| **SpanID** | 唯一标识一个操作 | 每个中转站的扫描记录 |
| **Parent SpanID** | 父操作的 ID | 上一个中转站 |

**你有没有注意到** go-zero 的日志里一直有这两个字段？
```json
"span":"f4cafaf9af78da99"
"trace":"3ce976fc73d1c260d6df151f8de22242"
```
go-zero **早就在生成 Trace ID 了**！我们只需要告诉它把数据发给 Jaeger。

### 1.3 OpenTelemetry vs Jaeger

- **OpenTelemetry (OTel)**: 数据采集标准（go-zero 内置支持）。
- **Jaeger**: 数据存储和可视化后端。

关系类比：OTel 是"数据格式"，Jaeger 是"数据库+UI"。

---

## 2. 配置篇

### 2.1 启动 Jaeger 容器

在 `docker-compose.yml` 里添加 Jaeger 服务（使用 All-in-One 镜像，包含 Collector + UI）：

```yaml
jaeger:
  image: jaegertracing/all-in-one:latest
  restart: always
  ports:
    - "16686:16686"   # Jaeger UI
    - "4317:4317"     # OTLP gRPC (go-zero 默认使用这个端口上报数据)
    - "4318:4318"     # OTLP HTTP
  environment:
    COLLECTOR_OTLP_ENABLED: "true"
```

### 2.2 配置各服务的 Telemetry

在每个服务的 YAML 里添加 `Telemetry` 配置块：

**Gateway** (`app/gateway/etc/gateway.yaml`):
```yaml
Telemetry:
  Name: gateway           # 服务名，在 Jaeger UI 里显示
  Endpoint: http://localhost:4318/v1/traces  # Jaeger OTLP HTTP 端点
  Sampler: 1.0            # 采样率 1.0 = 100% 采样（生产环境建议 0.1）
  Batcher: jaeger         # 使用 jaeger 格式
```

**User RPC** (`app/user/cmd/rpc/etc/user.yaml`):
```yaml
Telemetry:
  Name: user-rpc
  Endpoint: http://localhost:4318/v1/traces
  Sampler: 1.0
  Batcher: jaeger
```

**Interaction RPC** (`app/interaction/cmd/rpc/etc/interaction.yaml`):
```yaml
Telemetry:
  Name: interaction-rpc
  Endpoint: http://localhost:4318/v1/traces
  Sampler: 1.0
  Batcher: jaeger
```

**Ranking RPC** (`app/ranking/cmd/rpc/etc/ranking.yaml`):
```yaml
Telemetry:
  Name: ranking-rpc
  Endpoint: http://localhost:4318/v1/traces
  Sampler: 1.0
  Batcher: jaeger
```

> **注意**: `Endpoint` 里用 `localhost:4318`，因为 Go 服务运行在宿主机，直接访问 Docker 映射出来的端口。

### 2.3 重启所有服务

修改 YAML 后，必须重启所有 Go 服务才能生效。

---

## 3. 验证篇

### 3.1 发送几个请求
```powershell
# 发几个请求触发 Trace 数据
./stress.exe -c 5 -n 10 -u "http://localhost:8888/v1/ranking/top?n=10"
```

### 3.2 打开 Jaeger UI
访问 `http://localhost:16686`

1. 在左侧 **Service** 下拉框选择 `gateway`
2. 点击 **Find Traces**
3. 你会看到一列 Trace 记录，每条对应一个 HTTP 请求

### 3.3 点击一条 Trace 查看详情
你会看到类似这样的瀑布图：
```
gateway: GET /v1/ranking/top          [50ms]
  └─ ranking-rpc: /ranking.Ranking/GetTop  [45ms]
       └─ redis: ZRevRange                  [40ms]
```

每一行都有精确的开始时间和持续时间。

---

## 4. 面试篇

### Q: 你们是怎么做链路追踪的？

**满分回答**:
"我们使用 **OpenTelemetry + Jaeger** 实现了全链路追踪。

go-zero 框架内置了 OTel 支持，只需要在 YAML 里配置 `Telemetry` 字段，框架会自动：
1. 为每个请求生成唯一的 **TraceID**。
2. 在服务间调用时自动传递 TraceID（通过 gRPC metadata）。
3. 将每个操作的 **Span** 数据上报给 Jaeger。

通过 Jaeger UI，我们能看到一个请求在 Gateway → Ranking RPC → Redis 各层的精确耗时。

**实际案例**：在一次压测中，我发现 Gateway P99 是 50ms，但 Ranking RPC P99 只有 4ms。通过 Jaeger 的 Trace 详情，我确认了 46ms 的差值主要来自 Gateway 的连接等待（因为我设置了 `MaxConns: 100` 的限流），而不是业务逻辑本身的问题。"

### Q: 采样率 (Sampler) 怎么设置？

**满分回答**:
"采样率是一个权衡：
- **1.0 (100%)**: 每个请求都记录，适合开发/测试环境。数据完整但存储压力大。
- **0.1 (10%)**: 每 10 个请求记录 1 个，适合生产环境。
- **0.01 (1%)**: 高 QPS 场景（如 10000 QPS），只记录 100 个/秒，减少存储压力。

我们项目开发阶段用 1.0，如果上生产会改为 0.1。"

---

## 5. 总结

| 工具 | 解决的问题 | 访问地址 |
|:---|:---|:---|
| Prometheus | 哪个服务/接口慢？QPS 多少？| `http://localhost:9090` |
| Grafana | 可视化 Prometheus 数据 | `http://localhost:3000` |
| Jaeger | 一个请求慢在哪一层？ | `http://localhost:16686` |

**三者配合**，构成完整的可观测性体系：
1. Grafana 告警：P99 超过 500ms
2. Prometheus 定位：是 Ranking RPC 慢了
3. Jaeger 确认：是 Redis ZRevRange 慢了（可能是热点 Key）

---

# 附录：真实排查过程 (踩坑记录)

> 这段排查过程本身就是面试素材。能说清楚"我踩了什么坑、怎么分析、怎么解决"，比背答案更有说服力。

## 坑 1：Jaeger UI 里只有 `jaeger-all-in-one`，没有业务服务

**现象**：配置了 `Batcher: jaeger`，重启服务后，Jaeger UI 的 Service 下拉框里只有 `jaeger-all-in-one`，没有 `gateway` 或 `ranking-rpc`。

**排查过程**：
```powershell
# 查询 Jaeger 已注册的服务列表
curl.exe -s "http://localhost:16686/api/services"
# 返回: {"data":["jaeger-all-in-one"],"total":1,...}
# 确认：业务服务的 Trace 数据根本没有到达 Jaeger
```

**根本原因**：
```
Batcher: jaeger  →  使用 Jaeger Thrift UDP 协议  →  默认端口 6831
```
但我们的 docker-compose 只开放了 OTLP 端口（4317/4318），没有开放 6831！
数据发出去了，但 Jaeger 容器收不到。

**解决方案**：把 `Batcher` 改为 `otlphttp`，走 4318 端口。

---

## 坑 2：HTTP 415 Unsupported Media Type

**现象**：改为 `Batcher: otlphttp` 后，Gateway 日志出现：
```
[otel] error: failed to upload traces; HTTP status code: 415
```

**排查过程**：
415 = 媒体类型不支持。说明 Jaeger 收到了请求，但数据格式不对。
查看 go-zero v1.6.3 的源码 `core/trace/agent.go`：
```go
case kindOtlpHttp:
    opts := []otlptracehttp.Option{
        otlptracehttp.WithInsecure(),
        otlptracehttp.WithEndpoint(c.Endpoint),  // ← 关键！
    }
```
`WithEndpoint` 接受的是 **`host:port`** 格式，而我们填的是完整 URL：
```yaml
Endpoint: http://localhost:4318/v1/traces  # ❌ 错误！
```
`otlptracehttp` 内部会自动拼接 `http://` 前缀，导致最终 URL 变成：
```
http://http://localhost:4318/v1/traces/v1/traces  # 双重前缀，格式错误
```
这就是 415 的原因：URL 解析失败，请求发到了错误的端点。

**解决方案**：拆分 Endpoint 和路径：
```yaml
Endpoint: localhost:4318       # 只填 host:port
OtlpHttpPath: /v1/traces       # 路径单独填
```

---

## 坑 3：URL 解析错误（双重编码）

**现象**：改完后日志出现：
```
[otel] error: traces export: parse "http://http:%2F%2Flocalhost:4318%2Fv1%2Ftraces/v1/traces": invalid port
```

**原因**：这是坑 2 的遗留问题。旧的 `Endpoint: http://localhost:4318/v1/traces` 被 URL 编码后拼接到新路径上，导致双重编码。

**解决方案**：确认 YAML 已经正确修改为 `Endpoint: localhost:4318`，重启服务即可。

---

## 最终正确配置

经过三次迭代，最终确认的正确配置如下：

```yaml
Telemetry:
  Name: gateway              # 服务名（在 Jaeger UI 里显示）
  Endpoint: localhost:4318   # ✅ 只填 host:port，不加 http:// 前缀
  Sampler: 1.0               # 采样率 100%
  Batcher: otlphttp          # ✅ 使用 OTLP HTTP 协议
  OtlpHttpPath: /v1/traces   # ✅ 路径单独配置
```

**配置对照表（避免再踩坑）**：

| Batcher 值 | 协议 | 需要开放的端口 | Endpoint 格式 |
|:---|:---|:---|:---|
| `jaeger` | Jaeger Thrift UDP | **6831** | `udp://host:6831` |
| `otlpgrpc` | OTLP gRPC | **4317** | `host:4317` |
| `otlphttp` | OTLP HTTP | **4318** | `host:4318` ✅ |

我们的 docker-compose 开放了 4317 和 4318，所以选 `otlphttp` 或 `otlpgrpc` 都可以。

---

## 成功结果

配置正确后，Jaeger UI (`http://localhost:16686`) 显示：
- **Service 列表**：`gateway`、`ranking-rpc` 出现 ✅
- **Trace 列表**：每条 Trace 显示 **3 Spans**（gateway 2个 + ranking-rpc 1个）✅
- **耗时**：约 1ms（本地环境，Redis 极快）✅

下一节 → **附录 2：如何在 Jaeger UI 里解读 Trace 详情**

---

# 附录 2：Jaeger UI 解读指南

## 1. Trace 列表页解读

打开 `http://localhost:16686`，选择 Service = `gateway`，点击 **Find Traces**。

你会看到类似这样的列表：
```
gateway: /v1/ranking/top  [3 Spans]  1.07ms   Today 5:23:31 am
gateway: /v1/ranking/top  [3 Spans]  1.08ms   Today 5:23:31 am
gateway: /v1/ranking/top  [3 Spans]  1.08ms   Today 5:23:31 am
```

**关键字段解读**：
| 字段 | 含义 |
|:---|:---|
| `gateway: /v1/ranking/top` | 服务名 + 操作名（接口路径）|
| `3 Spans` | 这条链路经过了 3 个操作单元 |
| `1.07ms` | 整条链路的总耗时 |
| 右侧色块 | 不同颜色代表不同服务（蓝=gateway，橙=ranking-rpc）|

---

## 2. Trace 详情页解读（瀑布图）

点击任意一条 Trace，进入详情页，你会看到**瀑布图**：

```
▼ gateway: GET /v1/ranking/top                    [0ms ────────────────── 1.07ms]
    ▼ gateway: /ranking.Ranking/GetTop            [0.1ms ──────────── 0.9ms]
        ▼ ranking-rpc: /ranking.Ranking/GetTop   [0.2ms ──────── 0.8ms]
```

**如何读这张图**：
- **横轴**：时间轴，从左到右代表时间流逝。
- **每一行**：一个 Span（操作单元）。
- **缩进关系**：子 Span 是父 Span 的一部分（调用关系）。
- **色块长度**：代表这个操作的耗时。

**我们系统的 3 个 Span 含义**：

| Span | 服务 | 含义 |
|:---|:---|:---|
| `GET /v1/ranking/top` | gateway | HTTP 请求从进入到返回的全程 |
| `/ranking.Ranking/GetTop` (client) | gateway | Gateway 发起 gRPC 调用的耗时（含网络） |
| `/ranking.Ranking/GetTop` (server) | ranking-rpc | Ranking RPC 处理请求的耗时（含 Redis） |

**关键差值分析**：
```
gateway HTTP 总耗时:          1.07ms
  └─ gRPC client 耗时:        0.90ms
       └─ RPC server 耗时:    0.80ms
            └─ (Redis 耗时):  ~0.70ms  ← 未显示，但可推算
```

---

## 3. 实战：用 Jaeger 定位慢请求

### 场景：压测时发现 P99 飙升，用 Jaeger 找原因

**Step 1**: 在 Grafana 发现 Gateway P99 从 5ms 突然升到 200ms。

**Step 2**: 打开 Jaeger，按耗时排序（Sort by: Longest First）：
```
gateway: /v1/ranking/top  [3 Spans]  198ms   ← 找到这条慢 Trace
gateway: /v1/ranking/top  [3 Spans]  2ms
gateway: /v1/ranking/top  [3 Spans]  1ms
```

**Step 3**: 点击 198ms 的 Trace，查看瀑布图：
```
▼ gateway: GET /v1/ranking/top                    [0ms ──────────────────── 198ms]
    ▼ gateway: /ranking.Ranking/GetTop            [1ms ─────────────────── 197ms]
        ▼ ranking-rpc: /ranking.Ranking/GetTop   [2ms ──────────────────── 196ms]
            (Redis ZRevRange 耗时)               [3ms ──────────────────── 195ms]
```

**结论**：196ms 都在 Ranking RPC 里，而且主要是 Redis 操作慢了。
可能原因：Redis 连接池耗尽（结合 Prometheus 的 `redis_client_pool_conn_total_current` 确认）。

---

## 4. 面试话术：链路追踪

### Q: 你们是怎么排查线上慢请求的？

**满分回答**：
"我们有一套三层排查体系：

**第一层（Grafana）**：发现问题。
看 Gateway 的 P99 延迟曲线，如果从正常的 5ms 突然飙到 200ms，说明有问题。

**第二层（Prometheus）**：缩小范围。
用 PromQL 查询各接口的 P99，找出是哪个接口慢了：
```promql
topk(3, histogram_quantile(0.99,
  sum(rate(http_server_requests_duration_ms_bucket[5m])) by (le, path)
))
```

**第三层（Jaeger）**：精确定位。
在 Jaeger 里找到对应时间段的慢 Trace，打开瀑布图，一眼看出是 Gateway 层慢、RPC 层慢，还是 Redis 层慢。

**实际案例**：在一次压测中，我发现 Gateway P99 是 146ms，但通过 Jaeger 的 Trace 详情，确认 Ranking RPC 自身只用了 4ms，差值 142ms 是 Gateway 的连接等待时间（因为我设置了 `MaxConns: 100` 的限流）。这说明问题不在业务逻辑，而在 Gateway 的并发控制配置。"

### Q: TraceID 有什么用？

**满分回答**：
"TraceID 是一个请求在整个系统中的唯一标识符。

**用途 1：关联日志**。
go-zero 的每条日志都带有 `trace` 字段，当用户反馈某个请求有问题时，我们可以用 TraceID 在日志系统里搜索这个请求经过的所有服务的所有日志，快速还原现场。

**用途 2：跨服务追踪**。
一个 TraceID 对应一条完整的调用链。即使请求经过了 5 个服务，我们也能在 Jaeger 里用这个 ID 找到完整的调用路径。

**实现原理**：
go-zero 在 Gateway 收到请求时生成 TraceID，然后通过 gRPC metadata 自动传递给下游 RPC 服务。每个服务都会把自己的处理过程作为一个 Span 上报给 Jaeger，Jaeger 用 TraceID 把所有 Span 串联成一条完整的链路。"

---

## 5. Phase 6 总结

**我们完成了什么**：
- ✅ 部署 Jaeger all-in-one 容器
- ✅ 配置 go-zero 的 OpenTelemetry 集成（`otlphttp` 协议）
- ✅ 在 Jaeger UI 看到了 `gateway` → `ranking-rpc` 的完整调用链
- ✅ 理解了如何用 Jaeger 定位慢请求

**踩坑收获**：
- `Batcher: jaeger` 走 UDP 6831，`Batcher: otlphttp` 走 HTTP 4318
- `otlphttp` 的 `Endpoint` 只填 `host:port`，路径用 `OtlpHttpPath` 单独配置
- 遇到问题先看日志里的 `[otel] error`，再查源码确认参数格式

**可观测性三件套已全部就位**：
```
Prometheus + Grafana  →  知道"哪里慢"
Jaeger                →  知道"为什么慢"
go-zero 结构化日志    →  知道"发生了什么"
```

