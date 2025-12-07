package api

import (
	"context"
	"net/http"
	"strings"
)

type ctxKey string

const ctxUserIDKey ctxKey = "user_id"

// clientIP extracts client IP, preferring X-Forwarded-For header.
func clientIP(r *http.Request) string {
	if xf := r.Header.Get("X-Forwarded-For"); xf != "" {
		parts := strings.Split(xf, ",")
		return strings.TrimSpace(parts[0])
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		return host[:idx]
	}
	return host
}

func WithAuth(verify func(token string) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authH := r.Header.Get("Authorization")
			if authH == "" {
				ErrorJSON(w, http.StatusUnauthorized, "authorization required")
				return
			}
			parts := strings.SplitN(authH, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				ErrorJSON(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}
			token := parts[1]
			userID, err := verify(token)
			if err != nil {
				ErrorJSON(w, http.StatusUnauthorized, "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserIDKey, userID)
			// attach client IP as context if needed by handlers
			ctx = context.WithValue(ctx, ctxKey("client_ip"), clientIP(r))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func FromContextUserID(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxUserIDKey)
	if v == nil {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
}
