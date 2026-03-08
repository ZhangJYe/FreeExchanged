package logic

import (
	"context"
	"fmt"
	"time"

	"freeexchanged/app/user/cmd/rpc/internal/svc"
	"freeexchanged/app/user/cmd/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// Redis Key 前缀
	TokenBlacklistPrefix = "token:blacklist:"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Logout 注销 Token
func (l *LogoutLogic) Logout(in *pb.LogoutReq) (*pb.LogoutResp, error) {
	// 1. 验证并解析 Token
	// 我们直接复用 TokenMaker，如果 Token 已经无效（过期或签名不对），则不需要加入黑名单
	payload, err := l.svcCtx.TokenMaker.VerifyToken(in.Token)
	if err != nil {
		// Token 无效或已过期，直接返回成功即可，没必要再拉黑
		l.Logger.Infof("Logout: token invalid or expired, ignore. err: %v", err)
		return &pb.LogoutResp{}, nil
	}

	// 2. 计算剩余有效期
	now := time.Now()
	duration := payload.ExpiredAt.Sub(now)
	if duration <= 0 {
		return &pb.LogoutResp{}, nil
	}

	// 3. 存入 Redis 黑名单
	// Key: token:blacklist:<jti>
	// Value: userId (可选)
	// TTL: 剩余有效期
	key := fmt.Sprintf("%s%s", TokenBlacklistPrefix, payload.ID)
	err = l.svcCtx.RedisClient.Setex(key, fmt.Sprintf("%d", payload.UserID), int(duration.Seconds()))
	if err != nil {
		l.Logger.Errorf("Logout: set redis blacklist failed: %v", err)
		return nil, err
	}

	l.Logger.Infof("Token revoked: jti=%s, uid=%d", payload.ID, payload.UserID)
	return &pb.LogoutResp{}, nil
}
