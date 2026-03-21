package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/VDV001/estimate-pro/backend/internal/shared/errors"
	"github.com/VDV001/estimate-pro/backend/pkg/jwt"
)

type contextKey string

const UserIDKey contextKey = "user_id"

func Auth(jwtService *jwt.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				errors.Unauthorized(w, "missing authorization header")
				return
			}

			token, found := strings.CutPrefix(header, "Bearer ")
			if !found {
				errors.Unauthorized(w, "invalid authorization header format")
				return
			}

			claims, err := jwtService.ValidateAccess(token)
			if err != nil {
				errors.Unauthorized(w, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDKey).(string)
	return id, ok
}
