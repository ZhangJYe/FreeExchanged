package logic

import (
	"context"
	"strings"
	"time"

	"freeexchanged/app/user/cmd/rpc/internal/svc"
	"freeexchanged/app/user/cmd/rpc/pb"
	"freeexchanged/app/user/model"
	"freeexchanged/pkg/utils"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// Register 用户注册方法
// 1. 检查手机号是否已存在
// 2. 加密密码
// 3. 插入数据库
// 4. 生成 Token
func (l *RegisterLogic) Register(in *pb.RegisterReq) (*pb.RegisterResp, error) {
	// 1. 基础参数校验 (手机号格式等建议在网关层做，这里只做业务逻辑校验)
	mobile := strings.TrimSpace(in.Mobile)
	if mobile == "" {
		return nil, status.Error(codes.InvalidArgument, "mobile is required")
	}
	if len(in.Password) < 6 {
		return nil, status.Error(codes.InvalidArgument, "password too short")
	}

	// 2. 也是最重要的安全性步骤：密码加密
	// 绝对不要明文存密码！
	hashedPassword, err := utils.HashPassword(in.Password)
	if err != nil {
		l.Logger.Errorf("Hash password failed: %v", err)
		return nil, status.Error(codes.Internal, "internal server error")
	}

	// 3. 构建用户对象并插入数据库
	newUser := &model.User{
		Username: mobile,
		Mobile:   mobile,
		Password: hashedPassword, // 存加密后的
		Nickname: strings.TrimSpace(in.Nickname),
	}

	// 调用 model 层的 Insert 方法
	// 注意：Insert 可能会因为唯一索引冲突而报错
	res, err := l.svcCtx.UserModel.Insert(l.ctx, newUser)
	if err != nil {
		// 判断是否是唯一键冲突 (Error 1062)
		if strings.Contains(err.Error(), "Duplicate entry") {
			return nil, status.Error(codes.AlreadyExists, "mobile number already exists")
		}
		l.Logger.Errorf("Insert user failed: %v", err)
		return nil, status.Error(codes.Internal, "database error")
	}

	// 4. 获取新插入用户的 ID
	userId, err := res.LastInsertId()
	if err != nil {
		l.Logger.Errorf("Get last insert id failed: %v", err)
		return nil, status.Error(codes.Internal, "database error")
	}

	// 5. 注册成功后直接“自动登录”，生成 Token 返回给前端
	// Token 有效期从配置中读取 (Config.Auth.AccessExpire)
	duration := time.Duration(l.svcCtx.Config.Identity.AccessExpire) * time.Second
	token, payload, err := l.svcCtx.TokenMaker.CreateToken(userId, duration)
	if err != nil {
		l.Logger.Errorf("Create token failed: %v", err)
		return nil, status.Error(codes.Internal, "failed to create token")
	}

	// 6. 构造返回结果
	return &pb.RegisterResp{
		Id:       userId,
		Token:    token,
		ExpireAt: payload.ExpiredAt.Unix(), // 返回 Unix 时间戳给前端
	}, nil
}
