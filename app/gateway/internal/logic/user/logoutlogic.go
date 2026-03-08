package user

import (
	"context"
	"errors"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"freeexchanged/app/user/cmd/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户注销 (需要鉴权)
func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutLogic) Logout(req *types.LogoutReq) (resp *types.LogoutResp, err error) {
	// 从 Context 中获取 Token (由 Middleware 注入)
	tokenObj := l.ctx.Value("token")
	if tokenObj == nil {
		return nil, errors.New("token not found in context")
	}
	tokenStr, ok := tokenObj.(string)
	if !ok {
		return nil, errors.New("invalid token type in context")
	}

	// 调用 User RPC 进行注销
	_, err = l.svcCtx.UserRpc.Logout(l.ctx, &userclient.LogoutReq{
		Token: tokenStr,
	})
	if err != nil {
		l.Logger.Errorf("Logout rpc error: %v", err)
		return nil, err
	}

	return &types.LogoutResp{}, nil
}
