package svc

import (
	"freeexchanged/app/article/cmd/rpc/internal/config"
	"freeexchanged/app/article/model"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config       config.Config
	Conn         sqlx.SqlConn
	ArticleModel model.ArticlesModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DataSource)
	return &ServiceContext{
		Config:       c,
		Conn:         conn,
		ArticleModel: model.NewArticlesModel(conn),
	}
}
