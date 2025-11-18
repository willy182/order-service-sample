package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"order-service-sample/helper"
)

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Token abc")
	rec := httptest.NewRecorder()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid_token_123")
	rec := httptest.NewRecorder()

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rec.Code)
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	// generate valid jwt
	token, err := helper.GenerateJWT(99) // userID = 99
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	var handlerCalled bool

	handler := AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		// extract user ID from context
		uid := helper.GetUserIDFromContext(r.Context())
		if uid != 99 {
			t.Fatalf("expected user_id 99, got %d", uid)
		}

		w.WriteHeader(200)
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	if !handlerCalled {
		t.Fatalf("next handler was NOT called on valid token")
	}
}
