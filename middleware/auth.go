package middleware

import (
	"context"
	"net/http"
	"strings"

	"order-service-sample/helper"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			helper.WriteErrorJSON(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			helper.WriteErrorJSON(w, http.StatusUnauthorized, "invalid Authorization format (use Bearer token)")
			return
		}

		claims, err := helper.ValidateJWT(tokenStr)
		if err != nil {
			helper.WriteErrorJSON(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		userID, ok := claims["user_id"].(float64)
		if !ok {
			helper.WriteErrorJSON(w, http.StatusUnauthorized, "invalid token payload")
			return
		}

		ctx := context.WithValue(r.Context(), helper.UserIDKey, int(userID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
