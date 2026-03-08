package user

import (
	"context"
	"encoding/json"

	"freeexchanged/app/gateway/internal/svc"
	"freeexchanged/app/gateway/internal/types"
	"freeexchanged/app/user/cmd/rpc/userclient"

	"github.com/zeromicro/go-zero/core/logx"
)

type UserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserInfoLogic {
	return &UserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// UserInfo 网关层逻辑：调用 RPC 获取用户信息
func (l *UserInfoLogic) UserInfo(req *types.UserInfoReq) (resp *types.UserInfoResp, err error) {
	// 1. 从 context 获取 userId (由 PasetoMiddleware 注入)
	userIdval := l.ctx.Value("userId")
	if userIdval == nil {
		// 理论上不会走到这里，因为中间件已经拦截了
		return nil, nil // 或者返回未授权错误
	}

	// 注意：从 context 取出的 value 是 any 类型，需要断言
	// 我们的 PasetoMiddleware 注入的是 int64 (JSON 数字默认是 float64，但我们的 Payload 定义明确是 int64)
	var userId int64
	switch v := userIdval.(type) {
	case int64:
		userId = v
	case json.Number: // 如果是 json 解析出来的，可能是 json.Number
		userId, _ = v.Int64()
	case float64:
		userId = int64(v)
	default:
		l.Logger.Errorf("Invalid userId type in context: %T", v)
		// 兜底尝试强转
		userId = userIdval.(int64)
	}

	// 2. 调用 User RPC
	// 需要在 Gateway 的 ServiceContext 里添加 UserRpc 客户端
	// 这里假设 svcCtx.UserRpc 已经有了 (下面会去添加)
	rpcResp, err := l.svcCtx.UserRpc.UserInfo(l.ctx, &userclient.UserInfoReq{
		Id: userId,
	})
	if err != nil {
		l.Logger.Errorf("Call UserRpc UserInfo failed: %v", err)
		return nil, err
	}

	// 3. 转换并返回
	return &types.UserInfoResp{
		Id:       rpcResp.Id,
		Mobile:   rpcResp.Mobile,
		Nickname: rpcResp.Nickname,
	}, nil
}
