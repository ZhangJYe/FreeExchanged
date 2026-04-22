package logic

import (
	"context"

	"freeexchanged/app/interaction/cmd/rpc/internal/svc"
	"freeexchanged/app/interaction/cmd/rpc/pb"
	"freeexchanged/pkg/events"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReadLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadLogic {
	return &ReadLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ReadLogic) Read(in *pb.ReadReq) (*pb.ReadResp, error) {
	if err := publishInteractionEvent(l.ctx, l.svcCtx.EventProducer, events.EventInteractionRead, in.UserId, in.ArticleId); err != nil {
		l.Logger.Errorf("Read: publish msg error: %v", err)
		return nil, err
	}

	l.Logger.Infof("Read event published: uid=%d, aid=%d", in.UserId, in.ArticleId)
	return &pb.ReadResp{}, nil
}
