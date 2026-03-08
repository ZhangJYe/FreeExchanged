# Phase 12: 工程化部署 (Deployment) - Part 1

> 本章的目标是将所有微服务容器化，让项目具备“云原生”交付能力。
> 这是区分“玩具项目”与“工业级项目”的关键分水岭。面试时可以展示 Docker 镜像构建流程和分层优化技巧。

---

# Part 1: Docker 容器化

## 1.1 基础镜像规划

为了减小镜像体积并与 go-zero 兼容，我们统一使用 `golang:1.22-alpine` 作为构建环境（Build Stage），使用 `alpine:latest` 作为运行环境（Run Stage）。

**多阶段构建 (Multi-stage Build)** 是必须掌握的技巧：
1.  **Build Stage**: 包含 Go 编译器，负责编译二进制文件。
2.  **Run Stage**: 仅包含最小 Runtime，体积极小（通常 < 20MB）。

---

## 1.2 编写 Gateway Dockerfile

**文件路径**：`app/gateway/Dockerfile`

```dockerfile
# Stage 1: Build
FROM golang:1.22-alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

# 1. 预下载依赖（利用 Docker 缓存层）
COPY go.mod go.sum ./
RUN go mod download

# 2. 复制源码并编译
COPY . .
# 这里的路径根据 go-zero 工程结构，需要指向 main 包
COPY app/gateway/etc/gateway.yaml /app/etc/gateway.yaml
RUN go build -ldflags="-s -w" -o /app/gateway app/gateway/gateway.go


# Stage 2: Run
FROM alpine:latest

# 必须安装 ca-certificates (如果不装，HTTPS 请求会报错 x509)
# 安装 tzdata (如果不装，日志时间会差8小时)
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update --no-cache && \
    apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai

WORKDIR /app

# 从 builder 阶段复制二进制文件和配置文件
COPY --from=builder /app/gateway .
COPY --from=builder /app/etc/gateway.yaml ./etc/gateway.yaml

EXPOSE 8888 9091

CMD ["./gateway", "-f", "etc/gateway.yaml"]
```

**面试题**: 为什么要设置 `CGO_ENABLED=0`？
> 答：为了静态编译（Statically Linked），让生成的 Go 二进制文件不依赖系统的 libc 库。如果不设这个，在 alpine 这种极简镜像里（用的是 musl libc 而非 glibc）运行可能会报错 "no such file or directory"。

---

## 1.3 编写 RPC 服务 Dockerfile (通用模板)

所有 RPC 服务（User, Interaction, Article, Ranking, Rate, Watchlist）结构高度一致。我们可以创建一个通用的 Dockerfile 模板，只需改变构建路径。但为了清晰，建议每个服务目录放一个。

以 **User RPC** 为例：`app/user/cmd/rpc/Dockerfile`

```dockerfile
FROM golang:1.22-alpine AS builder

LABEL stage=gobuilder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# 关键路径修改
COPY app/user/cmd/rpc/etc/user.yaml /app/etc/user.yaml
RUN go build -ldflags="-s -w" -o /app/user-rpc app/user/cmd/rpc/user.go

FROM alpine:latest

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories && \
    apk update --no-cache && \
    apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/user-rpc .
COPY --from=builder /app/etc/user.yaml ./etc/user.yaml

EXPOSE 8080 9090

CMD ["./user-rpc", "-f", "etc/user.yaml"]
```

**其他 RPC 服务的端口对应表（用于 EXPOSE）：**
- User: 8080 (RPC) / 9090 (Metric)
- Interaction: 8081 / 9092
- Ranking: 8082 / 9093
- Rate: 8083 / 9094
- Article: 8084 / 9096
- Watchlist: 8085 / 9097

---

## 1.4 编写 Rate Job Dockerfile (CronTask)

Rate 服务还有一个定时任务进程 (Job)。

**文件路径**：`app/rate/cmd/job/Dockerfile`

```dockerfile
FROM golang:1.22-alpine AS builder
LABEL stage=gobuilder
ENV CGO_ENABLED=0 GOOS=linux GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY app/rate/cmd/job/etc/job.yaml /app/etc/job.yaml
RUN go build -ldflags="-s -w" -o /app/rate-job app/rate/cmd/job/ratejob.go

FROM alpine:latest
RUN apk update --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /app/rate-job .
COPY --from=builder /app/etc/job.yaml ./etc/job.yaml
# Job 通常不需要 EXPOSE 端口，除非它开了 metrics
CMD ["./rate-job", "-f", "etc/job.yaml"]
```

---

## 1.5 编写 Makefile 自动化

手动输 `docker build` 命令太慢且易错。
更新根目录的 `Makefile`，增加 `build` 命令：

```makefile
# 定义服务列表
SERVICES := user interaction article ranking rate watchlist
GATEWAY := gateway

.PHONY: docker-base docker-rpc docker-api

# 构建所有镜像
docker-all: docker-base docker-rpc docker-api docker-job

# 构建 Gateway
docker-api:
	docker build -t freeexchanged/gateway:v1 -f app/gateway/Dockerfile .

# 批量构建所有 RPC
docker-rpc:
	for svc in $(SERVICES); do \
		docker build -t freeexchanged/$$svc-rpc:v1 -f app/$$svc/cmd/rpc/Dockerfile . ; \
	done

# 构建 Job
docker-job:
	docker build -t freeexchanged/rate-job:v1 -f app/rate/cmd/job/Dockerfile .
```

*注意：`docker build` 的 context (上下文) 必须要是项目根目录 `.`，不能进子目录 build，否则 `COPY go.mod` 会找不到文件。*

---

## 1.6 本地验证

构建完成后，用 `docker images` 查看镜像体积。
期望结果：所有微服务的镜像大小应该在 **15MB - 30MB** 之间（得益于 alpine + 静态编译 + ldflags -s -w）。这也经常是面试官会随口问的一个数据。

---

下一节 → **Part 2: Kubernetes YAML 编写与部署**

---

# Part 2: Kubernetes 编排与部署

## 2.1 整体架构设计

我们将为每个服务创建以下 K8s 资源：
1.  **ConfigMap**: 存储业务配置（如 `user.yaml`），挂载到 `/app/etc`。
2.  **Deployment**: 管理无状态的业务 Pod。
3.  **Service**: 提供以服务名访问的稳定 DNS（ClusterIP），如 `user-rpc.default.svc.cluster.local`。
4.  **HPA**: 基于 CPU 利用率自动扩缩容（仅对核心服务）。

---

## 2.2 创建 ConfigMap (配置中心化)

我们将所有 RPC 的配置文件统一管理。
创建一个文件 `deploy/k8s/configmaps.yaml`：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: user-rpc-conf
  namespace: default
data:
  user.yaml: |
    Name: user.rpc
    ListenOn: 0.0.0.0:8080
    # K8s 中的 MySQL 服务名（通常是 externalName 或 headless service）
    # 这里假设我们已经在 K8s 外部或通过 Service 部署了 MySQL
    DataSource: root:root@tcp(mysql-service:3306)/freeexchanged?charset=utf8mb4&parseTime=true&loc=Asia%2FShanghai
    BizRedis:
      Host: redis-service:6379
      Type: node
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: gateway-conf
namespace: default
data:
  gateway.yaml: |
    Name: gateway
    Host: 0.0.0.0
    Port: 8888
    # RPC 服务发现：在 K8s 中直接用 Service DNS
    UserRpc:
      Endpoints:
        - user-rpc-service:8080
      NonBlock: true
    # ... 其他 RPC 配置同理 ...
```

**教学点**：
- `data` 里的 key 是文件名，value 是文件内容。
- 在 Pod 里挂载这个 ConfigMap 到 `/app/etc` 目录，程序读取 `/app/etc/user.yaml` 时就能读到这里的配置。
- **配置热更新**：修改 ConfigMap 后，Pod 重启即可生效，无需重新打镜像。

---

## 2.3 编写 Deployment (核心负载)

以 **User RPC** 为例，创建 `deploy/k8s/user-rpc-deployment.yaml`：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: user-rpc
  namespace: default
  labels:
    app: user-rpc
spec:
  replicas: 2  # 初始副本数：2（高可用最低要求）
  selector:
    matchLabels:
      app: user-rpc
  template:
    metadata:
      labels:
        app: user-rpc
    spec:
      containers:
      - name: user-rpc
        image: freeexchanged/user-rpc:v1 # 你的镜像名
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 8080 # RPC 端口
        - containerPort: 9090 # Metrics 端口
        
        # 资源限制（非常重要！防止 OOM）
        resources:
          requests:
            cpu: 100m     # 0.1 Core
            memory: 128Mi
          limits:
            cpu: 500m     # 0.5 Core
            memory: 256Mi
            
        # 挂载配置文件
        volumeMounts:
        - name: config-volume
          mountPath: /app/etc
          
        # 存活探针（Liveness Probe）
        livenessProbe:
          tcpSocket:
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
          
      volumes:
      - name: config-volume
        configMap:
          name: user-rpc-conf # 引用 2.2 创建的 CM
```

**教学点**：
- **`replicas: 2`**: 保证挂了一个还有一个，实现高可用。
- **`resources`**: 如果不加 limit，单个 Pod 可能吃光宿主机内存（OOM）。面试必问！
- **`livenessProbe`**: K8s 会定期 ping 8080 端口，如果不通，自动重启 Pod（自愈能力）。

---

## 2.4 Service (服务发现)

K8s 内部通信不通过 IP（因为 Pod IP 会变），而是通过 Service DNS。

创建 `deploy/k8s/user-rpc-service.yaml`：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: user-rpc-service # 这个名字就是 DNS 域名
  namespace: default
spec:
  selector:
    app: user-rpc # 匹配 Deployment 的 label
  ports:
  - name: rpc
    port: 8080        # Service 暴露的端口
    targetPort: 8080  # Pod 内部的端口
  - name: metrics
    port: 9090
    targetPort: 9090
  type: ClusterIP # 仅集群内部可访问（默认）
```

**效果**：Gateway 配置里的 `user-rpc-service:8080` 就能解析到这个 Service，然后负载均衡到 2 个 Pod。

---

## 2.5 HPA (水平自动伸缩)

这是展示“抗高并发”能力的杀手锏。

创建 `deploy/k8s/user-rpc-hpa.yaml`：

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: user-rpc-hpa
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: user-rpc
  minReplicas: 2
  maxReplicas: 10  # 流量大时最多扩容到 10 个
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 60 # CPU 超过 60% 就扩容
```

**原理**：K8s Metric Server 每 15s 采集一次 metrics。如果 `user-rpc` 的 CPU 平均利用率超过 60%，HPA 会自动增加 `replicas`，直到利用率降下去或达到 `maxReplicas`。流量下去后自动缩容。

---

## 2.6 Gateway 与 Ingress

Gateway 是对外入口，需要配置 **Ingress** 来暴露 HTTP 服务。

`deploy/k8s/gateway-ingress.yaml`:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: gateway-ingress
  namespace: default
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
spec:
  ingressClassName: nginx
  rules:
  - host: api.freeexchanged.com # 你的域名
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: gateway-service
            port:
              number: 8888
```

---

## 2.7 一键部署脚本

编写 `deploy/deploy.sh`：

```bash
#!/bin/bash

# 1. 创建 ConfigMaps
kubectl apply -f deploy/k8s/configmaps.yaml

# 2. 部署 MySQL/Redis (如果你用 Helm，跳过这一步)
kubectl apply -f deploy/k8s/infrastructure.yaml

# 3. 部署所有 RPC 服务
kubectl apply -f deploy/k8s/user-rpc-deployment.yaml
kubectl apply -f deploy/k8s/user-rpc-service.yaml
# ... 重复其他服务 ...

# 4. 部署 Gateway
kubectl apply -f deploy/k8s/gateway-deployment.yaml
kubectl apply -f deploy/k8s/gateway-service.yaml

# 5. 部署 Ingress
kubectl apply -f deploy/k8s/gateway-ingress.yaml

echo "Deployment completed! Check status with: kubectl get pods"
```

---

> ✅ **Phase 12 完成**：你现在掌握了从 Docker 镜像构建到 Kubernetes 高可用编排的全流程。
> **面试杀手锏**：被问到“如何保证服务高可用”时，直接甩出 HPA + Liveness Probe + ReplicaSet 的组合拳。

