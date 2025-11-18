package helper

import (
	"context"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ContextKey string

const UserIDKey ContextKey = "user_id"

var ReservationTTLMinutesDefault = 5

func GetEnv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

// secret key disimpan di environment variable, misalnya JWT_SECRET
// contoh di docker-compose.yml:
// environment:
//   - JWT_SECRET=mysecretkey
func jwtSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "defaultsecret" // fallback, jangan dipakai di production
	}
	return []byte(secret)
}

// GenerateJWT membuat token JWT baru untuk user tertentu
func GenerateJWT(userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(), // expired 1 hari
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret())
}

// ValidateJWT memverifikasi token JWT dan mengembalikan claims-nya
func ValidateJWT(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return jwtSecret(), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrTokenInvalidClaims
}

// helper function untuk mengambil user id dari context di handler
func GetUserIDFromContext(ctx context.Context) int {
	if v, ok := ctx.Value(UserIDKey).(int); ok {
		return v
	}
	return 0
}
