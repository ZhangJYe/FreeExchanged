package svc

import (
	"freeexchanged/app/article/cmd/rpc/internal/config"
	"freeexchanged/app/article/cmd/rpc/internal/mq"
	"freeexchanged/app/article/model"

	_ "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config       config.Config
	ArticleModel model.ArticlesModel
	Producer     *mq.Producer
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.DataSource)
	return &ServiceContext{
		Config:       c,
		ArticleModel: model.NewArticlesModel(conn),
		Producer: mq.NewProducer(mq.RabbitMqConf{
			Host:     c.RabbitMQ.Host,
			Port:     c.RabbitMQ.Port,
			Username: c.RabbitMQ.Username,
			Password: c.RabbitMQ.Password,
		}),
	}
}
