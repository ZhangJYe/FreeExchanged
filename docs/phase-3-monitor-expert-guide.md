# Phase 3: 监控进阶 - 可视化压测与面试指南

本指南是 `phase-3-monitor-setup-guide.md` 的补充。当你完成了基础监控配置后，请跟随本指南进行实战演练。

## 1. Grafana 可视化实战 (手把手)

Prometheus 只是数据库，Grafana 才是这一让老板和面试官眼花缭乱的“面子工程”。

### 步骤 1.1: 配置数据源
1.  打开浏览器访问 `http://localhost:3000` (账号 admin / 密码 admin)。
2.  点击左侧齿轮图标 (Configuration) -> **Data Sources**。
3.  点击 **Add data source**。
4.  选择 **Prometheus**。
5.  **关键设置**:
    *   **HTTP URL**: `http://prometheus:9090` (注意：这里必须用容器名 `prometheus`，不能用 localhost，因为 Grafana 在容器里)。
6.  点击底部 **Save & Test**。如果显示绿色 "Data source is working"，通过。

### 步骤 1.2: 导入 Go-Zero 官方仪表盘
你不需要自己一个个画图，Go-Zero 社区已经做好了完美的模板。

1.  鼠标移到左侧加号 (+) -> **Import**。
2.  在 **Import via grafana.com** 下方输入 ID: **15160** (这是专门适配 Go-Zero 的 Dashboard)。
3.  点击 **Load**。
4.  在底部的 **Prometheus** 下拉框中，选择你刚才创建的数据源 (Prometheus)。
5.  点击 **Import**。

### 步骤 1.3: 仪表盘初体验
你现在应该能看到一个非常专业的面板，包含：
*   **RPC QPS**: 每秒请求量。
*   **RPC Server / Client Duration**: 请求耗时（P99, P95）。
*   **Memory / CPU**: 各个服务的资源占用。

右上角的 **Job** 下拉框，你可以选择 `gateway`、`user-rpc` 等来查看不同服务的详情。

---

## 2. 压测实战：让曲线动起来

监控系统最怕“死水一潭”。我们需要制造一点“人为事故”（流量洪峰），来看看监控的反应。

我们将使用项目根目录下的 `stress_test.go` 脚本。

### 2.1 执行压测
确保所有服务正在运行，然后在终端执行：
```bash
go run stress_test.go
```

这个脚本会模拟 20 个并发用户，持续 30 秒疯狂请求 Ranking 接口。

### 2.2 观察现象 (面试考点)
运行脚本的同时，盯着 **Grafana** (设置 Auto refresh 为 5s)。

1.  **QPS 飙升**: 你会看到 Gateway 和 Ranking RPC 的 QPS 曲线瞬间拉升。
2.  **Latency 变化**: 随着并发增加，P99 延迟可能会轻微抖动。如果耗时突然变长，说明系统到了瓶颈（可能是 CPU，也可能是 Redis 连接池）。
3.  **内存变化**: Go 的内存分配会随着请求量增加，但 GC 会定期回收，你会看到锯齿状的内存曲线。

---

## 3. 深度解读：面试如何“看图说话”

**面试官**: 你怎么通过监控发现性能瓶颈的？

**你的回答 (满分模板)**:

"在压测 Ranking 服务时，我关注了三个核心指标：**QPS、P99 Latency 和 Error Rate**。

1.  **P99 vs Average**: 我不看平均耗时，因为它是骗人的。我看 **P99** (99%的请求都在这个时间内完成)。例如，平均耗时 5ms，但 P99 飙到了 200ms，说明有 1% 的长尾请求被阻塞了，这通常是因为 **Redis 连接池耗尽** 或者 **Go 协程调度延迟**。

2.  **CPU / Memory**: 我发现当 QPS 达到 2000 时，Gateway 的 CPU 还没满，但 Latency 升高了。结合 **Histogram** 图表，我发现这些慢请求主要卡在 `RPC Client Duration`，也就是 Gateway 等待 Ranking RPC 返回的时间。

3.  **最终定位**: 进一步查看 Ranking RPC 的面板，发现 Redis 操作耗时增加。这推导出可能是热点 Key 问题（Hot Key），于是引入了本地缓存（Map）来解决。"

---

## 4. 常见问题排查 (Troubleshooting)

*   **Grafana 显示 "No Data"**:
    1.  Prometheus Targets 里对应的 Job 是 UP 吗？
    2.  Prometheus 当前时间范围 (Time Range) 对吗？改为 "Last 5 minutes"。
    3.  服务确实在运行且有流量吗？(跑一下 `check_qps.go` 或 `stress_test.go`)。

*   **监控数据由断点**:
    *   可能是 Prometheus 抓取超时。检查 `scrape_interval`。docker 资源不足也会导致抓取失败。
