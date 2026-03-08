package svc

import (
	"freeexchanged/app/user/cmd/rpc/internal/config"
	"freeexchanged/app/user/model"
	"freeexchanged/pkg/token"

	"github.com/zeromicro/go-zero/core/stores/redis" // 引入 go-zero 自带 redis
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config      config.Config
	UserModel   model.UserModel
	RedisClient *redis.Redis // 注入 Redis
	TokenMaker  token.Maker  // 注入 Paseto Maker
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DataSource)

	// 2. 初始化 Redis (从配置文件读取)
	// redisConf := c.Redis // 直接使用配置里的 RedisConf

	// 3. 初始化 Paseto Maker
	// AccessSecret 必须是 Base64 编码后的 32 字节 Key
	maker, err := token.NewPasetoMakerFromBase64Key(c.Identity.AccessSecret, "freeexchanged", "user")
	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config:      c,
		UserModel:   model.NewUserModel(conn),
		RedisClient: redis.MustNewRedis(c.BizRedis),
		TokenMaker:  maker,
	}
}
