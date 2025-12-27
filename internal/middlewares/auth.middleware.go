package middlewares

import (
	"context"
	"mkdir-auth/internal/config"
	"mkdir-auth/internal/models"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			next(w, r)
			return
		}

		claims := &models.UserClaims{}
		token, err := jwt.ParseWithClaims(cookie.Value, claims, func(token *jwt.Token) (interface{}, error) {
			return config.JWTKey, nil
		})

		if err == nil && token.Valid {
			ctx := context.WithValue(r.Context(), "user", claims)
			next(w, r.WithContext(ctx))
			return
		}

		next(w, r)
	}
}
