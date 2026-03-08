# User Service 开发文档 (Phase 1) - 基础架构与配置

本文档指导你如何从零构建 User 服务的核心部分，包括数据模型、服务发现配置和鉴权模块集成。

## 1. 核心依赖引入

由于我们使用 **Consul** 作为注册中心，首先确保 `go.mod` 引入了相关包：

```bash
go get github.com/zeromicro/zero-contrib/zrpc/registry/consul
go mod tidy
```

## 2. 配置文件详解 (etc/user.yaml)

User 服务不仅需要连接数据库和缓存，还需要配置 Paseto 鉴权密钥。

### 关键配置项

| 配置项 | 说明 | 示例值 |
| :--- | :--- | :--- |
| `Name` | 服务名称（用于 Consul 注册） | `user.rpc` |
| `ListenOn` | 监听地址 | `0.0.0.0:8080` |
| `Consul` | 注册中心配置 | `Host: 127.0.0.1:8500`, `Key: user.rpc` |
| `DataSource` | MySQL 连接串 | `root:root@tcp(127.0.0.1:3307)/freeexchanged...` |
| `Auth` | Paseto 鉴权配置 | `AccessExpire: 86400` (秒) |

### ⚠️ 特别注意：AccessSecret

Paseto V2 (Local模式) 要求密钥必须是 **32字节** 的随机字符串。这里的配置项推荐存 **Base64 编码后的字符串**，在代码中解码。

**如何生成合法的 AccessSecret?**
你已经在 `pkg/token/paseto.go` 中编写了 `GenerateSymmetricKeyBase64()` 工具函数。你可以写个临时 main 函数调用它生成一个 Key，填入 `user.yaml`。

## 3. 服务上下文 (ServiceContext)

`ServiceContext` 是 go-zero 中资源初始化的核心位置。在这里我们将初始化：
1.  **MySQL 连接** (`sqlx.SqlConn`)
2.  **Redis 连接** (`redis.Redis`)
3.  **Paseto Maker** (用于 Token 生成)

### 代码逻辑预览

```go
type ServiceContext struct {
    Config      config.Config
    UserModel   model.UserModel
    RedisClient *redis.Redis
    TokenMaker  token.Maker // 新增：鉴权工具接口
}

func NewServiceContext(c config.Config) *ServiceContext {
    // 1. 初始化 DB
    conn := sqlx.NewMysql(c.DataSource)
    
    // 2. 初始化 Redis
    redisConf := redis.RedisConf{
        Host: "127.0.0.1:6380",
        Type: "node",
    }
    
    // 3. 初始化 Paseto Maker
    // 注意：这里需要从配置读取 Base64 Key 并解码
    maker, err := token.NewPasetoMakerFromBase64Key(c.Auth.AccessSecret, "freeexchanged", "user")
    if err != nil {
        panic(fmt.Sprintf("failed to create token maker: %v", err))
    }

    return &ServiceContext{
        Config:      c,
        UserModel:   model.NewUserModel(conn),
        RedisClient: redis.MustNewRedis(redisConf),
        TokenMaker:  maker,
    }
}
```

## 4. 数据模型 (Model)

我们已经通过 `goctl model` 生成了 `user` 表的基础代码。
User 服务的核心业务逻辑（注册）将强依赖于Model层提供的：
- `Insert`: 插入新用户
- `FindOneByMobile`: 检查手机号是否已注册

---

**下一步**：在 `Phase 2` 文档中，我们将详细讲解如何编写 `Register` 和 `Login` 的业务逻辑。
