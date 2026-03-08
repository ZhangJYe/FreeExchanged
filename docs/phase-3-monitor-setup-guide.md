# Phase 3: 基础监控搭建指南

本指南专注于搭建 Prometheus + Grafana 监控体系，涵盖了让微服务“吐出” Metrics 数据并被抓取的全过程。

## 1. 监控架构与原理

### 1.1 核心组件
*   **Exporter**: 你的微服务 (Go-Zero)。哪怕你不写代码，它也能吐出 `/metrics`。
*   **Prometheus**: 定期抓取 Exporter 的数据并存储。
*   **Grafana**: 展示数据的 Dashboard。

### 1.2 数据流向
`Service (:909x)` <--- Pull --- `Prometheus (:9090)` --- Query ---> `Grafana (:3000)`

---

## 2. 服务端配置

我们需要告知每个服务开启 Prometheus 端口。

### 步骤 2.1: Gateway (`app/gateway/etc/gateway.yaml`)
```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9091
  Path: /metrics
```

### 步骤 2.2: User Service (`app/user/cmd/rpc/etc/user.yaml`)
```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9092
  Path: /metrics
```

### 步骤 2.3: Interaction Service (`app/interaction/cmd/rpc/etc/interaction.yaml`)
```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9093
  Path: /metrics
```

### 步骤 2.4: Ranking Service (`app/ranking/cmd/rpc/etc/ranking.yaml`)
```yaml
Prometheus:
  Host: 0.0.0.0
  Port: 9094
  Path: /metrics
```

**注意**: 修改配置后，必须**重启所有 Go 服务**才能生效！

---

## 3. Prometheus 配置 (`deploy/prometheus/prometheus.yml`)

创建或修改此文件，告诉 Prometheus 去哪里抓数据。
由于 Prometheus 运行在 Docker 容器中，我们要用 `host.docker.internal` 访问宿主机上的服务。

```yaml
global:
  scrape_interval: 10s

scrape_configs:
  - job_name: 'gateway'
    static_configs:
      - targets: ['host.docker.internal:9091']

  - job_name: 'user-rpc'
    static_configs:
      - targets: ['host.docker.internal:9092']

  - job_name: 'interaction-rpc'
    static_configs:
      - targets: ['host.docker.internal:9093']

  - job_name: 'ranking-rpc'
    static_configs:
      - targets: ['host.docker.internal:9094']
```

---

## 4. 启动监控设施

确保 `docker-compose.yml` 包含 prometheus 和 grafana 服务。

执行启动命令：
```bash
docker-compose up -d prometheus grafana
```
如果修改了 `prometheus.yml`，可能需要重启容器：
```bash
docker restart freeexchanged-prometheus-1
```

---

## 5. 验证

1.  访问 `http://localhost:9090/targets`。
2.  看到所有 Target 状态为 **UP (绿色)** 即为成功。

完成基础搭建后，请参考 `phase-3-monitor-expert-guide.md` 进行进阶实战。
