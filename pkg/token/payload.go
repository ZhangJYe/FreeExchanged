package token

import (
	"errors"
	"time"
)

var (
	ErrExpiredToken   = errors.New("token expired")
	ErrNotValidYet    = errors.New("token not valid yet")
	ErrInvalidToken   = errors.New("token invalid")
	ErrInvalidKeySize = errors.New("paseto v2.local key must be 32 bytes (after base64 decode)")
)

// Payload 是 token 里承载的声明（claims）
// PASETO v2.local 会把它作为 JSON 加密进 token 中（机密性 + 完整性）
type Payload struct {
	ID        string    `json:"jti"`           // token id，用于撤销/踢人
	UserID    int64     `json:"uid"`           // 用户 id
	Issuer    string    `json:"iss,omitempty"` // 签发方（可选）
	Audience  string    `json:"aud,omitempty"` // 受众（可选）
	IssuedAt  time.Time `json:"iat"`           // 签发时间
	NotBefore time.Time `json:"nbf,omitempty"` // 生效时间（可选）
	ExpiredAt time.Time `json:"exp"`           // 过期时间
}

// Valid 做基础合法性校验（过期/未生效）
func (p *Payload) Valid(now time.Time) error {
	if now.Before(p.NotBefore) && !p.NotBefore.IsZero() {
		return ErrNotValidYet
	}
	if now.After(p.ExpiredAt) {
		return ErrExpiredToken
	}
	return nil
}
