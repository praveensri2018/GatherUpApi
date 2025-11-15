/* Place: backend/go/api/middleware.go */
package api

import (
	"context"
	"net/http"
	"strings"
)

// ctx key for user id
type ctxKey string

const ctxUserIDKey ctxKey = "user_id"

// WithAuth returns middleware that uses verify function to validate token and set user id in context
func WithAuth(verify func(token string) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authH := r.Header.Get("Authorization")
			if authH == "" {
				http.Error(w, "authorization required", http.StatusUnauthorized)
				return
			}
			parts := strings.SplitN(authH, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				http.Error(w, "invalid authorization header", http.StatusUnauthorized)
				return
			}
			token := parts[1]
			userID, err := verify(token)
			if err != nil {
				http.Error(w, "invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// FromContextUserID extracts user id from request context
func FromContextUserID(ctx context.Context) (string, bool) {
	v := ctx.Value(ctxUserIDKey)
	if v == nil {
		return "", false
	}
	id, ok := v.(string)
	return id, ok
}
