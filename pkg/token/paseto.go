package token

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/o1egl/paseto"
)

// Maker 抽象接口：方便你以后换 JWT/PASETO v4 或者做单测
type Maker interface {
	CreateToken(userID int64, duration time.Duration) (string, *Payload, error)
	VerifyToken(token string) (*Payload, error)
}

// PasetoMaker 使用 PASETO v2.local（对称加密）
// key 必须是 32 bytes（强制要求）
type PasetoMaker struct {
	paseto   *paseto.V2
	key      []byte
	issuer   string
	audience string
}

// NewPasetoMakerFromBase64Key 推荐：配置里存 base64 字符串（更容易管理）
// base64 解码后必须是 32 bytes
func NewPasetoMakerFromBase64Key(keyBase64 string, issuer string, audience string) (*PasetoMaker, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		// 兼容 URL-safe base64（有些人喜欢用这个）
		key, err = base64.RawURLEncoding.DecodeString(keyBase64)
		if err != nil {
			return nil, errors.New("invalid base64 key")
		}
	}

	if len(key) != 32 {
		return nil, ErrInvalidKeySize
	}

	return &PasetoMaker{
		paseto:   paseto.NewV2(),
		key:      key,
		issuer:   issuer,
		audience: audience,
	}, nil
}

func (m *PasetoMaker) CreateToken(userID int64, duration time.Duration) (string, *Payload, error) {
	now := time.Now().UTC()

	payload := &Payload{
		ID:        uuid.NewString(),
		UserID:    userID,
		Issuer:    m.issuer,
		Audience:  m.audience,
		IssuedAt:  now,
		NotBefore: now, // 立即生效
		ExpiredAt: now.Add(duration),
	}

	token, err := m.paseto.Encrypt(m.key, payload, nil)
	if err != nil {
		return "", nil, err
	}
	return token, payload, nil
}

func (m *PasetoMaker) VerifyToken(token string) (*Payload, error) {
	var payload Payload
	err := m.paseto.Decrypt(token, m.key, &payload, nil)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if err := payload.Valid(time.Now().UTC()); err != nil {
		return nil, err
	}

	// 可选：做 iss/aud 校验（你如果填了 issuer/audience）
	if m.issuer != "" && payload.Issuer != m.issuer {
		return nil, ErrInvalidToken
	}
	if m.audience != "" && payload.Audience != m.audience {
		return nil, ErrInvalidToken
	}

	return &payload, nil
}

// GenerateSymmetricKeyBase64 生成一把安全的 32 bytes key（base64 形式）
// 你可以把输出贴到 yaml 的 AccessSecret 里
func GenerateSymmetricKeyBase64() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
