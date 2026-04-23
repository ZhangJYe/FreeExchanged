package logic

import (
	"context"

	"freeexchanged/app/interaction/cmd/rpc/internal/svc"
	"freeexchanged/app/interaction/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type LikeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLikeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LikeLogic {
	return &LikeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LikeLogic) Like(in *pb.LikeReq) (*pb.LikeResp, error) {
	changed, err := recordLike(l.ctx, l.svcCtx.Conn, in.UserId, in.ArticleId)
	if err != nil {
		l.Logger.Errorf("Like: record state error: %v", err)
		return nil, err
	}
	if !changed {
		l.Logger.Infof("Like ignored because state already liked: uid=%d, aid=%d", in.UserId, in.ArticleId)
		return &pb.LikeResp{}, nil
	}

	l.Logger.Infof("Like event enqueued: uid=%d, aid=%d", in.UserId, in.ArticleId)
	return &pb.LikeResp{}, nil
}
