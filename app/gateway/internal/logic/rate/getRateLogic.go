package logic

import (
	"context"
	"fmt"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	ratepb "freeexchanged/app/rate/cmd/rpc/rate"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GetRateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRateLogic {
	return &GetRateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRateLogic) GetRate(req *types.GetRateReq) (resp *types.GetRateResp, err error) {
	rateResp, err := l.svcCtx.RateRpc.GetRate(l.ctx, &ratepb.GetRateReq{
		From: req.From,
		To:   req.To,
	})
	if err != nil {
		// 区分 gRPC 错误类型，给前端友好提示
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, fmt.Errorf("不支持的货币对: %s -> %s", req.From, req.To)
			case codes.InvalidArgument:
				return nil, fmt.Errorf("参数错误: %s", st.Message())
			}
		}
		return nil, err
	}

	return &types.GetRateResp{
		From:      rateResp.From,
		To:        rateResp.To,
		Rate:      rateResp.Rate,
		UpdatedAt: rateResp.UpdatedAt,
	}, nil
}
