package logic

import (
	"context"

	"freeexchanged/app/user/cmd/rpc/internal/svc"
	"freeexchanged/app/user/cmd/rpc/pb"
	"freeexchanged/app/user/model"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UserInfoLogic {
	return &UserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// UserInfo 获取用户信息
func (l *UserInfoLogic) UserInfo(in *pb.UserInfoReq) (*pb.UserInfoResp, error) {
	// 1. 根据 ID 查询用户
	user, err := l.svcCtx.UserModel.FindOne(l.ctx, in.Id)
	if err != nil {
		if err == model.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		l.Logger.Errorf("Find user error: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// 2. 返回结果
	return &pb.UserInfoResp{
		Id:       user.Id,
		Mobile:   user.Mobile,
		Nickname: user.Nickname,
	}, nil
}
