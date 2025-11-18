package helper

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGetEnv(t *testing.T) {
	// key belum ada → return default
	v := GetEnv("UNKNOWN_ENV_KEY_XYZ", "default123")
	if v != "default123" {
		t.Fatalf("expected default123, got %s", v)
	}

	// set environment → harus return env value
	os.Setenv("MY_ENV_TEST", "hello123")
	v = GetEnv("MY_ENV_TEST", "fallback")
	if v != "hello123" {
		t.Fatalf("expected hello123, got %s", v)
	}
}

func TestGenerateAndValidateJWT(t *testing.T) {
	os.Setenv("JWT_SECRET", "unittestsecret") // set secret untuk test

	token, err := GenerateJWT(42)
	if err != nil {
		t.Fatalf("GenerateJWT returned error: %v", err)
	}
	if token == "" {
		t.Fatalf("expected token string, got empty")
	}

	claims, err := ValidateJWT(token)
	if err != nil {
		t.Fatalf("ValidateJWT returned error: %v", err)
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		t.Fatalf("expected user_id claim in token")
	}
	if int(userID) != 42 {
		t.Fatalf("expected user_id=42, got %d", int(userID))
	}
}

func TestValidateJWT_InvalidToken(t *testing.T) {
	invalidToken := "random.invalid.token.12345"

	_, err := ValidateJWT(invalidToken)
	if err == nil {
		t.Fatalf("expected error for invalid token")
	}
}

func TestValidateJWT_WrongSignature(t *testing.T) {
	// set secret 1 untuk sign
	os.Setenv("JWT_SECRET", "secretA")
	token, _ := GenerateJWT(99)

	// ubah secret → signature mismatch
	os.Setenv("JWT_SECRET", "secretB")

	_, err := ValidateJWT(token)
	if err == nil {
		t.Fatalf("expected signature error but got nil")
	}
}

func TestValidateJWT_Expired(t *testing.T) {
	os.Setenv("JWT_SECRET", "expiretestsecret")

	// create expired token manual
	claims := jwt.MapClaims{
		"user_id": 11,
		"exp":     time.Now().Add(-1 * time.Hour).Unix(), // sudah lewat
		"iat":     time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := token.SignedString(jwtSecret())

	_, err := ValidateJWT(expiredToken)
	if err == nil {
		t.Fatalf("expected error for expired token")
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	ctx := context.Background()

	ctx = context.WithValue(ctx, UserIDKey, 123)

	uid := GetUserIDFromContext(ctx)
	if uid != 123 {
		t.Fatalf("expected 123, got %d", uid)
	}

	// test no-value case
	ctx = context.Background()
	uid = GetUserIDFromContext(ctx)
	if uid != 0 {
		t.Fatalf("expected default 0 when not set, got %d", uid)
	}
}
