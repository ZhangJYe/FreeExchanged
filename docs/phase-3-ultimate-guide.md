# Phase 3: Prometheus + Grafana 全链路监控指南 (从0到1)

> 本文档是 Phase 3 的权威指南，整合了所有监控相关内容。分为四个部分：
> - **Part 1**: 原理篇 —— 搞懂监控体系的设计思想
> - **Part 2**: 配置篇 —— 让服务暴露数据
> - **Part 3**: 可视化篇 —— Prometheus + Grafana 实战
> - **Part 4**: 面试篇 —— 用监控数据讲故事

---

# Part 1: 原理篇

## 1.1 为什么需要监控？

想象一下，你的系统上线了，用户开始使用。突然有人反馈"系统很慢"。
- **没有监控的你**: 打开日志，一行一行翻，不知道从哪里开始。
- **有监控的你**: 打开 Grafana，5秒内定位到 Ranking RPC 的 P99 延迟在 3 分钟前突然从 10ms 飙升到 500ms，同时 Redis 连接池耗尽。

这就是监控的价值：**把不可见的问题变成可见的图表**。

---

## 1.2 监控体系的三大支柱 (面试必背)

业界公认的可观测性 (Observability) 由三部分组成：

| 支柱 | 工具 | 回答的问题 |
|:---|:---|:---|
| **Metrics (指标)** | Prometheus + Grafana | 系统现在**健不健康**？QPS 多少？延迟多少？|
| **Logging (日志)** | ELK / Loki | 某个请求**发生了什么**？报错信息是什么？|
| **Tracing (链路追踪)** | Jaeger / Zipkin | 一个请求**经过了哪些服务**？慢在哪里？|

我们 Phase 3 专注于 **Metrics**，这是最基础也是最重要的一环。

---

## 1.3 Prometheus 的工作原理

### Pull 模型 (拉取模式)

Prometheus 和大多数监控系统不同，它采用 **Pull (拉取)** 模式：

```
你的服务 (:9091/metrics)  <───── Prometheus 每10秒来抓一次
你的服务 (:9092/metrics)  <───── Prometheus 每10秒来抓一次
你的服务 (:9093/metrics)  <───── Prometheus 每10秒来抓一次
```

**对比 Push 模式**:
- Push: 服务主动把数据推给监控中心 → 服务需要知道监控中心在哪里，耦合度高。
- Pull: 监控中心主动来拉 → 服务只需要暴露一个 HTTP 端口，解耦。

**面试题**: 为什么 Prometheus 选择 Pull 模式？
> 答: Pull 模式更适合云原生环境。服务不需要关心监控系统的地址，只需要暴露 `/metrics` 端口。Prometheus 通过 Service Discovery (服务发现，如 Consul/K8s) 自动找到所有服务并拉取数据。

---

## 1.4 Metrics 的四种类型 (面试必背)

go-zero 暴露的指标，都属于以下四种类型之一：

### Counter (计数器)
- **特点**: 只增不减，服务重启后归零。
- **例子**: `http_server_requests_code_total` (总请求数)
- **用途**: 计算 QPS = `rate(counter[1m])`

### Gauge (仪表盘)
- **特点**: 可增可减，反映当前状态。
- **例子**: `go_goroutines` (当前 Goroutine 数量)
- **用途**: 直接查看当前值

### Histogram (直方图)
- **特点**: 把数据分桶统计，可以计算百分位数 (P99)。
- **例子**: `http_server_requests_duration_ms_bucket` (请求耗时分布)
- **用途**: `histogram_quantile(0.99, ...)` 计算 P99 延迟

### Summary (摘要)
- **特点**: 类似 Histogram，但百分位数在客户端计算。
- **用途**: 较少使用，Histogram 更灵活。

---

## 1.5 我们系统中的关键指标

通过之前的验证，我们的系统已经暴露了以下指标：

| 指标名 | 类型 | 含义 |
|:---|:---|:---|
| `http_server_requests_duration_ms_count` | Counter | Gateway HTTP 请求总数 |
| `http_server_requests_duration_ms_bucket` | Histogram | Gateway HTTP 请求耗时分布 |
| `http_server_requests_code_total` | Counter | Gateway HTTP 按状态码统计 |
| `rpc_server_requests_duration_ms_count` | Counter | RPC Server 请求总数 |
| `rpc_server_requests_duration_ms_bucket` | Histogram | RPC Server 请求耗时分布 |
| `rpc_client_requests_duration_ms_count` | Counter | RPC Client 调用总数 |
| `redis_client_requests_duration_ms_count` | Counter | Redis 操作总数 |
| `redis_client_requests_duration_ms_bucket` | Histogram | Redis 操作耗时分布 |
| `redis_client_pool_conn_total_current` | Gauge | Redis 连接池当前连接数 |
| `go_goroutines` | Gauge | 当前 Goroutine 数量 |
| `go_memstats_heap_alloc_bytes` | Gauge | 堆内存占用 |

---

**Part 1 小结**:
- Prometheus 是 Pull 模式，主动抓取服务的 `/metrics`。
- Metrics 有四种类型：Counter、Gauge、Histogram、Summary。
- go-zero 自动暴露了 HTTP、RPC、Redis 三层的指标，无需写代码。

下一节 → **Part 2: 配置篇**

---

# Part 2: 配置篇

## 2.1 整体配置流程

配置监控体系只需要三步：

```
Step 1: 在每个 Go 服务的 YAML 里开启 Prometheus 端口
         ↓
Step 2: 编写 prometheus.yml，告诉 Prometheus 去哪里抓数据
         ↓
Step 3: 用 docker-compose 启动 Prometheus 和 Grafana 容器
```

---

## 2.2 Step 1: 开启服务的 Metrics 端口

**原理**: go-zero 框架内置了 Prometheus 支持。只要在 YAML 里加上 `Prometheus` 配置块，框架会自动：
1. 启动一个独立的 HTTP Server（监听你指定的端口）。
2. 在 `/metrics` 路径上暴露所有指标。
3. 自动注入 HTTP/RPC 拦截器，统计每个接口的请求数和耗时。

**你不需要写任何 Go 代码！**

### Gateway (`app/gateway/etc/gateway.yaml`)
```yaml
Name: gateway
Host: 0.0.0.0
Port: 8888

# 新增这一块 ↓
Prometheus:
  Host: 0.0.0.0
  Port: 9091        # 对外暴露 /metrics 的端口
  Path: /metrics    # 固定写 /metrics
```

### User RPC (`app/user/cmd/rpc/etc/user.yaml`)
```yaml
Name: user.rpc
ListenOn: 0.0.0.0:8080

# 新增这一块 ↓
Prometheus:
  Host: 0.0.0.0
  Port: 9092
  Path: /metrics
```

### Interaction RPC (`app/interaction/cmd/rpc/etc/interaction.yaml`)
```yaml
Name: interaction.rpc
ListenOn: 0.0.0.0:8081

# 新增这一块 ↓
Prometheus:
  Host: 0.0.0.0
  Port: 9093
  Path: /metrics
```

### Ranking RPC (`app/ranking/cmd/rpc/etc/ranking.yaml`)
```yaml
Name: ranking.rpc
ListenOn: 0.0.0.0:8082

# 新增这一块 ↓
Prometheus:
  Host: 0.0.0.0
  Port: 9094
  Path: /metrics
```

**端口规划总结**:
| 服务 | 业务端口 | Metrics 端口 |
|:---|:---|:---|
| Gateway | 8888 | **9091** |
| User RPC | 8080 | **9092** |
| Interaction RPC | 8081 | **9093** |
| Ranking RPC | 8082 | **9094** |

> ⚠️ **注意**: 修改 YAML 后必须**重启对应的 Go 服务**才能生效！

---

## 2.3 Step 2: 配置 Prometheus 抓取目标

**文件位置**: `deploy/prometheus/prometheus.yml`

```yaml
global:
  scrape_interval: 10s     # 每 10 秒抓取一次数据

scrape_configs:
  # Gateway 的 HTTP 指标
  - job_name: 'gateway'
    static_configs:
      - targets: ['host.docker.internal:9091']
        # host.docker.internal 是 Docker Desktop 提供的特殊域名
        # 它指向宿主机的 localhost
        # 因为 Prometheus 在容器里，不能直接用 localhost 访问宿主机的服务

  # User RPC 的指标
  - job_name: 'user-rpc'
    static_configs:
      - targets: ['host.docker.internal:9092']

  # Interaction RPC 的指标
  - job_name: 'interaction-rpc'
    static_configs:
      - targets: ['host.docker.internal:9093']

  # Ranking RPC 的指标
  - job_name: 'ranking-rpc'
    static_configs:
      - targets: ['host.docker.internal:9094']
```

**关键知识点: `host.docker.internal` 是什么？**

```
┌─────────────────────────────────────────────┐
│  宿主机 (你的 Windows 电脑)                   │
│                                              │
│  go run gateway.go  → 监听 0.0.0.0:9091     │
│  go run user.go     → 监听 0.0.0.0:9092     │
│                                              │
│  ┌────────────────────────────────────────┐  │
│  │  Docker 容器网络                        │  │
│  │                                        │  │
│  │  Prometheus 容器                       │  │
│  │  想访问宿主机的 9091 端口...            │  │
│  │  用 host.docker.internal:9091 ✅       │  │
│  │  用 localhost:9091 ❌ (这是容器自己)   │  │
│  └────────────────────────────────────────┘  │
└─────────────────────────────────────────────┘
```

---

## 2.4 Step 3: 启动 Prometheus 和 Grafana

**文件位置**: `docker-compose.yml` (已经配置好了)

```yaml
prometheus:
  image: prom/prometheus
  ports:
    - "9090:9090"           # 宿主机 9090 → 容器 9090
  volumes:
    - ./deploy/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    # 把我们写的配置文件挂载进容器

grafana:
  image: grafana/grafana
  ports:
    - "3000:3000"           # 宿主机 3000 → 容器 3000
  environment:
    GF_SECURITY_ADMIN_PASSWORD: admin
  depends_on:
    - prometheus
```

**启动命令**:
```powershell
# 启动（如果已经在运行，这个命令会更新配置）
docker-compose up -d prometheus grafana

# 如果修改了 prometheus.yml，需要强制重建容器
docker rm -f freeexchanged-prometheus-1
docker-compose up -d prometheus
```

---

## 2.5 验证配置是否生效

### 验证 1: 服务的 Metrics 端口是否开启
直接用浏览器或 curl 访问：
```
http://localhost:9091/metrics   → Gateway 的指标
http://localhost:9092/metrics   → User RPC 的指标
http://localhost:9094/metrics   → Ranking RPC 的指标
```

你应该看到大量文本，类似：
```
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 22
# HELP http_server_requests_duration_ms_count ...
http_server_requests_duration_ms_count{method="GET",path="/v1/ranking/top"} 10000
```

### 验证 2: Prometheus 是否成功抓取
打开 `http://localhost:9090/targets`

你应该看到类似：
```
gateway          http://host.docker.internal:9091/metrics   UP   (绿色)
user-rpc         http://host.docker.internal:9092/metrics   UP   (绿色)
interaction-rpc  http://host.docker.internal:9093/metrics   UP   (绿色)
ranking-rpc      http://host.docker.internal:9094/metrics   UP   (绿色)
```

如果是 **DOWN (红色)**，常见原因：
- Go 服务没有重启（YAML 改了但没生效）
- 端口被防火墙拦截
- `host.docker.internal` 解析失败（Linux 下需要特殊配置）

---

**Part 2 小结**:
- go-zero 只需在 YAML 加 `Prometheus` 配置，无需写代码。
- Prometheus 在 Docker 里，用 `host.docker.internal` 访问宿主机服务。
- 验证方法：先看 `/metrics` 端口有没有数据，再看 Prometheus Targets 是否 UP。

下一节 → **Part 3: 可视化篇 (Prometheus PromQL + Grafana 实战)**

---

# Part 3: 可视化篇

## 3.1 PromQL 入门 —— Prometheus 的查询语言

PromQL 是 Prometheus 专用的查询语言。你在 Grafana 里画的每一张图，背后都是一条 PromQL 语句。

**打开练习场**: `http://localhost:9090/graph`
在搜索框里输入下面的语句，点 **Execute**，然后切换到 **Graph** 标签看曲线。

---

### 3.1.1 最简单的查询：直接查指标名

```promql
go_goroutines
```
**含义**: 查询所有服务当前的 Goroutine 数量。
**返回**: 每个服务一行数据（带 `job` 和 `instance` 标签）。

```promql
go_goroutines{job="gateway"}
```
**含义**: 只看 Gateway 的 Goroutine 数量。
**语法**: `{}` 里是**标签过滤器**，精确筛选你想看的数据。

---

### 3.1.2 最重要的函数：`rate()` 计算 QPS

Counter 类型的指标（如请求总数）是单调递增的，直接看没意义。
我们需要用 `rate()` 函数计算**每秒增长速率**，这才是 QPS。

```promql
rate(http_server_requests_duration_ms_count[1m])
```
**含义**: 计算过去 1 分钟内，Gateway HTTP 请求数的**每秒平均增长速率**，即 QPS。
**`[1m]`**: 时间窗口，表示"看过去 1 分钟的数据"。

```promql
sum(rate(http_server_requests_duration_ms_count[1m])) by (path)
```
**含义**: 按接口路径分组，分别计算每个接口的 QPS。
**`sum(...) by (path)`**: 聚合函数，把相同 `path` 的数据加在一起。

**实战**: 压测后执行这条语句，你会看到 `/v1/ranking/top` 的 QPS 曲线飙升！

---

### 3.1.3 最有价值的查询：`histogram_quantile()` 计算 P99

P99 延迟是衡量系统性能最重要的指标。

```promql
histogram_quantile(0.99,
  sum(rate(http_server_requests_duration_ms_bucket[1m])) by (le, path)
)
```
**含义**: 计算过去 1 分钟内，各接口的 **P99 延迟**（单位 ms）。
**`0.99`**: 第 99 百分位，即 99% 的请求都在这个时间内完成。
**`le`**: Histogram 的特殊标签，表示"less than or equal to"（桶的上界）。

**其他百分位**:
```promql
histogram_quantile(0.95, ...)   # P95
histogram_quantile(0.50, ...)   # P50 (中位数)
```

---

### 3.1.4 常用 PromQL 速查表

| 场景 | PromQL |
|:---|:---|
| Gateway QPS | `sum(rate(http_server_requests_duration_ms_count[1m])) by (path)` |
| Gateway P99 延迟 | `histogram_quantile(0.99, sum(rate(http_server_requests_duration_ms_bucket[1m])) by (le, path))` |
| RPC Server QPS | `sum(rate(rpc_server_requests_duration_ms_count[1m])) by (instance, method)` |
| RPC Server P99 延迟 | `histogram_quantile(0.99, sum(rate(rpc_server_requests_duration_ms_bucket[1m])) by (le, instance, method))` |
| Redis 操作 QPS | `sum(rate(redis_client_requests_duration_ms_count[1m])) by (command)` |
| Redis P99 延迟 | `histogram_quantile(0.99, sum(rate(redis_client_requests_duration_ms_bucket[1m])) by (le, command))` |
| 各服务 Goroutine 数 | `go_goroutines` |
| 各服务堆内存 (MB) | `go_memstats_heap_alloc_bytes / 1024 / 1024` |
| HTTP 错误率 | `sum(rate(http_server_requests_code_total{code!="200"}[1m])) by (path, code)` |
| Redis 连接池使用率 | `redis_client_pool_conn_total_current / redis_client_pool_conn_max` |

---

## 3.2 Grafana 实战 —— 从0配置到看到图表

### Step 1: 登录 Grafana
打开 `http://localhost:3000`
- 账号: `admin`
- 密码: `admin`（首次登录会让你修改，可以直接 Skip）

---

### Step 2: 添加 Prometheus 数据源

1. 左侧菜单 → **Connections** → **Data Sources**
2. 点击 **Add new data source**
3. 搜索并选择 **Prometheus**
4. 填写配置：
   ```
   Name:  Prometheus
   URL:   http://prometheus:9090
   ```
   > ⚠️ **关键**: URL 必须填 `http://prometheus:9090`，不能填 `localhost:9090`！
   > 因为 Grafana 在容器里，`prometheus` 是同一个 docker-compose 网络里的服务名。

5. 点击 **Save & Test**
6. 看到 ✅ **"Successfully queried the Prometheus API"** 即成功

---

### Step 3: 导入自定义 Dashboard

我们已经为你准备好了专门适配本项目指标的 Dashboard JSON 文件。

1. 左侧菜单 → **Dashboards** → **Import**
2. 点击 **"Upload dashboard JSON file"**
3. 选择文件：`deploy/grafana/freeexchanged-dashboard.json`
4. 在 **Prometheus** 下拉框选择刚才配置的数据源
5. 点击 **Import**

---

### Step 4: 让数据动起来（压测）

导入后如果显示 "No Data"，是因为没有流量。执行压测：

```powershell
# 在项目根目录执行
./stress.exe -c 50 -n 200 -u "http://localhost:8888/v1/ranking/top?n=10"
```

然后在 Grafana 右上角：
- 时间范围改为 **"Last 5 minutes"**
- 刷新间隔改为 **"5s"**（Auto refresh）

---

### Step 5: 解读 Dashboard 的 9 个面板

```
┌──────────────────────┬──────────────────────┐
│  Gateway HTTP QPS    │  Gateway HTTP P99    │
│  (每秒请求数)         │  (99%请求的延迟)      │
├──────────────────────┼──────────────────────┤
│  RPC Server QPS      │  RPC Server P99      │
│  (Ranking/User调用量) │  (RPC调用延迟)        │
├──────────────────────┼──────────────────────┤
│  Redis 操作 QPS      │  Redis P99 延迟      │
│  (ZRevRange等操作量)  │  (Redis响应时间)      │
├──────────────────────┼──────────────────────┤
│  各服务 Goroutine 数  │  各服务内存占用(MB)   │
├──────────────────────┴──────────────────────┤
│  HTTP 错误率 (非200状态码)                   │
└─────────────────────────────────────────────┘
```

**压测时你应该观察到**：
1. **Gateway HTTP QPS** → 从 0 飙升到 ~2400 req/s
2. **RPC Server QPS** → Ranking RPC 的 GetTop 同步飙升
3. **Redis 操作 QPS** → ZRevRange 操作量飙升
4. **Gateway HTTP P99** → 应该稳定在 50ms 以内（说明系统健康）
5. **Goroutine 数量** → 压测期间短暂增加，压测结束后回落
6. **HTTP 错误率** → 应该一直是 0（说明没有报错）

---

## 3.3 PromQL 进阶：在 Prometheus Graph 里自己画图

除了用我们提供的 Dashboard，你也可以在 `http://localhost:9090/graph` 里自己探索。

### 实战练习 1：找出最慢的接口
```promql
topk(3,
  histogram_quantile(0.99,
    sum(rate(http_server_requests_duration_ms_bucket[5m])) by (le, path)
  )
)
```
**含义**: 找出 P99 延迟最高的 3 个接口。

### 实战练习 2：Redis 连接池健康度
```promql
redis_client_pool_conn_total_current / redis_client_pool_conn_max
```
**含义**: 连接池使用率。如果接近 1.0（100%），说明连接池快耗尽了，需要扩容。

### 实战练习 3：服务是否存活
```promql
up
```
**含义**: 所有被监控服务的存活状态。1 = UP，0 = DOWN。
这是最简单的告警规则：如果 `up == 0`，立刻报警！

---

**Part 3 小结**：
- `rate()` 把 Counter 变成 QPS，是最常用的函数。
- `histogram_quantile(0.99, ...)` 计算 P99 延迟，是衡量性能的黄金指标。
- Grafana 数据源 URL 必须用容器名 `http://prometheus:9090`。
- 我们的自定义 Dashboard 覆盖了 HTTP → RPC → Redis 全链路。

下一节 → **Part 4: 面试篇 —— 用监控数据讲故事**

---

# Part 4: 面试篇

> "普通候选人说：'我做了一个排行榜，用了 Redis ZSet。'
> 优秀候选人说：'我做了一个排行榜，压测时 QPS 达到 2458，P99 延迟 37ms，通过 Prometheus 监控发现 Redis 连接池使用率在高并发时达到 80%，于是我将连接池从默认的 10 调整到 50，P99 降到了 22ms。'"
>
> 同样的项目，差距在于**你有没有数据支撑**。

---

## 4.1 面试高频问题与满分回答

### Q1: 你的项目如何保证高可用？

**普通回答**: "我用了微服务架构，服务之间解耦了。"

**满分回答**:
"我们的系统有三层保障：

**第一层：服务健康监控**。每个服务都接入了 Prometheus，我设置了 `up == 0` 的告警规则。一旦某个服务挂掉，Prometheus 在 10 秒内就能检测到（我们的 scrape_interval 是 10s），并触发告警。

**第二层：流量保护**。go-zero 内置了自适应限流（基于 CPU 使用率），当 CPU 超过 90% 时自动拒绝部分请求，防止雪崩。在 Grafana 的 `shedding_stat` 面板里可以看到被降级的请求数。

**第三层：异步削峰**。点赞这种高频写操作通过 RabbitMQ 异步化。即使 Ranking Service 短暂不可用，消息会堆积在队列里，等服务恢复后继续消费，不会丢数据。"

---

### Q2: 你的系统 QPS 能达到多少？瓶颈在哪里？

**满分回答**:
"我用 `go-stress-testing` 做了基准测试：

- **排行榜接口** (`GET /v1/ranking/top`): 50 并发下 QPS 达到 **2458 req/s**，TP99 **37ms**，成功率 100%。
- **瓶颈分析**: 通过 Prometheus 的 `redis_client_pool_conn_total_current` 指标，我发现在高并发时 Redis 连接池使用率接近上限。理论上，如果继续加并发，Redis 连接池会成为第一个瓶颈。
- **优化方向**: 
  1. 增大 Redis 连接池（go-zero 的 `redis.RedisConf` 里调整 `MaxActive`）。
  2. 在 Ranking Service 里加一层本地缓存（`sync.Map` 或 `ristretto`），热点数据不走 Redis，QPS 可以再提升 3-5 倍。"

---

### Q3: 如何定位线上慢请求？

**满分回答**:
"我有一套完整的排查流程：

**第一步：看 Grafana 大盘**。
先看 Gateway 的 P99 延迟是否异常。如果 P99 从正常的 10ms 突然飙到 500ms，说明有问题。

**第二步：用 PromQL 缩小范围**。
```promql
topk(3, histogram_quantile(0.99,
  sum(rate(http_server_requests_duration_ms_bucket[5m])) by (le, path)
))
```
这条语句能找出 P99 最高的 3 个接口，快速定位是哪个接口慢。

**第三步：看 RPC 层**。
如果 Gateway 慢，但 Gateway 自身的 CPU/内存正常，那问题一定在下游 RPC。
查看 `rpc_client_requests_duration_ms` 指标，看是哪个 RPC 调用慢。

**第四步：看 Redis 层**。
如果 RPC 慢，查看 `redis_client_requests_duration_ms`，看是不是 Redis 操作慢了。
结合 `redis_client_pool_conn_total_current`，判断是否是连接池耗尽导致排队。

**第五步：看 Goroutine 数量**。
如果 `go_goroutines` 持续增长不回落，说明有 Goroutine 泄漏，需要 pprof 进一步分析。"

---

### Q4: Prometheus 的 Pull 模式有什么缺点？你怎么解决？

**满分回答**:
"Pull 模式有两个主要缺点：

**缺点 1：短生命周期任务无法被抓取**。
比如一个 Job 跑 5 秒就结束了，而 Prometheus 每 10 秒才来抓一次，这个 Job 的数据就丢了。
**解决方案**: 使用 `Pushgateway`。Job 主动把数据推给 Pushgateway，Prometheus 再从 Pushgateway 拉取。

**缺点 2：大规模服务发现复杂**。
如果有几百个服务，手动在 `prometheus.yml` 里写 `static_configs` 不现实。
**解决方案**: 使用 Service Discovery。我们项目用了 Consul 做注册中心，Prometheus 可以配置 `consul_sd_configs`，自动发现所有注册到 Consul 的服务，无需手动维护配置。"

---

### Q5: P99 和平均延迟有什么区别？为什么要看 P99？

**满分回答**:
"这是一个非常好的问题。

**平均延迟的问题**: 假设 100 个请求里，99 个耗时 1ms，1 个耗时 10000ms（超时），平均延迟是 `(99*1 + 10000) / 100 ≈ 101ms`。这个数字看起来还好，但实际上有 1% 的用户等了 10 秒！

**P99 的意义**: P99 = 37ms 意味着 99% 的用户在 37ms 内得到了响应，只有 1% 的用户可能更慢。这才是真实的用户体验指标。

**为什么关注长尾**:
在微服务架构中，一个请求可能经过 5 个服务。如果每个服务的 P99 是 50ms，那这个请求的 P99 可能是 250ms（5个服务串联）。所以每个服务都要把 P99 控制得足够低。

**我们项目的数据**:
- Gateway P99: ~37ms（包含了 RPC 调用时间）
- Ranking RPC P99: 预计 < 10ms（纯 Redis 读取）
- Redis P99: 预计 < 5ms（内存操作）"

---

## 4.2 监控相关的加分项

### 加分项 1: 告警配置
面试时可以提到你配置了 Prometheus Alertmanager：
```yaml
# 示例告警规则 (deploy/prometheus/alert.rules.yml)
groups:
  - name: service_alerts
    rules:
      # 服务挂了
      - alert: ServiceDown
        expr: up == 0
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: "服务 {{ $labels.job }} 已宕机"

      # P99 延迟过高
      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(http_server_requests_duration_ms_bucket[5m])) > 500
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "接口 {{ $labels.path }} P99 延迟超过 500ms"

      # 错误率过高
      - alert: HighErrorRate
        expr: rate(http_server_requests_code_total{code!="200"}[5m]) > 0.05
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "接口错误率超过 5%"
```

### 加分项 2: 自定义业务指标
go-zero 暴露的是框架级别的指标，你还可以埋入业务指标：
```go
import "github.com/prometheus/client_golang/prometheus"

// 定义一个业务 Counter
var likeCounter = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Name: "business_like_total",
        Help: "Total number of like actions",
    },
    []string{"article_id"},
)

func init() {
    prometheus.MustRegister(likeCounter)
}

// 在点赞逻辑里调用
likeCounter.WithLabelValues(fmt.Sprint(articleId)).Inc()
```
这样你就能在 Grafana 里看到每篇文章的点赞趋势！

---

## 4.3 总结：你的项目亮点清单

面试时，你可以自信地说出以下亮点：

✅ **架构层面**
- 微服务架构（User / Interaction / Ranking / Gateway）
- 服务注册与发现（Consul）
- 统一 API 网关（go-zero REST）
- Paseto Token 鉴权

✅ **性能层面**
- RabbitMQ 异步削峰（点赞写操作）
- Redis ZSet 实时排行榜（O(logN) 复杂度）
- 压测验证：50并发 QPS 2458，TP99 37ms

✅ **可观测性层面**
- Prometheus 全链路监控（HTTP → RPC → Redis）
- 自定义 Grafana Dashboard（9个面板）
- PromQL 查询：QPS、P99、错误率、连接池使用率

✅ **工程化层面**
- Docker Compose 一键启动基础设施
- go-stress-testing 基准测试
- 完整的开发文档（Phase 1-4）

---

**Phase 3 全部完成！**

你现在拥有了一套完整的微服务监控体系，以及一套完整的面试话术。
祝你面试顺利！🎉
