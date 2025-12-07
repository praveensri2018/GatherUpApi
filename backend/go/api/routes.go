package api

import (
	"net/http"

	"gatherup/auth"
	"gatherup/repository"
	"gatherup/service"

	"github.com/go-chi/chi/v5"
)

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

	// Auth endpoints - rate limited
	// Rate: 0.2 tokens/sec (1 per 5s), burst 5
	r.Post("/auth/register", RateLimitAuth(0.2, 5)(http.HandlerFunc(authHandler.Register)).ServeHTTP)
	r.Post("/auth/login", RateLimitAuth(0.2, 5)(http.HandlerFunc(authHandler.Login)).ServeHTTP)
	r.Post("/auth/refresh", RateLimitAuth(0.5, 10)(http.HandlerFunc(authHandler.Refresh)).ServeHTTP)
	r.Post("/auth/revoke", RateLimitAuth(0.5, 10)(http.HandlerFunc(authHandler.Revoke)).ServeHTTP)

	// NEW forgot-password routes
	r.Post("/auth/forgot/send-otp", RateLimitAuth(0.1, 5)(http.HandlerFunc(authHandler.SendForgotOTP)).ServeHTTP)
	r.Post("/auth/forgot/verify-otp", RateLimitAuth(0.2, 10)(http.HandlerFunc(authHandler.VerifyForgotOTP)).ServeHTTP)
	r.Post("/auth/forgot/reset", RateLimitAuth(0.2, 10)(http.HandlerFunc(authHandler.ResetPassword)).ServeHTTP)

	// Protected routes
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
