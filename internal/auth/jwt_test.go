package auth

import (
	"errors"
	"testing"
	"time"
)

const testSecret = "test-secret-key-that-is-long-enough"

func TestGenerateAndValidateAccessToken(t *testing.T) {
	manager := NewJWTManager(testSecret, time.Minute, time.Hour)

	token, err := manager.GenerateAccessToken(42, "admin")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("user id = %d, want 42", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("role = %q, want admin", claims.Role)
	}
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	manager := NewJWTManager(testSecret, time.Minute, time.Hour)

	token, err := manager.GenerateRefreshToken(7)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.UserID != 7 {
		t.Errorf("user id = %d, want 7", claims.UserID)
	}
	if claims.Role != "" {
		t.Errorf("refresh token role = %q, want empty", claims.Role)
	}
}

func TestValidateExpiredToken(t *testing.T) {
	manager := NewJWTManager(testSecret, -time.Minute, time.Hour)

	token, err := manager.GenerateAccessToken(1, "user")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := manager.ValidateToken(token); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("error = %v, want ErrExpiredToken", err)
	}
}

func TestValidateTokenWithWrongSecret(t *testing.T) {
	signer := NewJWTManager(testSecret, time.Minute, time.Hour)
	verifier := NewJWTManager("another-secret-key-that-is-long", time.Minute, time.Hour)

	token, err := signer.GenerateAccessToken(1, "user")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := verifier.ValidateToken(token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}

func TestValidateGarbageToken(t *testing.T) {
	manager := NewJWTManager(testSecret, time.Minute, time.Hour)

	if _, err := manager.ValidateToken("not-a-token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("error = %v, want ErrInvalidToken", err)
	}
}
