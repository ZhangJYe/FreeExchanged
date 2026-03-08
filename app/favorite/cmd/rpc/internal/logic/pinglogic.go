package logic

import (
	"context"

	"freeexchanged/app/favorite/cmd/rpc/favorite"
	"freeexchanged/app/favorite/cmd/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type PingLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PingLogic {
	return &PingLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PingLogic) Ping(in *favorite.PingReq) (*favorite.PingResp, error) {
	// todo: add your logic here and delete this line

	return &favorite.PingResp{}, nil
}
