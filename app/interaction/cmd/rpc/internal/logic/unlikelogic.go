package logic

import (
	"context"

	"freeexchanged/app/interaction/cmd/rpc/internal/svc"
	"freeexchanged/app/interaction/cmd/rpc/pb"

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
	changed, err := recordUnlike(l.ctx, l.svcCtx.Conn, in.UserId, in.ArticleId)
	if err != nil {
		l.Logger.Errorf("Unlike: record state error: %v", err)
		return nil, err
	}
	if !changed {
		l.Logger.Infof("Unlike ignored because state is not liked: uid=%d, aid=%d", in.UserId, in.ArticleId)
		return &pb.UnlikeResp{}, nil
	}

	l.Logger.Infof("Unlike event enqueued: uid=%d, aid=%d", in.UserId, in.ArticleId)
	return &pb.UnlikeResp{}, nil
}
