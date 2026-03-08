package logic

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"freeexchanged/app/rate/cmd/rpc/internal/svc"
	"freeexchanged/app/rate/cmd/rpc/rate"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Redis Key 格式: rate:{FROM_CURRENCY}
// 例如: rate:USD → Hash{CNY: "7.2534", EUR: "0.9234", ...}
const (
	rateKeyPrefix  = "rate:"
	updatedAtField = "_updated_at" // 特殊 field，存储最后更新时间
)

type GetRateLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRateLogic {
	return &GetRateLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetRateLogic) GetRate(in *rate.GetRateReq) (*rate.GetRateResp, error) {
	// 参数校验：转大写，防止大小写问题
	from := strings.ToUpper(strings.TrimSpace(in.From))
	to := strings.ToUpper(strings.TrimSpace(in.To))

	if from == "" || to == "" {
		return nil, status.Error(codes.InvalidArgument, "from and to are required")
	}

	// 相同货币直接返回 1
	if from == to {
		return &rate.GetRateResp{
			From:      from,
			To:        to,
			Rate:      1.0,
			UpdatedAt: time.Now().Unix(),
		}, nil
	}

	// 从 Redis Hash 读取汇率: HGET rate:USD CNY
	redisKey := fmt.Sprintf("%s%s", rateKeyPrefix, from)
	rateStr, err := l.svcCtx.Redis.Hget(redisKey, to)
	if err != nil {
		l.Errorf("redis hget failed, key=%s field=%s err=%v", redisKey, to, err)
		return nil, status.Error(codes.Internal, "failed to get rate from cache")
	}
	if rateStr == "" {
		// 缓存未命中：数据未就绪（Job 还没跑）或不支持的货币对
		return nil, status.Errorf(codes.NotFound,
			"rate not found for %s->%s, data may not be ready yet", from, to)
	}

	// 解析汇率值
	rateVal, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return nil, status.Error(codes.Internal, "invalid rate data in cache")
	}

	// 读取最后更新时间
	updatedAtStr, _ := l.svcCtx.Redis.Hget(redisKey, updatedAtField)
	updatedAt, _ := strconv.ParseInt(updatedAtStr, 10, 64)

	return &rate.GetRateResp{
		From:      from,
		To:        to,
		Rate:      rateVal,
		UpdatedAt: updatedAt,
	}, nil
}
