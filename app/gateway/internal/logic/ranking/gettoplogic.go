// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package ranking

import (
	"context"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	rankingclient "freeexchanged/app/ranking/cmd/rpc/rankingclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetTopLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取热榜
func NewGetTopLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTopLogic {
	return &GetTopLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetTopLogic) GetTop(req *types.GetTopReq) (resp *types.GetTopResp, err error) {
	// 直接使用 svcCtx.RankingRpc (已经是 Ranking 接口，无需重新 NewRanking)
	rpcResp, err := l.svcCtx.RankingRpc.GetTop(l.ctx, &rankingclient.GetTopReq{
		N: req.N,
	})
	if err != nil {
		l.Logger.Errorf("Call Ranking RPC failed: %v", err)
		return nil, err
	}

	var items []types.RankItem
	if rpcResp.Items != nil {
		items = make([]types.RankItem, len(rpcResp.Items))
		for i, v := range rpcResp.Items {
			items[i] = types.RankItem{
				ArticleId: v.ArticleId,
				Score:     v.Score,
				Title:     v.Title,
			}
		}
	}

	return &types.GetTopResp{
		Items: items,
	}, nil
}
