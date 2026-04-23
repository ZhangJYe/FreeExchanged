package svc

import (
	"freeexchanged/app/interaction/cmd/rpc/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config config.Config
	Conn   sqlx.SqlConn
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config: c,
		Conn:   sqlx.NewMysql(c.DataSource),
	}
}
