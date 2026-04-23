package logic

import (
	"context"
	"fmt"
	"strconv"

	"freeexchanged/app/ranking/cmd/rpc/internal/svc"
	"freeexchanged/app/ranking/cmd/rpc/pb"
	"freeexchanged/app/ranking/internal/constant"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTopLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTopLogic {
	return &GetTopLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetTopLogic) GetTop(in *pb.GetTopReq) (*pb.GetTopResp, error) {
	// 1. 获取前 N 名
	// ZrevrangeWithScores: 从大到小排序
	// start: 0, stop: n-1
	if in.N <= 0 {
		return &pb.GetTopResp{}, nil
	}

	pairs, err := l.svcCtx.Redis.ZrevrangeWithScores(constant.RankingHotKey, 0, int64(in.N-1))
	if err != nil {
		l.Logger.Errorf("GetTop error: %v", err)
		return nil, err
	}

	var items []*pb.RankItem
	for _, pair := range pairs {
		aid, _ := strconv.ParseInt(pair.Key, 10, 64)
		items = append(items, &pb.RankItem{
			ArticleId: aid,
			Score:     int64(pair.Score),
			Title:     fmt.Sprintf("Article %d", aid), // 暂时 Mock 标题，以后可以通过 RPC 调 Article Service 获取
		})
	}

	return &pb.GetTopResp{
		Items: items,
	}, nil
}
