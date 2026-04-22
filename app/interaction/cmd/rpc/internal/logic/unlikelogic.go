package logic

import (
	"context"

	"freeexchanged/app/interaction/cmd/rpc/internal/svc"
	"freeexchanged/app/interaction/cmd/rpc/pb"
	"freeexchanged/pkg/events"

	"github.com/zeromicro/go-zero/core/logx"
)

type UnlikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUnlikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UnlikeLogic {
	return &UnlikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UnlikeLogic) Unlike(in *pb.UnlikeReq) (*pb.UnlikeResp, error) {
	if err := publishInteractionEvent(l.ctx, l.svcCtx.EventProducer, events.EventInteractionUnlike, in.UserId, in.ArticleId); err != nil {
		l.Logger.Errorf("Unlike: publish msg error: %v", err)
		return nil, err
	}

	l.Logger.Infof("Unlike event published: uid=%d, aid=%d", in.UserId, in.ArticleId)
	return &pb.UnlikeResp{}, nil
}
