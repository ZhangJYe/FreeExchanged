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
	// todo: add your logic here and delete this line

	return &pb.ReadResp{}, nil
}
