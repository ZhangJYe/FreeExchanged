package user

import (
	"context"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"freeexchanged/app/user/cmd/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户登录
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	// 调用 User RPC 登录接口
	rpcResp, err := l.svcCtx.UserRpc.Login(l.ctx, &userclient.LoginReq{
		Mobile:   req.Mobile,
		Password: req.Password,
	})
	if err != nil {
		l.Logger.Errorf("Login rpc error: %v", err)
		return nil, err
	}

	return &types.LoginResp{
		Id:       rpcResp.Id,
		Nickname: rpcResp.Nickname,
		Token:    rpcResp.Token,
		ExpireAt: rpcResp.ExpireAt,
	}, nil
}
