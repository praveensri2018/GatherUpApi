/* Place: backend/go/api/routes.go */
package api

import (
	"net/http"

	"gatherup/auth"
	"gatherup/repository"
	"gatherup/service"

	"github.com/go-chi/chi/v5"
)

// WireRouter wires handlers and middleware; pass in repo, jwt manager and auth service
func WireRouter(repo *repository.UserRepo, jwtMgr *auth.JWTManager, authSvc *service.AuthService) http.Handler {
	r := chi.NewRouter()

	verifyFn := func(token string) (string, error) {
		claims, err := jwtMgr.Verify(token)
		if err != nil {
			return "", err
		}
		return claims.UserID, nil
	}

	authHandler := NewAuthHandler(authSvc)
	userHandler := NewUserHandler(repo)

	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/refresh", authHandler.Refresh)

	r.Group(func(r chi.Router) {
		r.Use(WithAuth(verifyFn))
		r.Get("/api/me", userHandler.Me)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return r
}
