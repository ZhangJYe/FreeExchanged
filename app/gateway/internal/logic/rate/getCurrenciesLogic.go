package logic

import (
	"context"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	ratepb "freeexchanged/app/rate/cmd/rpc/rate"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCurrenciesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCurrenciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrenciesLogic {
	return &GetCurrenciesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCurrenciesLogic) GetCurrencies(req *types.GetCurrenciesReq) (resp *types.GetCurrenciesResp, err error) {
	currResp, err := l.svcCtx.RateRpc.GetSupportedCurrencies(l.ctx, &ratepb.GetSupportedCurrenciesReq{})
	if err != nil {
		return nil, err
	}

	return &types.GetCurrenciesResp{
		Currencies: currResp.Currencies,
	}, nil
}
