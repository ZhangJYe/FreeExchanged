# Phase 2: 互动与排行榜系统测试指南

本指南将指导你完成从用户点赞到排行榜实时更新的全链路测试。

## 1. 环境准备

确保以下服务全部启动（并且无报错）：
1.  **User RPC**: `go run app/user/cmd/rpc/user.go -f app/user/cmd/rpc/etc/user.yaml` (端口 8080)
2.  **Interaction RPC**: `go run app/interaction/cmd/rpc/interaction.go -f app/interaction/cmd/rpc/etc/interaction.yaml` (端口 8081)
3.  **Ranking RPC**: `go run app/ranking/cmd/rpc/ranking.go -f app/ranking/cmd/rpc/etc/ranking.yaml` (端口 8082)
4.  **API Gateway**: `go run app/gateway/gateway.go -f app/gateway/etc/gateway.yaml` (端口 8888)

**注意**: 如果你修改了 Gateway 代码但还没重启，请务必重启 Gateway。

---

## 2.获取用户 Token

我们需要一个 Token 来调用受保护的点赞接口。

### 步骤 2.1: 注册用户 (如果是第一次测试)
```bash
curl -X POST http://localhost:8888/v1/user/register \
-H "Content-Type: application/json" \
-d '{
    "mobile": "13800138000",
    "password": "password123",
    "nickname": "TestUser1"
}'
```
**预期结果**: 返回 `token`。

### 步骤 2.2: 登录用户 (获取 Token)
```bash
curl -X POST http://localhost:8888/v1/user/login \
-H "Content-Type: application/json" \
-d '{
    "mobile": "13800138000",
    "password": "password123"
}'
```
**关键动作**: 复制返回的 `token` 字段的值。后续我们记作 `<YOUR_TOKEN>`。

---

## 3. 测试点赞 (Interaction -> MQ -> Ranking)

### 步骤 3.1: 发送点赞请求
我们将对文章 ID `1001` 进行点赞。

```bash
curl -X POST http://localhost:8888/v1/article/like \
-H "Content-Type: application/json" \
-H "Authorization: Bearer <YOUR_TOKEN>" \
-d '{
    "article_id": 1001,
    "action": 1
}'
```
**预期结果**: HTTP 200 OK (无返回体或 `{}`)。

### 步骤 3.2: 观察 Interaction Service 日志
查看 Interaction RPC 的控制台输出，应该能看到类似：
```
INFO: Like event published: uid=1, aid=1001
```
这说明消息已成功发送到 RabbitMQ。

### 步骤 3.3: 观察 Ranking Service 日志 (消费者)
查看 Ranking RPC 的控制台输出，应该能看到类似：
```
INFO: Received like event: uid=1, aid=1001
INFO: Updated rank for article 1001
```
这说明消费者成功监听到了消息，并更新了 Redis。

---

## 4. 测试排行榜 (Ranking -> Redis)

### 步骤 4.1: 获取 Top N 文章
查询前 10 名热榜文章。

```bash
curl "http://localhost:8888/v1/ranking/top?n=10"
```
注意：这是一个 GET 请求，且不需要 Token (公开接口)。

**预期结果**:
```json
{
    "items": [
        {
            "article_id": 1001,
            "score": 1,
            "title": "Article 1001"
        }
    ]
}
```
你应该能看到 `article_id: 1001` 的 `score` 变成了 `1`（或者更高，如果你点了多次）。

---

## 5. 压力/并发测试 (可选)

你可以尝试对同一篇文章多次点赞（虽然目前没有去重），或者对不同的文章点赞。
```bash
# 文章 1002 点赞
curl -X POST http://localhost:8888/v1/article/like ... -d '{"article_id": 1002}'

# 文章 1001 再次点赞
curl -X POST http://localhost:8888/v1/article/like ... -d '{"article_id": 1001}'
```

再次查询 Top 10，应该看到：
1. Article 1001 (Score 2)
2. Article 1002 (Score 1)

---

## 6. 排错指南

*   **RabbitMQ 连接失败**: 检查 `config.yaml` 里的 RabbitMQ 端口和账号密码。Docker 是否启动？
*   **Redis 连接失败**: 检查 Redis 端口。
*   **Gateway 报错 `rpc error: code = Unavailable`**: 对应的 RPC 服务没启动，或者端口配置对不上。
*   **401 Unauthorized**: Token 过期或错误。
*   **Consul 报错**: 确保 Consul 已启动 (localhost:8500)。
