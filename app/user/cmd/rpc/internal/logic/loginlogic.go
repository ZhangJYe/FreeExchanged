package logic

import (
	"context"
	"time"

	"freeexchanged/app/user/cmd/rpc/internal/svc"
	"freeexchanged/app/user/cmd/rpc/pb"
	"freeexchanged/app/user/model"
	"freeexchanged/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Login 用户登录方法
// 1. 根据手机号查用户
// 2. 比对密码
// 3. 签发 Token
func (l *LoginLogic) Login(in *pb.LoginReq) (*pb.LoginResp, error) {
	// 1. 根据手机号查询用户
	// FindOneByMobile 是 goctl 生成的 model 方法
	user, err := l.svcCtx.UserModel.FindOneByMobile(l.ctx, in.Mobile)
	if err != nil {
		// 如果是 ErrNotFound，说明用户不存在
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		// 其他错误（如 DB 连接失败）
		l.Logger.Errorf("Find user by mobile error: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// 2. 校验密码
	// 传入用户输入的明文密码和数据库里存的 Hash 值
	if err := utils.CheckPassword(in.Password, user.Password); err != nil {
		// 密码不匹配
		return nil, status.Error(codes.Unauthenticated, "invalid password")
	}

	// 3. 登录成功，签发 Token
	duration := time.Duration(l.svcCtx.Config.Identity.AccessExpire) * time.Second
	token, payload, err := l.svcCtx.TokenMaker.CreateToken(user.Id, duration)
	if err != nil {
		l.Logger.Errorf("Create token error: %v", err)
		return nil, status.Error(codes.Internal, "create token failed")
	}

	// 4. 返回完整信息
	return &pb.LoginResp{
		Id:       user.Id,
		Token:    token,
		ExpireAt: payload.ExpiredAt.Unix(),
		Nickname: user.Nickname,
	}, nil
}
