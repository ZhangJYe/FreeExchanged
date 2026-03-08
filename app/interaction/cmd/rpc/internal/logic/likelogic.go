package logic

import (
	"context"
	"encoding/json"
	"time"

	"freeexchanged/app/interaction/cmd/rpc/internal/svc"
	"freeexchanged/app/interaction/cmd/rpc/pb"

	"github.com/streadway/amqp"
	"github.com/zeromicro/go-zero/core/logx"
)

type LikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

type LikeMsg struct {
	UserId    int64 `json:"user_id"`
	ArticleId int64 `json:"article_id"`
	Timestamp int64 `json:"timestamp"`
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LikeLogic) Like(in *pb.LikeReq) (*pb.LikeResp, error) {
	// 1. (Optional) 这里的 Check Duplicate (去重) 逻辑可以先不做，或者之后加 Redis Bitmap
	// 目前先做全量发消息

	msg := LikeMsg{
		UserId:    in.UserId,
		ArticleId: in.ArticleId,
		Timestamp: time.Now().Unix(),
	}
	body, _ := json.Marshal(msg)

	// 2. 发送消息到 RabbitMQ
	err := l.svcCtx.MqChannel.Publish(
		"interaction.topic", // exchange
		"article.like",      // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	)

	if err != nil {
		l.Logger.Errorf("Like: publish msg error: %v", err)
		return nil, err
	}

	l.Logger.Infof("Like event published: uid=%d, aid=%d", in.UserId, in.ArticleId)
	return &pb.LikeResp{}, nil
}
