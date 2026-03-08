# User Service 开发文档 (Phase 2) - 业务逻辑实现

本部分讲解如何在 `internal/logic` 层实现具体的 `Register` 和 `Login` 业务。

## 1. 注册逻辑 (RegisterLogic)

### 流程图
1.  **参数校验**：检查手机号格式、密码强度（可选）。
2.  **查重**：查询数据库，确认手机号未被注册。
    *   *注意：Model 层的 `Insert` 通常会有唯一索引冲突错误，也可以利用这个特性。*
3.  **密码加密**：**绝对不能明文存密码**。使用 `bcrypt` 对密码进行哈希。
4.  **落库**：写入 `user` 表。
5.  **生成 Token**：调用 `pkg/token` 生成 Paseto Token。
6.  **返回结果**：返回用户 ID、Token 和过期时间。

### 关键代码片段 (预告)

```go
// 1. 密码加密
// 需要在 pkg/utils/password.go 实现 HashPassword 函数
hashedPassword, err := utils.HashPassword(in.Password)
if err != nil {
    return nil, status.Error(codes.Internal, "failed to hash password")
}

// 2. 插入数据库
newUser := &model.User{
    Mobile:   in.Mobile,
    Password: hashedPassword,
    Nickname: in.Nickname,
}
res, err := l.svcCtx.UserModel.Insert(l.ctx, newUser)
if err != nil {
    // 处理唯一索引冲突 (error code 1062)
    if strings.Contains(err.Error(), "Duplicate entry") {
        return nil, status.Error(codes.AlreadyExists, "mobile already registered")
    }
    return nil, status.Error(codes.Internal, "database error")
}

// 3. 生成 Token
userId, _ := res.LastInsertId()
token, payload, err := l.svcCtx.TokenMaker.CreateToken(userId, time.Duration(l.svcCtx.Config.Auth.AccessExpire)*time.Second)
```

## 2. 登录逻辑 (LoginLogic)

### 流程图
1.  **查询用户**：根据手机号查询用户信息。若不存在，返回“用户不存在”。
2.  **校验密码**：使用 `bcrypt.CompareHashAndPassword` 比较输入密码和数据库哈希值。
3.  **生成 Token**：逻辑同注册。
4.  **返回结果**。

### 关键代码片段 (预告)

```go
// 1. 查询用户
user, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, in.Mobile)
if err == model.ErrNotFound {
    return nil, status.Error(codes.NotFound, "user not found")
}

// 2. 校验密码
if err := utils.CheckPassword(in.Password, user.Password); err != nil {
    return nil, status.Error(codes.Unauthenticated, "invalid password")
}

// 3. 签发 Token
token, payload, err := l.svcCtx.TokenMaker.CreateToken(user.Id, duration)
```

## 3. 完善 pkg/utils

为了支持上述逻辑，我们需要在 `pkg/utils` 下创建一个 `password.go` 文件，封装 `bcrypt` 操作。

### 依赖
```bash
go get golang.org/x/crypto/bcrypt
```

---

**准备好了吗？** 
按照这个文档，我们可以开始编写代码了。建议顺序：
1.  **完善 `pkg/utils`** (添加密码加密工具)
2.  **补充 `pkg/xerr`** (添加通用错误码，如 `ReuqestErr`, `DbErr`)
3.  **实现 `Register`**
4.  **实现 `Login`**
