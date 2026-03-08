package logic

import (
	"context"

	"freeexchanged/app/rate/cmd/rpc/internal/svc"
	"freeexchanged/app/rate/cmd/rpc/rate"

	"github.com/zeromicro/go-zero/core/logx"
)

// 支持的货币列表（与 Job 拉取的保持一致）
var supportedCurrencies = []string{
	"USD", "CNY", "EUR", "GBP", "JPY",
	"HKD", "KRW", "SGD", "AUD", "CAD",
}

type GetSupportedCurrenciesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSupportedCurrenciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSupportedCurrenciesLogic {
	return &GetSupportedCurrenciesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetSupportedCurrenciesLogic) GetSupportedCurrencies(in *rate.GetSupportedCurrenciesReq) (*rate.GetSupportedCurrenciesResp, error) {
	return &rate.GetSupportedCurrenciesResp{
		Currencies: supportedCurrencies,
	}, nil
}
