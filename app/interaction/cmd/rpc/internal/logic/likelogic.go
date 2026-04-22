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
	if err := publishInteractionEvent(l.svcCtx.MqChannel, "like", in.UserId, in.ArticleId); err != nil {
		l.Logger.Errorf("Like: publish msg error: %v", err)
		return nil, err
	}

	l.Logger.Infof("Like event published: uid=%d, aid=%d", in.UserId, in.ArticleId)
	return &pb.LikeResp{}, nil
}
