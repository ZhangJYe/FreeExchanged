# Phase 4: 使用 go-stress-testing 进行工程化压测

> "上线前的最后一道防线，不是测试人员，而是压测报告。"

在 Phase 3 中，我们通过 Grafana 看到了系统的实时状态。现在，我们要主动向系统施加压力，找出它的崩溃点。本阶段我们将使用业界优秀的开源工具 **go-stress-testing**。

---

## 1. 工具介绍 (面试题: 为什么选它?)

*   **go-stress-testing** (`link1st/go-stress-testing`) 是一个 Go 语言实现的压测工具。
*   **特性**:
    *   **原生并发**: 充分利用 Goroutine，单机可模拟数万并发。
    *   **无依赖**: 编译后就是一个二进制文件，不需要安装 Java (JMeter) 或 Python (Locust)。
    *   **可视化报告**: 自动生成 HTML/Markdown 报告，包含详细的耗时分布图。
    *   **HTTP/2 支持**: 能够测试现代 Web 服务。

---

## 2. 基础用法

假设你已经编译好了 `stress.exe`。

### 2.1 简单压测 (GET)
测试排行榜接口。
```powershell
./stress.exe -c 50 -n 1000 -u "http://localhost:8888/v1/ranking/top?n=10"
```
*   `-c 50`: 并发数 (Concurrency)，即同时有 50 个用户在请求。
*   `-n 1000`: 请求总数 (Total Requests)，**注意：这里是单并发的请求数**。总请求数 = 50 * 1000 = 50,000。
*   `-u`: 目标 URL。

### 2.2 复杂压测 (POST + Body + Header)
测试点赞接口 (需要 Token)。
```powershell
./stress.exe -c 20 -n 100 -u "http://localhost:8888/v1/article/like" -data '{"article_id": 9999, "action": 1}' -H "Content-Type: application/json" -H "Authorization: Bearer <YOUR_TOKEN>"
```

---

## 3. 实战演练：压测 Ranking 服务 (读性能)

Ranking 服务是 **IO 密集型 (Redis)**，主要瓶颈在于 Redis 带宽和连接池。

### 步骤 3.1: 准备环境
确保 `Ranking Service` 和 `Gateway` 运行中。

### 步骤 3.2: 执行压测
我们模拟 **100 并发**，每个用户请求 **100 次** (共 10000 次请求)。
```powershell
./stress.exe -c 100 -n 100 -u "http://localhost:8888/v1/ranking/top?n=20" -o "ranking_report.html"
```
*   `-o`: 输出报告文件名。

### 3.3 观察控制台输出
你会看到类似：
```
开始启动并发: 100
请求总数: 10000
正在执行... [==============>                 ] 50%
```

### 3.4 观察 Grafana
在压测进行时，去 `http://localhost:3000` 查看 Ranking RPC 的 **Monitor Panel**。
*   **QPS**: 是否达到了 2000+ ?
*   **Latency**: P99 是否能保持在 10ms 以内？如果飙高，说明 Redis 连接池可能不够大了。

### 3.5 分析报告
打开生成的 `ranking_report.html`。
重点关注：
1.  **耗时分布图**: 看柱状图是否集中在左侧 (快)。如果右侧有长尾，说明有慢请求。
2.  **错误率**: 非 200 状态码的比例。

---

## 4. 实战演练：压测 Interaction 服务 (写性能)

Interaction 服务是 **异步写 (RabbitMQ)**，瓶颈在于 MQ 的写入速度和消费速度。

### 步骤 4.1: 获取 Token
确保你有一个有效的 Token。如果你忘了，可以用 `go run test_e2e.go` (如果还在) 或者 `curl` 获取一个。

### 步骤 4.2: 执行压测 (削峰填谷验证)
模拟 **50 并发**，持续疯狂点赞。
```powershell
./stress.exe -c 50 -n 200 -u "http://localhost:8888/v1/article/like" -data "{\"article_id\": 8888, \"action\": 1}" -H "Content-Type: application/json" -H "Authorization: Bearer <YOUR_TOKEN>" -o "like_report.html"
```

### 4.3 关键观察点 (面试核心)
这时候，你的 **Ranking Service (Consumer)** 可能处理不过来。
*   **Grafana**: 查看 Interaction RPC 的 QPS (例如 1000 qps)。
*   **RabbitMQ**: (如果有监控) 队列长度是否在堆积？
*   **结论**: 只要 Interaction RPC 返回快 (Latency 低)，说明 **削峰成功**。Redis 更新慢一点没关系 (最终一致性)。

---

## 5. 性能调优建议 (面试加分)

如果压测发现性能上不去，怎么调优？

1.  **调整 Go-Zero 配置**:
    *   增加 `MaxBytes` (RPC 包大小限制)。
    *   调整 `CpuThreshold` (降级阈值，默认 900)。
2.  **调整连接池**:
    *   Redis/DB 连接池大小 (MaxOpenConns, MaxIdleConns)。
3.  **资源扩容**:
    *   增加 Pod 数量 (Horizontal Pod Autoscaling)。

---

## 6. 下一步

现在你已经掌握了压测工具。
*   **作业**: 尝试把并发加到 500 (注意保护你的电脑 CPU)，看看系统什么时候会崩 (Error 率上升)。
*   **思考**: 如果 QPS 上不去，瓶颈是在 Gateway 还是 RPC？(看 Tracing 或日志)。
