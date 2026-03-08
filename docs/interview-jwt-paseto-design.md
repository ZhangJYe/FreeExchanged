# 面试指南：项目中的 Token 鉴权设计与 PASETO 实践

在微服务架构或分布式系统中，安全、高效、可扩展的鉴权机制是系统的核心基石。本项目的鉴权体系基于 **Token-Based Authentication**，并且在设计上做了一套完善的接口抽象，使得底层实现在 JWT 与 PASETO 之间可以无缝切换，目前实际落地采用了业界更为先进和安全的 **PASETO (Platform-Agnostic Security Tokens)** 协议。

本文档将深入分析本项目中 Token 鉴权的架构设计、PASETO 的优势以及完整的生命周期（生成、校验、注销）。

---

## 1. 核心架构设计

为了保证系统未来的可扩展性和降低框架耦合，我们在架构上做了一个非常优秀的设计：**依赖倒置（依赖接口，而非具体实现）**。

### 1.1 抽象 Maker 接口 (`pkg/token/paseto.go`)
我们在系统的 `pkg` 公共组件包中定义了一个 `Maker` 接口，所有的业务逻辑（如 Login 生成 Token，Middleware 校验 Token）都只依赖这个接口：

```go
// Maker 抽象接口：方便以后换 JWT/PASETOv4 或者做单测
type Maker interface {
    CreateToken(userID int64, duration time.Duration) (string, *Payload, error)
    VerifyToken(token string) (*Payload, error)
}
```

**面试答题要点（设计模式亮点）**：
*   **单一职责**：将 Token 的生成和校验职责收拢在一处。
*   **开闭原则与里氏替换**：如果有朝一日 PASETO 发现严重漏洞需要切回 JWT，或者需要升级到 PASETO v4，只需要新增一个实现了 `Maker` 接口的结构体并在初始化的地方替换即可，业务层代码 (`loginlogic`, `middleware`) 甚至不需要改动一行代码。
*   **便于单元测试**：在测试业务逻辑时，可以轻易地注入一个 `MockMaker`。

### 1.2 标准化 Payload 设计 (`pkg/token/payload.go`)
参考了 JWT 的标准 Claims 设计思想，我们封装了一个通用的 `Payload` 结构体：

```go
type Payload struct {
    ID        string    `json:"jti"`           // 万能的 jti (JWT ID)，用于精准撤销
    UserID    int64     `json:"uid"`           // 业务核心，用户唯一标识
    Issuer    string    `json:"iss,omitempty"` // 签发方
    Audience  string    `json:"aud,omitempty"` // 接收方
    IssuedAt  time.Time `json:"iat"`           // 签发时间
    NotBefore time.Time `json:"nbf,omitempty"` // 生效时间
    ExpiredAt time.Time `json:"exp"`           // 过期时间
}
```

**面试答题要点**：
为什么不直接把所有零散的字段塞进去，而是严格按照标准模型设计？规范的 payload 字段名（符合 RFC 定义）能让系统间交互更容易。尤其是 `ID (jti)`，对我们系统后续实现安全注销（踢人）的黑名单机制起到了至关重要的作用。

---

## 2. 为什么选择 PASETO 而非 JWT？

这是一个极大概率被面试官问到的**鉴别度极高的技术选型问题**。

项目中使用了 `github.com/o1egl/paseto`，并且具体使用的是 **PASETO v2.local** 版本。

### 2.1 JWT 的常见安全隐患（对比衬托）
当面试官问你为什么不用 JWT 时，你可以指出 JWT 存在的这几个根本性设计缺陷：
1.  **"alg": "none" 漏洞**：由于 JWT 将算法 `alg` 置于公开的头部（Header）中，历史上曾出现过攻击者将 `alg` 改为 `none`，导致服务端直接跳过签名验证的严重漏洞。
2.  **非对称加密变成对称加密漏洞（RS256 降级 HS256）**：攻击者将原本使用公私钥对配置的 RS256 篡改为使用对称加密 HS256，利用获取到的公钥作为对称密钥成功伪造 Token。
3.  **开发者心智负担重**：开发者在使用 JWT 时需要面临大量的配置项和算法选择（几十种加密算法），选错了就容易导致安全漏洞。

### 2.2 PASETO 的降维打击优势
PASETO (Platform-Agnostic Security Tokens) 直接锁死了算法，不给开发者试错的机会。
在我们的项目中，选择了 **V2.local** 版本：
*   **V2** 对应的底层密码学算法是被安全界公认强健的 `ChaCha20-Poly1305` 或 `Ed25519`。
*   **.local** 代表是对称加密（Symmetric Encryption）。不仅保证了 Token 数据的**完整性（防篡改）**，由于使用了加密算法而不是单纯的编码，同时也保证了**机密性（Token 中的 payload 不是明文，无法被反序列化偷窥信息）**。

> **💡 面试原话总结**："比起 JWT 将使用什么算法的权力交给客户端（头部携带 alg），PASETO 的哲学是**服务端掌握绝对控制权**。因为我明确要在内网信任环境做对称加密，所以我直接采用 PASETO v2.local，开发者不需要选算法，底层直接写死最安全的现代算法套件，杜绝了一切降级攻击的可能，同时自带 Payload 的数据加密，这比 JWT 仅仅做 Base64 编码并在尾部附上签名要安全得多。"

---

## 3. 业务链路落地详解

项目中关于 Token 的生命周期主要包含：**签发(Login)** -> **校验(Middleware)** -> **注销(Logout)**。

### 3.1 签发 Token (Login)
位于 `app/user/cmd/rpc/internal/logic/loginlogic.go`
1. 查询用户明文密码对应的 Hash 密码是否匹配。
2. 匹配成功后，调用 `svcCtx.TokenMaker.CreateToken`。
3. `CreateToken` 内部会利用 `uuid.NewString()` 生成唯一的 `ID(jti)`，并将用户ID(`uid`) 打包。
4. 使用 `paseto.NewV2().Encrypt()` 用 32 bytes 的对称密钥加密成字符串 Token，并返回客户端。

### 3.2 拦截校验 Token (API Gateway)
位于 `app/gateway/internal/middleware/pasetomiddleware.go`
我们没有在每个业务服务里去解析 token，而是利用 Api Gateway 在最外层做统一拦截：
1. 校验请求头：提取 `Authorization: Bearer <token>`
2. 校验合法性：调用 `TokenMaker.VerifyToken(accessToken)`，不仅会在底层解密并校验签名、防篡改，同时我们的 `Payload.Valid()` 还会检查 `Exp` (是否过期) 和 `Nbf` (是否提前使用)。
3. **黑名单校验（重要亮点）**：去 Redis 中校验这个 Token 的 ID (`jti`) 是否存在于黑名单中。
4. 上下文传递：将解密出的 `userId` 注入到 http `context` 中，向后流转给具体的业务 Logic。

### 3.3 无状态向有状态的妥协：安全注销 (Logout)
由于 PASETO 和 JWT 一样，本身是**无状态（Stateless）**的，即签发后只要不到过期时间，就算用户点击了“退出登录”或者被管理员“封禁”，Token 本身在数学上依然是合法的。为了彻底干掉一个作废的 Token，我们引入了 Redis 组成**黑名单机制**。

位于 `app/user/cmd/rpc/internal/logic/logoutlogic.go`
1. 当前系统调用注销接口时，带上即将被注销的 Token。
2. 解析 Token 获取其中的 `ID(jti)`（Token 的唯一身份证号）。
3. 计算该 Token 距离原生过期时间（`ExpiredAt`）还差多少秒（`Duration`）。
4. **将 `jti` 作为 Key 存入 Redis 黑名单** (`TokenBlacklistPrefix + ID`)。
5. **神来之笔：设置 TTL 为计算出的剩余 `Duration`**。
   * *为什么要设 TTL？* 因为只需要防范在它原生生命周期内被别人滥用。一旦到达其原生过期时间，底层 `VerifyToken(过期时间)` 就直接拦截了，不再需要经过 Redis。这保证了 Redis 黑名单的体积永远不会无限膨胀，而是动态滚动的！

---

## 4. 面试高频问题与对答演练 (Q&A)

**Q1：了解过你们项目的鉴权机制吗？为什么不用传统的 Session？**
> **答**：因为我们是微服务架构，并且业务可能有高并发的需求。使用 Session 就涉及到服务端状态同步的问题（Session 集中存储的单点瓶颈）。使用 Token 机制（如 PASETO）则是无状态的，Api 网关甚至任意微服务节点都可以独立通过密钥完成对 Token 的自验证，不需要每次向中心服务器去做 DB 查询比对，扩展性极强。

**Q2：那为什么你用了无状态的 PASETO，后续注销的时候又引入了 Redis 黑名单，这不是又变回有状态了吗？**
> **答**：这是一个安全折中的权衡。纯天然无状态的 Token 的最大缺陷就是**无法被主动撤销**！如果用户发现账号被盗，哪怕立刻改了密码，黑客手里拿着那个原本有效的 Token 在过期之前是可以一直为所欲为的。
> 所以我们引入了轻量级的“有状态”妥协方案：黑名单。而且我们在设计中非常克制，并非每次校验都在缓存里去匹配庞大的白名单记录。大部分正常请求走的是验证代码自校验算法，只有验证成功了我们才会额外查一次 Redis O(1) 的黑名单排异。并且利用 Token 原生的剩余存活时长作为 Redis Key 的 TTL，保持了 Redis 极小的内存占用，完美实现了高性能自包含校验与强制撤销控制的平衡。

**Q3：你提到代码里设计了 Maker 接口，如果让你现在将 PASETO 替换为 JWT，你需要做哪些改动？**
> **答**：基本上属于零业务侵入。我会新建一个 `jwt_maker.go`，里面定义一个结构体实现 `time.Duration) (string, *Payload, error)` 和 `VerifyToken(token string) (*Payload, error)` 两个方法，内部调用 golang-jwt 库的 Api。
> 随后在网关和 user 微服务初始化 `ServiceContext` 的时候，将注入的 `token.NewPasetoMaker` 修改为 `token.NewJWTMaker` 即可，其他地方一行代码都不需要动！这完全得益于依赖抽象接口设计的优势。
