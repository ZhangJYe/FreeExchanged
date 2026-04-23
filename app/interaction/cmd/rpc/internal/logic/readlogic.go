package logic

import (
	"context"

	"freeexchanged/app/interaction/cmd/rpc/internal/svc"
	"freeexchanged/app/interaction/cmd/rpc/pb"

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
	if err := recordRead(l.ctx, l.svcCtx.Conn, in.UserId, in.ArticleId); err != nil {
		l.Logger.Errorf("Read: record state error: %v", err)
		return nil, err
	}

	l.Logger.Infof("Read event enqueued: uid=%d, aid=%d", in.UserId, in.ArticleId)
	return &pb.ReadResp{}, nil
}
