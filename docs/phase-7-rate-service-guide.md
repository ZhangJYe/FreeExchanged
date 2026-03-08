# Phase 7: 汇率服务实战指南

> "FreeExchanged 的核心是汇率。没有汇率服务，项目名字就是空的。"

本阶段我们将实现完整的汇率服务，包括：
- **Rate RPC**: 提供汇率查询接口
- **Rate Job**: 定时从外部 API 拉取汇率并缓存到 Redis
- **Gateway 集成**: 暴露 HTTP 接口

---

# Part 1: 架构与设计

## 1.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│  外部汇率 API (open.er-api.com)                          │
│  免费，无需注册，每小时更新一次                            │
└──────────────────┬──────────────────────────────────────┘
                   │ 每分钟拉取一次 (HTTP GET)
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Rate Job (定时任务)                                      │
│  - 解析汇率数据                                           │
│  - 写入 Redis Hash                                       │
└──────────────────┬──────────────────────────────────────┘
                   │ HSET rate:USD CNY 7.25 EUR 0.92 ...
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Redis                                                   │
│  rate:USD → {CNY: 7.25, EUR: 0.92, GBP: 0.79, ...}    │
│  rate:EUR → {USD: 1.08, CNY: 7.84, GBP: 0.86, ...}    │
└──────────────────┬──────────────────────────────────────┘
                   │ HGET rate:USD CNY
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Rate RPC (gRPC 服务)                                    │
│  - GetRate(from, to) → 从 Redis 读取                    │
│  - GetSupportedCurrencies() → 返回支持的货币列表          │
└──────────────────┬──────────────────────────────────────┘
                   │ gRPC 调用
                   ▼
┌─────────────────────────────────────────────────────────┐
│  Gateway (REST API)                                      │
│  GET /v1/rate?from=USD&to=CNY → {"rate": 7.25}         │
│  GET /v1/rate/currencies → {"currencies": [...]}        │
└─────────────────────────────────────────────────────────┘
```

---

## 1.2 设计模式：Cache Aside Pattern (面试必背)

这是缓存领域最经典的设计模式，我们的汇率服务完美体现了它：

```
┌─────────────────────────────────────────────────────────┐
│  Cache Aside Pattern                                     │
│                                                          │
│  写路径 (Job 负责):                                       │
│  外部 API → 解析数据 → 写入 Redis                         │
│                                                          │
│  读路径 (RPC 负责):                                       │
│  客户端请求 → 查 Redis → 命中则返回                        │
│                        → 未命中则返回错误（数据未就绪）     │
└─────────────────────────────────────────────────────────┘
```

**为什么不用 Cache Through（读时更新）？**
- 汇率数据是**时效性数据**，不需要实时精确，允许最多 1 分钟的延迟。
- Job 定时更新比请求时更新更稳定，不会因为并发请求导致多次调用外部 API（缓存击穿问题）。

---

## 1.3 Redis 数据结构设计

使用 **Hash** 存储汇率数据：

```
Key:   rate:{from_currency}
Field: {to_currency}
Value: {rate}

示例:
rate:USD → {
  CNY: "7.2534",
  EUR: "0.9234",
  GBP: "0.7891",
  JPY: "149.23",
  HKD: "7.8234",
  ...
}
```

**为什么用 Hash 而不是 String？**
- 如果用 String，每个货币对需要一个 Key（`rate:USD:CNY`、`rate:USD:EUR`...），Key 数量爆炸。
- 用 Hash，每个基准货币只需一个 Key，所有目标货币作为 Field，结构清晰，内存高效。

**TTL 设置**：
```
EXPIRE rate:USD 3600  # 1小时过期
```
Job 每分钟更新一次，TTL 设 1 小时，即使 Job 挂了，数据也能撑 1 小时。

---

## 1.4 外部 API 选型

使用 **ExchangeRate-API** 的免费端点：
```
GET https://open.er-api.com/v6/latest/{base_currency}
```

**响应示例**：
```json
{
  "result": "success",
  "base_code": "USD",
  "rates": {
    "CNY": 7.2534,
    "EUR": 0.9234,
    "GBP": 0.7891,
    "JPY": 149.23,
    "HKD": 7.8234
  }
}
```

**优点**：
- 完全免费，无需注册 API Key
- 每小时更新一次（我们每分钟拉，数据会重复，但保证新鲜度）
- 支持 170+ 种货币

---

## 1.5 Proto 接口设计

```protobuf
syntax = "proto3";
package rate;

// 获取汇率
message GetRateReq {
  string from = 1;  // 源货币，如 "USD"
  string to   = 2;  // 目标货币，如 "CNY"
}
message GetRateResp {
  string from = 1;
  string to   = 2;
  double rate = 3;  // 汇率，如 7.2534
  int64  updated_at = 4;  // 最后更新时间戳
}

// 获取支持的货币列表
message GetSupportedCurrenciesReq {}
message GetSupportedCurrenciesResp {
  repeated string currencies = 1;  // ["USD", "CNY", "EUR", ...]
}

service Rate {
  rpc GetRate(GetRateReq) returns(GetRateResp);
  rpc GetSupportedCurrencies(GetSupportedCurrenciesReq) returns(GetSupportedCurrenciesResp);
}
```

---

## 1.6 目录结构规划

```
app/rate/
├── cmd/
│   ├── rpc/                    # RPC 服务（查询汇率）
│   │   ├── etc/rate.yaml
│   │   ├── internal/
│   │   │   ├── config/config.go
│   │   │   ├── logic/
│   │   │   │   ├── getratelogic.go          # 核心：从 Redis 读汇率
│   │   │   │   └── getsupportedcurrencieslogic.go
│   │   │   ├── server/rateserver.go
│   │   │   └── svc/servicecontext.go
│   │   └── rate.go             # 入口
│   └── job/                    # 定时任务（拉取并缓存汇率）
│       ├── etc/job.yaml
│       ├── internal/
│       │   ├── config/config.go
│       │   ├── fetcher/        # 外部 API 拉取逻辑
│       │   │   └── fetcher.go
│       │   └── svc/servicecontext.go
│       └── job.go              # 入口
└── desc/rate.proto
```

---

**Part 1 小结**：
- 架构：Job 写缓存，RPC 读缓存，Gateway 暴露 HTTP。
- 模式：Cache Aside Pattern，写读分离。
- 存储：Redis Hash，Key = `rate:{from}`，Field = `{to}`，Value = 汇率值。
- 外部 API：`open.er-api.com`，免费无需注册。

下一节 → **Part 2: 代码实现**

---

# Part 2: 代码实现

## 2.1 Step 1: 重写 Proto 文件

**文件**: `desc/rate.proto`

```protobuf
syntax = "proto3";

package rate;

option go_package = "./rate";

// --- 获取汇率 ---
message GetRateReq {
  string from = 1;  // 源货币，如 "USD"
  string to   = 2;  // 目标货币，如 "CNY"
}
message GetRateResp {
  string from       = 1;
  string to         = 2;
  double rate       = 3;  // 汇率值，如 7.2534
  int64  updated_at = 4;  // 最后更新时间戳 (Unix)
}

// --- 获取支持的货币列表 ---
message GetSupportedCurrenciesReq {}
message GetSupportedCurrenciesResp {
  repeated string currencies = 1;
}

service Rate {
  rpc GetRate(GetRateReq) returns(GetRateResp);
  rpc GetSupportedCurrencies(GetSupportedCurrenciesReq) returns(GetSupportedCurrenciesResp);
}
```

**重新生成代码**:
```powershell
# 在项目根目录执行
goctl rpc protoc desc/rate.proto --go_out=app/rate/cmd/rpc --go-grpc_out=app/rate/cmd/rpc --zrpc_out=app/rate/cmd/rpc --style=goZero
```

---

## 2.2 Step 2: 修改 YAML 配置

**文件**: `app/rate/cmd/rpc/etc/rate.yaml`

```yaml
Name: rate.rpc
ListenOn: 0.0.0.0:8083

Prometheus:
  Host: 0.0.0.0
  Port: 9095
  Path: /metrics

Telemetry:
  Name: rate-rpc
  Endpoint: localhost:4318
  Sampler: 1.0
  Batcher: otlphttp
  OtlpHttpPath: /v1/traces

# Consul 服务注册
Consul:
  Host: 127.0.0.1:8500
  Key: rate.rpc

# Redis 配置（读取汇率缓存）
BizRedis:
  Host: 127.0.0.1:6379
  Type: node
```

---

## 2.3 Step 3: 修改 Config 结构

**文件**: `app/rate/cmd/rpc/internal/config/config.go`

```go
package config

import (
    "github.com/zeromicro/go-zero/core/stores/redis"
    "github.com/zeromicro/go-zero/zrpc"
    consul "github.com/zeromicro/zero-contrib/zrpc/registry/consul"
)

type Config struct {
    zrpc.RpcServerConf
    Consul   consul.Conf
    BizRedis redis.RedisConf
}
```

---

## 2.4 Step 4: 修改 ServiceContext

**文件**: `app/rate/cmd/rpc/internal/svc/servicecontext.go`

```go
package svc

import (
    "freeexchanged/app/rate/cmd/rpc/internal/config"

    "github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
    Config config.Config
    Redis  *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        Config: c,
        Redis:  redis.MustNewRedis(c.BizRedis),
    }
}
```

---

## 2.5 Step 5: 实现 GetRate Logic（核心）

**文件**: `app/rate/cmd/rpc/internal/logic/getratelogic.go`

```go
package logic

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "time"

    "freeexchanged/app/rate/cmd/rpc/internal/svc"
    "freeexchanged/app/rate/cmd/rpc/rate"

    "github.com/zeromicro/go-zero/core/logx"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// Redis Key 格式: rate:{FROM_CURRENCY}
// 例如: rate:USD → Hash{CNY: "7.2534", EUR: "0.9234", ...}
const (
    rateKeyPrefix  = "rate:"
    updatedAtField = "_updated_at" // 特殊 field，存储最后更新时间
)

type GetRateLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func NewGetRateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRateLogic {
    return &GetRateLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

func (l *GetRateLogic) GetRate(in *rate.GetRateReq) (*rate.GetRateResp, error) {
    // 参数校验：转大写，防止大小写问题
    from := strings.ToUpper(strings.TrimSpace(in.From))
    to := strings.ToUpper(strings.TrimSpace(in.To))

    if from == "" || to == "" {
        return nil, status.Error(codes.InvalidArgument, "from and to are required")
    }

    // 相同货币直接返回 1
    if from == to {
        return &rate.GetRateResp{
            From:      from,
            To:        to,
            Rate:      1.0,
            UpdatedAt: time.Now().Unix(),
        }, nil
    }

    // 从 Redis Hash 读取汇率
    // HGET rate:USD CNY
    redisKey := fmt.Sprintf("%s%s", rateKeyPrefix, from)
    rateStr, err := l.svcCtx.Redis.Hget(redisKey, to)
    if err != nil {
        l.Errorf("redis hget failed, key=%s field=%s err=%v", redisKey, to, err)
        return nil, status.Error(codes.Internal, "failed to get rate from cache")
    }
    if rateStr == "" {
        // 缓存未命中：数据未就绪（Job 还没跑）或不支持的货币对
        return nil, status.Errorf(codes.NotFound,
            "rate not found for %s->%s, data may not be ready yet", from, to)
    }

    // 解析汇率值
    rateVal, err := strconv.ParseFloat(rateStr, 64)
    if err != nil {
        return nil, status.Error(codes.Internal, "invalid rate data in cache")
    }

    // 读取最后更新时间
    updatedAtStr, _ := l.svcCtx.Redis.Hget(redisKey, updatedAtField)
    updatedAt, _ := strconv.ParseInt(updatedAtStr, 10, 64)

    return &rate.GetRateResp{
        From:      from,
        To:        to,
        Rate:      rateVal,
        UpdatedAt: updatedAt,
    }, nil
}
```

---

## 2.6 Step 6: 实现 GetSupportedCurrencies Logic

**文件**: `app/rate/cmd/rpc/internal/logic/getsupportedcurrencieslogic.go`

```go
package logic

import (
    "context"

    "freeexchanged/app/rate/cmd/rpc/internal/svc"
    "freeexchanged/app/rate/cmd/rpc/rate"

    "github.com/zeromicro/go-zero/core/logx"
)

// 支持的货币列表（与 Job 拉取的保持一致）
var supportedCurrencies = []string{
    "USD", "CNY", "EUR", "GBP", "JPY",
    "HKD", "KRW", "SGD", "AUD", "CAD",
}

type GetSupportedCurrenciesLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
    logx.Logger
}

func NewGetSupportedCurrenciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSupportedCurrenciesLogic {
    return &GetSupportedCurrenciesLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
        Logger: logx.WithContext(ctx),
    }
}

func (l *GetSupportedCurrenciesLogic) GetSupportedCurrencies(
    in *rate.GetSupportedCurrenciesReq) (*rate.GetSupportedCurrenciesResp, error) {
    return &rate.GetSupportedCurrenciesResp{
        Currencies: supportedCurrencies,
    }, nil
}
```

---

## 2.7 Step 7: 实现 Job（定时拉取汇率）

这是整个汇率服务最核心的部分。

**文件**: `app/rate/cmd/job/internal/fetcher/fetcher.go`

```go
package fetcher

import (
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "strconv"
    "time"

    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/go-zero/core/stores/redis"
)

const (
    // 免费汇率 API，无需注册
    apiURL        = "https://open.er-api.com/v6/latest/%s"
    rateKeyPrefix = "rate:"
    updatedAtField = "_updated_at"
    rateKeyTTL    = 3600 // 1小时过期，Job 挂了数据还能撑 1 小时
)

// 支持的基准货币（每种都要拉一次 API）
var baseCurrencies = []string{
    "USD", "CNY", "EUR", "GBP", "JPY",
    "HKD", "KRW", "SGD", "AUD", "CAD",
}

// API 响应结构
type apiResponse struct {
    Result   string             `json:"result"`
    BaseCode string             `json:"base_code"`
    Rates    map[string]float64 `json:"rates"`
}

type Fetcher struct {
    redis  *redis.Redis
    client *http.Client
}

func NewFetcher(r *redis.Redis) *Fetcher {
    return &Fetcher{
        redis:  r,
        client: &http.Client{Timeout: 10 * time.Second},
    }
}

// FetchAll 拉取所有基准货币的汇率并写入 Redis
func (f *Fetcher) FetchAll() {
    for _, base := range baseCurrencies {
        if err := f.fetchAndCache(base); err != nil {
            logx.Errorf("[RateFetcher] failed to fetch %s: %v", base, err)
            // 单个货币失败不影响其他货币，继续执行
            continue
        }
        logx.Infof("[RateFetcher] successfully updated rates for %s", base)
    }
}

// fetchAndCache 拉取单个基准货币的汇率并写入 Redis
func (f *Fetcher) fetchAndCache(base string) error {
    // 1. 调用外部 API
    url := fmt.Sprintf(apiURL, base)
    resp, err := f.client.Get(url)
    if err != nil {
        return fmt.Errorf("http get failed: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return fmt.Errorf("read body failed: %w", err)
    }

    // 2. 解析响应
    var apiResp apiResponse
    if err := json.Unmarshal(body, &apiResp); err != nil {
        return fmt.Errorf("json unmarshal failed: %w", err)
    }
    if apiResp.Result != "success" {
        return fmt.Errorf("api returned non-success: %s", apiResp.Result)
    }

    // 3. 写入 Redis Hash
    // HSET rate:USD CNY 7.2534 EUR 0.9234 ...
    redisKey := rateKeyPrefix + base
    now := strconv.FormatInt(time.Now().Unix(), 10)

    // 构建 field-value 对
    pairs := make([]string, 0, len(apiResp.Rates)*2+2)
    for currency, rate := range apiResp.Rates {
        pairs = append(pairs, currency, strconv.FormatFloat(rate, 'f', 6, 64))
    }
    // 额外存储最后更新时间
    pairs = append(pairs, updatedAtField, now)

    if err := f.redis.Hmset(redisKey, pairsToMap(pairs)); err != nil {
        return fmt.Errorf("redis hmset failed: %w", err)
    }

    // 4. 设置 TTL（1小时）
    if err := f.redis.Expire(redisKey, rateKeyTTL); err != nil {
        logx.Errorf("[RateFetcher] set ttl failed for %s: %v", redisKey, err)
        // TTL 设置失败不是致命错误，继续
    }

    return nil
}

// pairsToMap 把 [k1, v1, k2, v2, ...] 转成 map
func pairsToMap(pairs []string) map[string]string {
    m := make(map[string]string, len(pairs)/2)
    for i := 0; i+1 < len(pairs); i += 2 {
        m[pairs[i]] = pairs[i+1]
    }
    return m
}
```

**文件**: `app/rate/cmd/job/job.go`（入口）

```go
package main

import (
    "flag"
    "time"

    "freeexchanged/app/rate/cmd/job/internal/config"
    "freeexchanged/app/rate/cmd/job/internal/fetcher"
    "freeexchanged/app/rate/cmd/job/internal/svc"

    "github.com/zeromicro/go-zero/core/conf"
    "github.com/zeromicro/go-zero/core/logx"
    "github.com/zeromicro/go-zero/core/stores/redis"
)

var configFile = flag.String("f", "etc/job.yaml", "the config file")

func main() {
    flag.Parse()

    var c config.Config
    conf.MustLoad(*configFile, &c)

    _ = svc.NewServiceContext(c) // 初始化（如有需要）

    r := redis.MustNewRedis(c.BizRedis)
    f := fetcher.NewFetcher(r)

    logx.Info("Rate Job started, fetching rates every 1 minute...")

    // 启动时立刻执行一次
    f.FetchAll()

    // 每分钟执行一次
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        f.FetchAll()
    }
}
```

**文件**: `app/rate/cmd/job/etc/job.yaml`

```yaml
BizRedis:
  Host: 127.0.0.1:6379
  Type: node
```

---

**Part 2 小结**：
- Proto 定义了 `GetRate` 和 `GetSupportedCurrencies` 两个接口。
- RPC Logic 从 Redis Hash 读取汇率，支持大小写不敏感、相同货币返回 1.0。
- Job 每分钟拉取 10 种基准货币的汇率，写入 Redis Hash，设置 1 小时 TTL。
- 单个货币拉取失败不影响其他货币（容错设计）。

下一节 → **Part 3: Gateway 集成 + 面试话术**

---

# Part 3: Gateway 集成

## 3.1 Step 1: 更新 gateway.api

在 `desc/gateway.api` 末尾追加汇率模块的类型和路由：

```go
// 在 type() 块里追加：
type (
    // --- Rate Module Types ---
    GetRateReq {
        From string `form:"from"`  // 源货币，如 USD
        To   string `form:"to"`   // 目标货币，如 CNY
    }
    GetRateResp {
        From      string  `json:"from"`
        To        string  `json:"to"`
        Rate      float64 `json:"rate"`
        UpdatedAt int64   `json:"updated_at"`
    }
    GetCurrenciesReq  {}
    GetCurrenciesResp {
        Currencies []string `json:"currencies"`
    }
)

// 在文件末尾追加路由块：
// --- Rate Module ---
@server (
    group:  rate
    prefix: v1/rate
)
service gateway {
    @doc "获取汇率（公开接口，无需鉴权）"
    @handler GetRate
    get / (GetRateReq) returns (GetRateResp)

    @doc "获取支持的货币列表"
    @handler GetCurrencies
    get /currencies (GetCurrenciesReq) returns (GetCurrenciesResp)
}
```

**重新生成 Gateway 代码**：
```powershell
goctl api go -api desc/gateway.api -dir app/gateway --style=goZero
```

---

## 3.2 Step 2: 更新 Gateway Config

**文件**: `app/gateway/internal/config/config.go`

```go
type Config struct {
    rest.RestConf
    // ... 已有字段 ...
    UserRpc        zrpc.RpcClientConf
    InteractionRpc zrpc.RpcClientConf
    RankingRpc     zrpc.RpcClientConf
    RateRpc        zrpc.RpcClientConf  // ← 新增
    // ...
}
```

---

## 3.3 Step 3: 更新 Gateway ServiceContext

**文件**: `app/gateway/internal/svc/servicecontext.go`

```go
import (
    // ... 已有 import ...
    "freeexchanged/app/rate/cmd/rpc/rateclient"  // ← 新增
)

type ServiceContext struct {
    // ... 已有字段 ...
    RateRpc rateclient.Rate  // ← 新增（使用强类型 client）
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        // ... 已有初始化 ...
        RateRpc: rateclient.NewRate(zrpc.MustNewClient(c.RateRpc)),  // ← 新增
    }
}
```

---

## 3.4 Step 4: 实现 Gateway Logic

**文件**: `app/gateway/internal/logic/rate/getratelogic.go`

```go
package logic

import (
    "context"
    "time"

    "freeexchanged/app/gateway/internal/svc"
    "freeexchanged/app/gateway/internal/types"
    ratepb "freeexchanged/app/rate/cmd/rpc/rate"

    "github.com/zeromicro/go-zero/core/logx"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type GetRateLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewGetRateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRateLogic {
    return &GetRateLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *GetRateLogic) GetRate(req *types.GetRateReq) (resp *types.GetRateResp, err error) {
    rateResp, err := l.svcCtx.RateRpc.GetRate(l.ctx, &ratepb.GetRateReq{
        From: req.From,
        To:   req.To,
    })
    if err != nil {
        // 区分错误类型，给前端友好提示
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.NotFound:
                return nil, fmt.Errorf("不支持的货币对: %s -> %s", req.From, req.To)
            case codes.InvalidArgument:
                return nil, fmt.Errorf("参数错误: %s", st.Message())
            }
        }
        return nil, err
    }

    return &types.GetRateResp{
        From:      rateResp.From,
        To:        rateResp.To,
        Rate:      rateResp.Rate,
        UpdatedAt: rateResp.UpdatedAt,
    }, nil
}
```

---

## 3.5 Step 5: 更新 gateway.yaml

在 `app/gateway/etc/gateway.yaml` 追加 Rate RPC 配置：

```yaml
# 在已有的 RankingRpc 后面追加：
RateRpc:
  Target: consul://127.0.0.1:8500/rate.rpc?wait=14s
```

---

## 3.6 验证

**Step 1**: 启动 Rate Job（先让 Redis 有数据）：
```powershell
go run app/rate/cmd/job/job.go -f app/rate/cmd/job/etc/job.yaml
```
等待约 5 秒，看到 `successfully updated rates for USD` 等日志。

**Step 2**: 启动 Rate RPC：
```powershell
go run app/rate/cmd/rpc/rate.go -f app/rate/cmd/rpc/etc/rate.yaml
```

**Step 3**: 重启 Gateway，然后测试：
```powershell
# 查询 USD → CNY 汇率
curl.exe "http://localhost:8888/v1/rate?from=USD&to=CNY"
# 期望返回:
# {"from":"USD","to":"CNY","rate":7.2534,"updated_at":1708300000}

# 查询支持的货币列表
curl.exe "http://localhost:8888/v1/rate/currencies"
# 期望返回:
# {"currencies":["USD","CNY","EUR","GBP","JPY","HKD","KRW","SGD","AUD","CAD"]}
```

---

# Part 4: 面试话术

## Q1: 你们的汇率数据是怎么获取和存储的？

**满分回答**：
"我们采用了 **Cache Aside Pattern（旁路缓存）** 来管理汇率数据。

架构上分两个进程：
1. **Rate Job**（定时任务）：每分钟调用 `open.er-api.com` 免费 API，拉取 10 种主流货币的汇率，以 **Redis Hash** 格式写入缓存，Key 是 `rate:{基准货币}`，Field 是目标货币，Value 是汇率值。同时设置 1 小时 TTL 作为兜底。
2. **Rate RPC**（查询服务）：只负责从 Redis 读取，不直接调用外部 API。

这样设计的好处是：
- **读写分离**：Job 和 RPC 完全解耦，互不影响。
- **防止缓存击穿**：所有读请求都走缓存，外部 API 只被 Job 调用，不会因为并发请求导致 API 被打爆。
- **高可用**：即使外部 API 短暂不可用，缓存数据还能支撑 1 小时。"

---

## Q2: 为什么用 Redis Hash 而不是 String？

**满分回答**：
"如果用 String，每个货币对需要一个独立的 Key，比如 `rate:USD:CNY`、`rate:USD:EUR`、`rate:USD:GBP`...

我们支持 10 种基准货币，每种货币对应 170+ 个目标货币，那就是 **1700+ 个 Key**。

用 Hash 的话，每种基准货币只需要 **1 个 Key**，所有目标货币作为 Field 存在同一个 Hash 里，只需要 10 个 Key。

另外，Hash 在 Redis 内部有特殊的内存优化（ziplist 编码），当 Field 数量较少时，内存占用比同等数量的 String Key 小得多。"

---

## Q3: 定时任务挂了怎么办？

**满分回答**：
"我们做了两层保护：

**第一层：TTL 兜底**。
Redis 里的汇率数据 TTL 是 1 小时。Job 每分钟更新一次，正常情况下数据永远是新鲜的。即使 Job 挂了，缓存数据还能撑 1 小时，期间 RPC 服务照常提供服务。

**第二层：错误隔离**。
Job 在拉取每种货币时，单个货币失败不会影响其他货币。比如 CNY 的 API 请求超时，USD、EUR 等其他货币照常更新。

**监控告警（生产环境）**：
通过 Prometheus 监控 Job 的执行状态，如果连续 3 次更新失败，触发告警通知运维介入。"

---

## Q4: 汇率数据的精度怎么保证？

**满分回答**：
"汇率数据我们保留 6 位小数（如 `7.253400`），存储为字符串格式，读取时用 `strconv.ParseFloat` 转换为 `float64`。

`float64` 有 15-17 位有效数字，对于汇率这种精度要求（通常 4-6 位小数）完全足够。

如果是金融级别的精度要求（比如外汇交易系统），应该用 `decimal` 库（如 `shopspring/decimal`）来避免浮点数精度问题。我们这个项目是展示用途，`float64` 足够。"

---

# Part 5: Phase 7 总结

**我们实现了什么**：
- ✅ Rate Job：定时拉取汇率，写入 Redis Hash，TTL 兜底
- ✅ Rate RPC：从 Redis 读取汇率，支持大小写不敏感
- ✅ Gateway 集成：暴露 `GET /v1/rate` 和 `GET /v1/rate/currencies`
- ✅ Cache Aside Pattern：读写分离，防缓存击穿

**技术亮点**：
```
外部 API → Job(定时) → Redis Hash → RPC(只读) → Gateway → 用户
                              ↑
                         TTL 1小时兜底
```

**面试一句话总结**：
> "汇率服务采用 Cache Aside Pattern，Job 定时写缓存，RPC 只读缓存，用 Redis Hash 存储货币对，TTL 兜底保证高可用。"

