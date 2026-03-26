package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// JWTManager JWT 管理器
type JWTManager struct {
	secretKey     []byte
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

// Claims JWT Claims
type Claims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Tier      string    `json:"tier"`
	TokenType string    `json:"token_type"` // access, refresh
	jwt.RegisteredClaims
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secretKey string, tokenExpiry, refreshExpiry time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateTokenPair 生成访问令牌和刷新令牌
func (m *JWTManager) GenerateTokenPair(userID uuid.UUID, email, tier string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateToken(userID, email, tier, "access", m.tokenExpiry)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = m.GenerateToken(userID, email, tier, "refresh", m.refreshExpiry)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

// GenerateToken 生成 JWT 令牌
func (m *JWTManager) GenerateToken(userID uuid.UUID, email, tier, tokenType string, expiry time.Duration) (string, error) {
	claims := Claims{
		UserID:    userID,
		Email:     email,
		Tier:      tier,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "servermind",
			Subject:   userID.String(),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// VerifyToken 验证并解析 JWT 令牌
func (m *JWTManager) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// 检查过期
	if claims.ExpiresAt.Before(time.Now()) {
		return nil, ErrExpiredToken
	}

	return claims, nil
}

// RefreshAccessToken 使用刷新令牌获取新的访问令牌
func (m *JWTManager) RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := m.VerifyToken(refreshToken)
	if err != nil {
		return "", err
	}

	if claims.TokenType != "refresh" {
		return "", ErrInvalidToken
	}

	// 生成新的访问令牌
	return m.GenerateToken(claims.UserID, claims.Email, claims.Tier, "access", m.tokenExpiry)
}
