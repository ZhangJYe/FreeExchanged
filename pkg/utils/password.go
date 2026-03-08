package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 对密码进行加密
// 这里使用 bcrypt 算法（Blowfish），相比 MD5/SHA256 更安全，能有效抵御彩虹表攻击
// cost 参数决定了计算复杂度，建议在 10-14 之间
func HashPassword(password string) (string, error) {
	// GenerateFromPassword 会自动加盐 (Salt)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// CheckPassword 校验密码是否正确
// password: 用户输入的明文密码
// hashedPassword: 数据库中存储的加密后的密码
func CheckPassword(password string, hashedPassword string) error {
	// CompareHashAndPassword 会自动提取 Salt 并进行运算比较
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
