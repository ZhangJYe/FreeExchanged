package user

import (
	"context"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"freeexchanged/app/user/cmd/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 用户注册
func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterReq) (resp *types.RegisterResp, err error) {
	// 调用 User RPC 注册接口
	rpcResp, err := l.svcCtx.UserRpc.Register(l.ctx, &userclient.RegisterReq{
		Mobile:   req.Mobile,
		Password: req.Password,
		Nickname: req.Nickname,
	})
	if err != nil {
		l.Logger.Errorf("Register rpc error: %v", err)
		return nil, err
	}

	return &types.RegisterResp{
		Id:       rpcResp.Id,
		Token:    rpcResp.Token,
		ExpireAt: rpcResp.ExpireAt,
	}, nil
}
