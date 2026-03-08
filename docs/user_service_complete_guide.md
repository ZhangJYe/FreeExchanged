# User Service (RPC) 开发指南

老弟啊，来，咱把这个 User 服务的全貌梳理得清清楚楚。你跟着我的节奏走，保证你不仅能把代码写出来，还能跟面试官吹得头头是道。

## 一、 整体设计思路

User 服务是我们微服务的基石，它的核心职责就俩：**存数据**（MySQL）和 **发令牌**（Paseto Token）。

1.  **架构模式**：采用 go-zero 的标准 RPC 架构。
2.  **服务发现**：我们用了 Consul，因为它是现在云原生环境（非 K8s 强绑定时）除了 Etcd 外最流行的选择。
3.  **安全性**：
    *   密码绝不存明文，必须用 `bcrypt` 加盐哈希。
    *   鉴权用 `Paseto`，比 JWT 更安全，不用担心弱加密算法攻击。

---

## 二、 目录结构梳理

老弟你看下这个结构，User 服务的核心资产都在这儿了：

```text
app/user/cmd/rpc/
├── etc/
│   └── user.yaml          # 配置文件（MySQL、Redis、Consul、Secret都在这）
├── internal/
│   ├── config/            # 配置对应的 Go 结构体
│   ├── svc/               # ServiceContext（资源管理器，MySQL/Redis/Consul连接都在这初始化）
│   ├── server/            # Go-Zero 生成的 grpc server 注册代码（不用动）
│   └── logic/             # ★★★ 业务逻辑核心 ★★★
│       ├── registerlogic.go  # 注册逻辑
│       ├── loginlogic.go     # 登录逻辑
│       └── pinglogic.go      # 健康检查（建议保留，后面细说）
└── user.go                # main 入口
```

---

## 三、 Step-by-Step 开发流程

### 第一步：依赖与基础库 (Infrastructure)

**1. 生成安全密钥 (pkg/token)**
咱们为了安全，手写了个生成 32 字节 Base64 Key 的小脚本（你之前做过的 `gen_key`）。这个 Key 填在 yaml 里，Paseto 就靠它加密。

**2. 密码加密工具 (pkg/utils)**
在 `pkg/utils/password.go` 里，我们封装了 `bcrypt`。
*   `HashPassword`: 注册时用，把明文变成密文。
*   `CheckPassword`: 登录时用，拿着明文跟数据库的密文比对。

### 第二步：配置文件 (Configuration)

在 `etc/user.yaml` 里，我们配好了三剑客：
*   **MySQL**: 存用户资料 (User 表)。
*   **Redis**: 做缓存，或者将来做防爆破限制。
*   **Consul**: 告诉别人 "我 user 服务要在 8080 端口营业了"。

### 第三步：资源初始化 (ServiceContext)

在 `svc/servicecontext.go` 里，我们把所有依赖都 `New` 出来了：
*   `conn := sqlx.NewMysql(...)` -> 连数据库。
*   `redis.MustNewRedis(...)` -> 连缓存（连不上直接 panic，防止带病工作）。
*   `token.NewPasetoMaker(...)` -> 初始化发令牌的印钞机。

### 第四步：核心业务逻辑 (Logic)

**1. 注册 (RegisterLogic)**
流程很简单：
1.  校验密码长度。
2.  `HashPassword` 加密密码。
3.  `UserModel.Insert` 落库。如果报错 "Duplicate entry"，直接返回 `AlreadyExists`。
4.  成功后，顺手签发一个 Token，让用户注册完直接通过。

**2. 登录 (LoginLogic)**
流程也很清晰：
1.  `UserModel.FindOneByMobile` 查人。没查到？返回 `NotFound`。
2.  `CheckPassword` 验密。不对？返回 `Unauthenticated`。
3.  `TokenMaker.CreateToken` 签发 Token。
4.  返回 UserID + Token + 过期时间。

---

## 四、 关于 PingLogic 删不删？

老弟，**千万别删**。

虽然它现在看起来很傻，只有一行代码，但它有两个重要作用：
1.  **Kubernetes 存活探针 (Liveness Probe)**：K8s 会定期调这个接口，问服务 "你死没死？"。如果删了，K8s 可能不知道你挂了。
2.  **调试用**：当你连不上服务时，这是最简单的排查手段。

---

## 五、 接下来的计划 (Phase 3)

User RPC 服务现在已经是**完全体**了。

**下一步建议**：
既然后端已经就绪，咱们可以写个简单的 **Test Client** (main 函数)，模拟 Gateway 去调用一下 User RPC，真实地走一遍 "注册 -> 登录" 的流程。亲眼看到 Token 生成，那才叫放心。

老弟，怎么样？这个思路清晰不？
