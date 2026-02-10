package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "user_id"
const UserRoleKey contextKey = "user_role"

func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" {
				next.ServeHTTP(w, r)
				return
			}
			if !strings.HasPrefix(auth, "Bearer ") {
				next.ServeHTTP(w, r)
				return
			}
			tokenStr := strings.TrimPrefix(auth, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				next.ServeHTTP(w, r)
				return
			}
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			sub, _ := claims["sub"].(string)
			role, _ := claims["role"].(string)
			ctx := context.WithValue(r.Context(), UserIDKey, sub)
			ctx = context.WithValue(ctx, UserRoleKey, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(UserIDKey).(string)
	return v
}

func UserRole(ctx context.Context) string {
	v, _ := ctx.Value(UserRoleKey).(string)
	return v
}
