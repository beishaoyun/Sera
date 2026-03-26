package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestNewJWTManager(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)
	if manager == nil {
		t.Fatal("Expected JWT manager to be created")
	}
	if string(manager.secretKey) != "test-secret" {
		t.Errorf("Expected secret key 'test-secret', got '%s'", string(manager.secretKey))
	}
}

func TestGenerateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)
	userID := uuid.New()

	token, err := manager.GenerateToken(userID, "test@example.com", "free", "access", time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if token == "" {
		t.Fatal("Expected token to be generated")
	}
}

func TestVerifyToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)
	userID := uuid.New()

	accessToken, _, err := manager.GenerateTokenPair(userID, "test@example.com", "free")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := manager.VerifyToken(accessToken)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", claims.Email)
	}
	if claims.Tier != "free" {
		t.Errorf("Expected tier 'free', got '%s'", claims.Tier)
	}
}

func TestVerifyTokenInvalid(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)

	_, err := manager.VerifyToken("invalid-token")
	if err == nil {
		t.Fatal("Expected error for invalid token")
	}
}

func TestVerifyTokenWrongSecret(t *testing.T) {
	manager1 := NewJWTManager("secret1", time.Hour, time.Hour*24)
	manager2 := NewJWTManager("secret2", time.Hour, time.Hour*24)
	userID := uuid.New()

	token, _, _ := manager1.GenerateTokenPair(userID, "test@example.com", "free")
	_, err := manager2.VerifyToken(token)
	if err == nil {
		t.Fatal("Expected error for token signed with different secret")
	}
}

func TestRefreshAccessToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)
	userID := uuid.New()

	_, refreshToken, err := manager.GenerateTokenPair(userID, "test@example.com", "free")
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	newAccessToken, err := manager.RefreshAccessToken(refreshToken)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	claims, err := manager.VerifyToken(newAccessToken)
	if err != nil {
		t.Fatalf("Failed to verify new access token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.TokenType != "access" {
		t.Errorf("Expected token type 'access', got '%s'", claims.TokenType)
	}
}

func TestRefreshAccessTokenWithInvalidToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)

	_, err := manager.RefreshAccessToken("invalid-refresh-token")
	if err == nil {
		t.Fatal("Expected error for invalid refresh token")
	}
}

func TestRefreshAccessTokenWithAccessToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)
	userID := uuid.New()

	accessToken, _, _ := manager.GenerateTokenPair(userID, "test@example.com", "free")

	_, err := manager.RefreshAccessToken(accessToken)
	if err == nil {
		t.Fatal("Expected error when using access token as refresh token")
	}
}

func TestTokenExpiration(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Millisecond*100, time.Hour*24)
	userID := uuid.New()

	token, _, _ := manager.GenerateTokenPair(userID, "test@example.com", "free")

	// Wait for token to expire
	time.Sleep(time.Millisecond * 150)

	_, err := manager.VerifyToken(token)
	if err == nil {
		t.Fatal("Expected error for expired token")
	}
}

func TestClaimsParsing(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour, time.Hour*24)
	userID := uuid.New()

	tokenString, _, _ := manager.GenerateTokenPair(userID, "test@example.com", "pro")

	// Manually parse token to verify claims structure
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return manager.secretKey, nil
	})

	if err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		t.Fatal("Failed to parse claims")
	}

	if claims.Issuer != "servermind" {
		t.Errorf("Expected issuer 'servermind', got '%s'", claims.Issuer)
	}
}
